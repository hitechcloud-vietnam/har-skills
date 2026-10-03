package har

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"
)

// ConvertFormat identifies a supported conversion format.
type ConvertFormat string

const (
	FormatCSV      ConvertFormat = "csv"
	FormatMarkdown ConvertFormat = "markdown"
	FormatHTML     ConvertFormat = "html"
	FormatText     ConvertFormat = "text"
)

// ConvertOptions configures conversion.
type ConvertOptions struct {
	// Fields to include.
	IncludeURL         bool
	IncludeMethod      bool
	IncludeStatus      bool
	IncludeContentType bool
	IncludeSize        bool
	IncludeTime        bool
	IncludeTimings     bool
	IncludeHeaders     bool
	IncludeDateTime    bool
	IncludePostData    bool // Include POST data.
	IncludeQueryString bool // Include query parameters.

	// Custom headers (optional; defaults are used when omitted).
	Headers []string

	// Filter options (optional; data is filtered before conversion).
	Filter *FilterOptions
}

// DefaultConvertOptions returns the default conversion options.
func DefaultConvertOptions() ConvertOptions {
	return ConvertOptions{
		IncludeURL:         true,
		IncludeMethod:      true,
		IncludeStatus:      true,
		IncludeContentType: true,
		IncludeSize:        true,
		IncludeTime:        true,
		IncludeTimings:     false,
		IncludeHeaders:     false,
		IncludeDateTime:    true,
	}
}

// Convert converts a HAR file to the specified format.
func (h *Har) Convert(format ConvertFormat, options ConvertOptions) (string, error) {
	if h == nil {
		return "", NewInvalidFormatError("HAR object is nil")
	}

	// Apply filters first, if any.
	entries := h.Log.Entries
	if options.Filter != nil {
		filterResult := h.Filter(*options.Filter)
		entries = filterResult.Entries
	}

	switch format {
	case FormatCSV:
		return convertToCSV(entries, options)
	case FormatMarkdown:
		return convertToMarkdown(entries, options)
	case FormatHTML:
		return convertToHTML(entries, options)
	case FormatText:
		return convertToText(entries, options)
	default:
		return "", NewUnsupportedError(fmt.Sprintf("unsupported conversion format: %s", format))
	}
}

// convertToCSV converts entries to CSV.
func convertToCSV(entries []Entries, options ConvertOptions) (string, error) {
	buf := &bytes.Buffer{}
	// bytes.Buffer.Write cannot fail, so writeCSVToWriter cannot return an error here.
	_ = writeCSVToWriter(buf, entries, options)

	return buf.String(), nil
}

