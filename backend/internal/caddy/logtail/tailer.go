package logtail

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"time"
)

// LastLines returns up to n trailing lines of the file (each without the newline).
func LastLines(path string, n int) ([][]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no log yet → empty, not an error
		}
		return nil, err
	}
	defer f.Close()
	var lines [][]byte
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := append([]byte(nil), sc.Bytes()...)
		lines = append(lines, line)
		if len(lines) > n {
			lines = lines[1:]
		}
	}
	return lines, sc.Err()
}

type Tailer struct {
	path     string
	interval time.Duration
}

func NewTailer(path string) *Tailer { return &Tailer{path: path, interval: 500 * time.Millisecond} }

// Stream backfills the last `backfill` lines then streams appended lines until ctx is done.
// Handles rotation: if the file shrinks or its identity changes, it reopens from the start.
func (t *Tailer) Stream(ctx context.Context, backfill int, out chan<- []byte) error {
	if backfill > 0 {
		lines, err := LastLines(t.path, backfill)
		if err != nil {
			return err
		}
		for _, l := range lines {
			select {
			case out <- l:
			case <-ctx.Done():
				return nil
			}
		}
	}
	var offset int64
	if fi, err := os.Stat(t.path); err == nil {
		offset = fi.Size()
	}
	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()
	var carry []byte
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			fi, err := os.Stat(t.path)
			if err != nil {
				continue // file may be mid-rotation
			}
			if fi.Size() < offset {
				offset = 0 // rotated/truncated → reread from start
				carry = nil
			}
			if fi.Size() == offset {
				continue
			}
			f, err := os.Open(t.path)
			if err != nil {
				continue
			}
			if _, err := f.Seek(offset, 0); err != nil {
				_ = f.Close()
				continue
			}
			buf := make([]byte, fi.Size()-offset)
			nRead, _ := f.Read(buf)
			_ = f.Close()
			offset += int64(nRead)
			data := make([]byte, 0, len(carry)+nRead)
			data = append(data, carry...)
			data = append(data, buf[:nRead]...)
			for {
				i := bytes.IndexByte(data, '\n')
				if i < 0 {
					break
				}
				line := append([]byte(nil), data[:i]...)
				data = data[i+1:]
				select {
				case out <- line:
				case <-ctx.Done():
					return nil
				}
			}
			carry = append([]byte(nil), data...)
		}
	}
}
