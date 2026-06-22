package service

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/aloks98/waygates/backend/internal/caddy/logtail"
)

// CaddyLogsService provides access to Caddy runtime and access log files.
type CaddyLogsService struct {
	logPath string
}

// NewCaddyLogsService creates a new CaddyLogsService using the given log directory.
func NewCaddyLogsService(logPath string) *CaddyLogsService {
	return &CaddyLogsService{logPath: logPath}
}

// FilePath maps a log source name to its absolute file path.
// Valid sources are "runtime" and "access".
func (s *CaddyLogsService) FilePath(source string) (string, error) {
	switch source {
	case "runtime":
		return filepath.Join(s.logPath, "runtime.log"), nil
	case "access":
		return filepath.Join(s.logPath, "access.log"), nil
	default:
		return "", fmt.Errorf("unknown log source %q: must be \"runtime\" or \"access\"", source)
	}
}

// Snapshot returns up to limit log lines from the given source as raw JSON messages.
// Each line in the Caddy log file is already a JSON object. Lines that are not
// valid JSON are returned as JSON-encoded strings so the caller always receives
// well-formed json.RawMessage values.
func (s *CaddyLogsService) Snapshot(source string, limit int) ([]json.RawMessage, error) {
	path, err := s.FilePath(source)
	if err != nil {
		return nil, err
	}
	rawLines, err := logtail.LastLines(path, limit)
	if err != nil {
		return nil, fmt.Errorf("reading log snapshot: %w", err)
	}
	msgs := make([]json.RawMessage, 0, len(rawLines))
	for _, line := range rawLines {
		if len(line) == 0 {
			continue
		}
		if json.Valid(line) {
			msgs = append(msgs, json.RawMessage(line))
		} else {
			// Fallback: encode the raw bytes as a JSON string so the envelope stays valid.
			encoded, encErr := json.Marshal(string(line))
			if encErr != nil {
				continue
			}
			msgs = append(msgs, json.RawMessage(encoded))
		}
	}
	return msgs, nil
}

// Tailer returns a new logtail.Tailer for the given source.
// Used by the SSE stream endpoint (Task 4).
func (s *CaddyLogsService) Tailer(source string) (*logtail.Tailer, error) {
	path, err := s.FilePath(source)
	if err != nil {
		return nil, err
	}
	return logtail.NewTailer(path), nil
}
