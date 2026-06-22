package logtail

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLastLines_ReturnsLastN verifies that LastLines returns only the last n lines.
func TestLastLines_ReturnsLastN(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	content := "line1\nline2\nline3\nline4\nline5\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	lines, err := LastLines(path, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if string(lines[0]) != "line3" || string(lines[1]) != "line4" || string(lines[2]) != "line5" {
		t.Fatalf("unexpected lines: %v", lines)
	}
}

// TestLastLines_FewerThanN verifies that LastLines returns all lines if fewer than n exist.
func TestLastLines_FewerThanN(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	content := "only\ntwo\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	lines, err := LastLines(path, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
}

// TestLastLines_MissingFile verifies that LastLines returns (nil, nil) for a missing file.
func TestLastLines_MissingFile(t *testing.T) {
	lines, err := LastLines("/nonexistent/path/does/not/exist.log", 10)
	if err != nil {
		t.Fatalf("expected nil error for missing file, got: %v", err)
	}
	if lines != nil {
		t.Fatalf("expected nil lines for missing file, got: %v", lines)
	}
}

// TestStream_BackfillThenPickUp verifies that Stream backfills existing lines then
// picks up newly appended lines.
func TestStream_BackfillThenPickUp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	// Write initial content
	if err := os.WriteFile(path, []byte("existing1\nexisting2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tailer := &Tailer{path: path, interval: 50 * time.Millisecond}
	out := make(chan []byte, 20)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go func() {
		_ = tailer.Stream(ctx, 2, out)
	}()

	// Collect backfill
	received := collectN(t, out, 2, time.Second)
	if string(received[0]) != "existing1" || string(received[1]) != "existing2" {
		t.Fatalf("unexpected backfill lines: %v", received)
	}

	// Append a new line
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("newline\n"); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	// Should receive the new line
	next := collectN(t, out, 1, 2*time.Second)
	if string(next[0]) != "newline" {
		t.Fatalf("expected 'newline', got %q", string(next[0]))
	}
}

// TestStream_Rotation verifies that Stream handles log rotation (truncate/replace)
// and continues receiving new lines after rotation.
func TestStream_Rotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	// Write initial content
	if err := os.WriteFile(path, []byte("before-rotation\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tailer := &Tailer{path: path, interval: 50 * time.Millisecond}
	out := make(chan []byte, 20)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go func() {
		_ = tailer.Stream(ctx, 0, out)
	}()

	// Give the tailer time to record initial offset
	time.Sleep(200 * time.Millisecond)

	// Simulate rotation: truncate (write smaller content, simulating size shrink)
	if err := os.WriteFile(path, []byte("after-rotation\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Should receive the post-rotation line
	next := collectN(t, out, 1, 2*time.Second)
	if string(next[0]) != "after-rotation" {
		t.Fatalf("expected 'after-rotation', got %q", string(next[0]))
	}
}

// collectN reads exactly n lines from out within deadline, or fails the test.
func collectN(t *testing.T, out <-chan []byte, n int, deadline time.Duration) [][]byte {
	t.Helper()
	timer := time.NewTimer(deadline)
	defer timer.Stop()
	var result [][]byte
	for len(result) < n {
		select {
		case line := <-out:
			result = append(result, line)
		case <-timer.C:
			t.Fatalf("timed out after %v waiting for %d lines (got %d): %v", deadline, n, len(result), result)
		}
	}
	return result
}
