package har

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// YAMLFormat constant.
const FormatYAML ConvertFormat = "yaml"

// ToYAML converts a Har object to a YAML string.
//
// This method converts HAR data to YAML for easier reading and editing.
// Note: this implementation does not depend on an external YAML library; it uses the built-in JSON-to-YAML conversion.
func (h *Har) ToYAML() (string, error) {
	if h == nil {
		return "", NewInvalidFormatError("HAR object is nil")
	}

	// Convert to JSON first.
	jsonData, err := h.ToJSON(true)
	if err != nil {
		return "", err
	}

	// Convert JSON to YAML.
	return jsonToYAML(jsonData), nil
}

// SaveAsYAML saves a Har object as a YAML file.
func (h *Har) SaveAsYAML(filePath string) error {
	yamlData, err := h.ToYAML()
	if err != nil {
		return err
	}
	return writeToFile(filePath, []byte(yamlData))
}

// jsonToYAML is a simple JSON-to-YAML converter.
// Provides basic YAML output without external libraries.
func jsonToYAML(jsonData []byte) string {
	var data interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return string(jsonData)
	}
	return valueToYAML(data, 0)
}

// valueToYAML recursively converts a value to YAML.
func valueToYAML(v interface{}, indent int) string {
	var sb strings.Builder
	prefix := strings.Repeat("  ", indent)

	switch val := v.(type) {
	case map[string]interface{}:
		first := true
		for k, v := range val {
			if !first {
				sb.WriteString("\n")
			}
			first = false

			switch child := v.(type) {
			case map[string]interface{}:
				sb.WriteString(fmt.Sprintf("%s%s:\n", prefix, k))
				sb.WriteString(valueToYAML(child, indent+1))
			case []interface{}:
				sb.WriteString(fmt.Sprintf("%s%s:\n", prefix, k))
				sb.WriteString(arrayToYAML(child, indent+1))
			case nil:
				sb.WriteString(fmt.Sprintf("%s%s: null\n", prefix, k))
			case string:
				if strings.ContainsAny(child, ":{}[]&*?|>-!%@`\"'\n") || child == "" {
					sb.WriteString(fmt.Sprintf("%s%s: \"%s\"\n", prefix, k, escapeYAMLString(child)))
				} else {
					sb.WriteString(fmt.Sprintf("%s%s: %s\n", prefix, k, child))
				}
			case float64:
				if child == float64(int64(child)) {
					sb.WriteString(fmt.Sprintf("%s%s: %d\n", prefix, k, int64(child)))
				} else {
					sb.WriteString(fmt.Sprintf("%s%s: %g\n", prefix, k, child))
				}
			case bool:
				sb.WriteString(fmt.Sprintf("%s%s: %v\n", prefix, k, child))
			default:
				sb.WriteString(fmt.Sprintf("%s%s: %v\n", prefix, k, child))
			}
		}
	case []interface{}:
		sb.WriteString(arrayToYAML(val, indent))
	case nil:
		sb.WriteString(fmt.Sprintf("%snull\n", prefix))
	case string:
		sb.WriteString(fmt.Sprintf("%s%s\n", prefix, val))
	case float64:
		if val == float64(int64(val)) {
			sb.WriteString(fmt.Sprintf("%s%d\n", prefix, int64(val)))
		} else {
			sb.WriteString(fmt.Sprintf("%s%g\n", prefix, val))
		}
	case bool:
		sb.WriteString(fmt.Sprintf("%s%v\n", prefix, val))
	default:
		sb.WriteString(fmt.Sprintf("%s%v\n", prefix, val))
	}

	return sb.String()
}

// arrayToYAML converts an array to YAML.
func arrayToYAML(arr []interface{}, indent int) string {
	var sb strings.Builder
	prefix := strings.Repeat("  ", indent)

	for _, item := range arr {
		switch val := item.(type) {
		case map[string]interface{}:
			sb.WriteString(fmt.Sprintf("%s-\n", prefix))
			sb.WriteString(valueToYAML(val, indent+1))
		case string:
			if strings.ContainsAny(val, ":{}[]&*?|>-!%@`\"'\n") || val == "" {
				sb.WriteString(fmt.Sprintf("%s- \"%s\"\n", prefix, escapeYAMLString(val)))
			} else {
				sb.WriteString(fmt.Sprintf("%s- %s\n", prefix, val))
			}
		case float64:
			if val == float64(int64(val)) {
				sb.WriteString(fmt.Sprintf("%s- %d\n", prefix, int64(val)))
			} else {
				sb.WriteString(fmt.Sprintf("%s- %g\n", prefix, val))
			}
		case bool:
			sb.WriteString(fmt.Sprintf("%s- %v\n", prefix, val))
		case nil:
			sb.WriteString(fmt.Sprintf("%s- null\n", prefix))
		default:
			sb.WriteString(fmt.Sprintf("%s- %v\n", prefix, val))
		}
	}

	return sb.String()
}

// writeToFile writes data to a file.
func writeToFile(filePath string, data []byte) error {
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return NewFileSystemError(fmt.Sprintf("Unable to write file '%s'", filePath), err)
	}

	return nil
}

// escapeYAMLString escapes special characters in a YAML string.
func escapeYAMLString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}

// WriteToWriter writes HAR data to the specified Writer.
// (Defined in builder.go; this variant accepts a ConvertFormat parameter.)

// ConvertTo converts HAR data to the specified format and writes it to a Writer.
func (h *Har) ConvertTo(format ConvertFormat, w io.Writer, options ConvertOptions) error {
	if h == nil {
		return NewInvalidFormatError("HAR object is nil")
	}
	if isNilWriter(w) {
		return NewInvalidFormatError("writer is nil")
	}

	var content string
	var err error

	switch format {
	case FormatYAML:
		content, err = h.ToYAML()
	case FormatCSV, FormatMarkdown, FormatHTML, FormatText:
		content, err = h.Convert(format, options)
	default:
		// Output JSON by default.
		var data []byte
		data, err = h.ToJSON(true)
		if err == nil {
			return writeAllToWriter(w, data, "failed to write converted HAR data")
		}
	}

	if err != nil {
		return err
	}

	return writeAllToWriter(w, []byte(content), "failed to write converted HAR data")
}
