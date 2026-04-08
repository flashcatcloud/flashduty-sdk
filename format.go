package flashduty

import (
	"encoding/json"
	"strings"

	toon "github.com/toon-format/toon-go"
)

// OutputFormat defines the serialization format for tool results
type OutputFormat string

const (
	// OutputFormatJSON uses standard JSON serialization (default)
	OutputFormatJSON OutputFormat = "json"
	// OutputFormatTOON uses Token-Oriented Object Notation for reduced token usage
	OutputFormatTOON OutputFormat = "toon"
)

// ParseOutputFormat converts a string to OutputFormat, defaulting to JSON
func ParseOutputFormat(s string) OutputFormat {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "toon":
		return OutputFormatTOON
	default:
		return OutputFormatJSON
	}
}

// String returns the string representation of OutputFormat
func (f OutputFormat) String() string {
	return string(f)
}

// Marshal serializes the given value using the specified format
func Marshal(v any, format OutputFormat) ([]byte, error) {
	switch format {
	case OutputFormatTOON:
		return toon.Marshal(v)
	default:
		return json.Marshal(v)
	}
}