func writeCSVToWriter(w io.Writer, entries []Entries, options ConvertOptions) error {
	if isNilWriter(w) {
		return NewInvalidFormatError("writer is nil")
	}

	writer := csv.NewWriter(w)

	// 写入表头
	headers := getHeaders(options)
	if err := writer.Write(headers); err != nil {
		return NewFileSystemError("failed to write CSV", err)
	}

	// 写入数据行
	for _, entry := range entries {
		row := createDataRow(entry, options)
		if err := writer.Write(row); err != nil {
			return NewFileSystemError("failed to write CSV", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return NewFileSystemError("failed to write CSV", err)
	}

	return nil
}

// convertToMarkdown converts entries to a Markdown table.
func convertToMarkdown(entries []Entries, options ConvertOptions) (string, error) {
	buf := &bytes.Buffer{}

	// Write the header.
	headers := getHeaders(options)
	fmt.Fprintf(buf, "| %s |\n", strings.Join(headers, " | "))

	// Write the separator row.
	fmt.Fprintf(buf, "|%s|\n", strings.Repeat(" --- |", len(headers)))

	// Write the data rows.
	for _, entry := range entries {
		row := createDataRow(entry, options)
		for i, cell := range row {
			// Escape special Markdown characters.
			row[i] = strings.ReplaceAll(cell, "|", "\\|")
		}
		fmt.Fprintf(buf, "| %s |\n", strings.Join(row, " | "))
	}

	return buf.String(), nil
}

// convertToHTML converts entries to an HTML table.
func convertToHTML(entries []Entries, options ConvertOptions) (string, error) {
	buf := &bytes.Buffer{}

	// Start the table.
	fmt.Fprintln(buf, "<table border=\"1\">")

	// Write the header.
	headers := getHeaders(options)
	fmt.Fprintln(buf, "  <thead>")
	fmt.Fprintln(buf, "    <tr>")
	for _, header := range headers {
		fmt.Fprintf(buf, "      <th>%s</th>\n", escapeHTML(header))
	}
	fmt.Fprintln(buf, "    </tr>")
	fmt.Fprintln(buf, "  </thead>")

	// Write the data rows.
	fmt.Fprintln(buf, "  <tbody>")
	for _, entry := range entries {
		row := createDataRow(entry, options)
		fmt.Fprintln(buf, "    <tr>")
		for _, cell := range row {
			fmt.Fprintf(buf, "      <td>%s</td>\n", escapeHTML(cell))
		}
		fmt.Fprintln(buf, "    </tr>")
	}
	fmt.Fprintln(buf, "  </tbody>")

	// End the table.
	fmt.Fprintln(buf, "</table>")

	return buf.String(), nil
}

// convertToText converts entries to plain text.
func convertToText(entries []Entries, options ConvertOptions) (string, error) {
	buf := &bytes.Buffer{}

	// Write the header.
	headers := getHeaders(options)
	fmt.Fprintln(buf, strings.Join(headers, "\t"))
	fmt.Fprintln(buf, strings.Repeat("-", 80))

	// Write the data rows.
	for _, entry := range entries {
		row := createDataRow(entry, options)
		fmt.Fprintln(buf, strings.Join(row, "\t"))
	}

	return buf.String(), nil
}

// getHeaders returns the headers to include.
func getHeaders(options ConvertOptions) []string {
	if len(options.Headers) > 0 {
		return options.Headers
	}

	var headers []string

	if options.IncludeDateTime {
		headers = append(headers, "Date/Time")
	}
	if options.IncludeMethod {
		headers = append(headers, "Method")
	}
	if options.IncludeURL {
		headers = append(headers, "URL")
	}
	if options.IncludeStatus {
		headers = append(headers, "Status Code")
	}
	if options.IncludeContentType {
		headers = append(headers, "Content Type")
	}
	if options.IncludeSize {
		headers = append(headers, "Size (bytes)")
	}
	if options.IncludeTime {
		headers = append(headers, "Time (ms)")
	}
	if options.IncludeTimings {
		headers = append(headers, "Blocked (ms)", "DNS (ms)", "Connect (ms)", "Send (ms)", "Wait (ms)", "Receive (ms)")
	}
	if options.IncludePostData {
		headers = append(headers, "POST Data Type", "POST Data")
	}
	if options.IncludeQueryString {
		headers = append(headers, "Query Parameters")
	}
	if options.IncludeHeaders {
		headers = append(headers, "Request Headers", "Response Headers")
	}

	return headers
}

// createDataRow creates a data row.
func createDataRow(entry Entries, options ConvertOptions) []string {
	var row []string

	// Date and time.
	if options.IncludeDateTime {
		row = append(row, entry.StartedDateTime.Format(time.RFC3339))
	}

	// Request method.
	if options.IncludeMethod {
		row = append(row, entry.Request.Method)
	}

	// URL.
	if options.IncludeURL {
		row = append(row, entry.Request.URL)
	}

	// Status code.
	if options.IncludeStatus {
		row = append(row, fmt.Sprintf("%d %s", entry.Response.Status, entry.Response.StatusText))
	}

	// Content type.
	if options.IncludeContentType {
		row = append(row, entry.Response.Content.MimeType)
	}

	// Size.
	if options.IncludeSize {
		row = append(row, fmt.Sprintf("%d", entry.Response.Content.Size))
	}

	// Total time.
	if options.IncludeTime {
		row = append(row, fmt.Sprintf("%.2f", entry.Time))
	}

	// Detailed timings.
	if options.IncludeTimings {
		row = append(row,
			fmt.Sprintf("%.2f", entry.Timings.Blocked),
			fmt.Sprintf("%.2f", entry.Timings.DNS),
			fmt.Sprintf("%.2f", entry.Timings.Connect),
			fmt.Sprintf("%.2f", entry.Timings.Send),
			fmt.Sprintf("%.2f", entry.Timings.Wait),
			fmt.Sprintf("%.2f", entry.Timings.Receive),
		)
	}

	// POST data.
	if options.IncludePostData {
		if entry.Request.PostData != nil {
			row = append(row, entry.Request.PostData.MimeType, entry.Request.PostData.Text)
		} else {
			row = append(row, "", "")
		}
	}

	// Query parameters.
	if options.IncludeQueryString {
		var qs []string
		for _, param := range entry.Request.QueryString {
			qs = append(qs, fmt.Sprintf("%s=%s", param.Name, param.Value))
		}
		row = append(row, strings.Join(qs, "&"))
	}

	// Request and response headers.
	if options.IncludeHeaders {
		var reqHeaders []string
		for _, h := range entry.Request.Headers {
			reqHeaders = append(reqHeaders, fmt.Sprintf("%s: %s", h.Name, h.Value))
		}
		var respHeaders []string
		for _, h := range entry.Response.Headers {
			respHeaders = append(respHeaders, fmt.Sprintf("%s: %s", h.Name, h.Value))
		}
		row = append(row, strings.Join(reqHeaders, "; "), strings.Join(respHeaders, "; "))
	}

	return row
}

// escapeHTML escapes special HTML characters.
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}
