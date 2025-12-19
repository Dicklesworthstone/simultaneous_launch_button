// Package output implements consistent JSON output formatting for SLB.
// All JSON output uses snake_case keys as specified in the plan.
package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"go.yaml.in/yaml/v3"
)

// Format represents the output format.
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
	FormatYAML Format = "yaml"
)

// Writer handles formatted output.
type Writer struct {
	format Format
	out    io.Writer
	errOut io.Writer
}

// Option configures the Writer.
type Option func(*Writer)

// WithOutput sets the standard output writer.
func WithOutput(w io.Writer) Option {
	return func(wr *Writer) {
		wr.out = w
	}
}

// WithErrorOutput sets the error output writer.
func WithErrorOutput(w io.Writer) Option {
	return func(wr *Writer) {
		wr.errOut = w
	}
}

// New creates a new output writer.
func New(format Format, opts ...Option) *Writer {
	w := &Writer{
		format: format,
		out:    os.Stdout,
		errOut: os.Stderr,
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// Write outputs data in the configured format.
func (w *Writer) Write(data any) error {
	switch w.format {
	case FormatJSON:
		enc := json.NewEncoder(w.out)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	case FormatYAML:
		normalized, err := normalizeForYAML(data)
		if err != nil {
			return err
		}
		b, err := yaml.Marshal(normalized)
		if err != nil {
			return err
		}
		if len(b) == 0 || b[len(b)-1] != '\n' {
			b = append(b, '\n')
		}
		_, err = w.out.Write(b)
		return err
	case FormatText:
		// Human-friendly output goes to stderr to keep stdout clean for piping.
		_, err := fmt.Fprintf(w.errOut, "%v\n", data)
		return err
	default:
		return fmt.Errorf("unsupported format: %s", w.format)
	}
}

// WriteNDJSON outputs data as NDJSON when in JSON mode (one JSON per line).
func (w *Writer) WriteNDJSON(data any) error {
	switch w.format {
	case FormatJSON:
		enc := json.NewEncoder(w.out)
		return enc.Encode(data)
	case FormatText:
		_, err := fmt.Fprintf(w.errOut, "%v\n", data)
		return err
	default:
		return fmt.Errorf("unsupported format: %s", w.format)
	}
}

// Success outputs a success message.
func (w *Writer) Success(msg string) {
	if w.format == FormatJSON || w.format == FormatYAML {
		_ = w.Write(map[string]any{"status": "success", "message": msg})
	} else {
		fmt.Fprintf(w.errOut, "✓ %s\n", msg)
	}
}

// Error outputs an error message.
func (w *Writer) Error(err error) {
	if w.format == FormatJSON {
		_ = OutputJSONError(err, 1)
	} else if w.format == FormatYAML {
		_ = OutputYAML(ErrorPayload{
			Error:   "error",
			Message: err.Error(),
			Details: map[string]any{"code": 1},
		})
	} else {
		fmt.Fprintf(w.errOut, "✗ %s\n", err.Error())
	}
}

func normalizeForYAML(v any) (any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	var normalized any
	if err := dec.Decode(&normalized); err != nil {
		return nil, err
	}
	return normalized, nil
}

// OutputYAML writes YAML to stdout, preserving JSON tags/field names by converting via JSON first.
func OutputYAML(v any) error {
	normalized, err := normalizeForYAML(v)
	if err != nil {
		return err
	}
	b, err := yaml.Marshal(normalized)
	if err != nil {
		return err
	}
	if len(b) == 0 || b[len(b)-1] != '\n' {
		b = append(b, '\n')
	}
	_, err = os.Stdout.Write(b)
	return err
}
