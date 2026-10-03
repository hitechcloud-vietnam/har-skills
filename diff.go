package har

import (
	"fmt"
	"strings"
)

// HarDiff represents the differences between two HAR files.
type HarDiff struct {
	Added     []DiffEntry     // Added requests.
	Removed   []DiffEntry     // Removed requests.
	Modified  []ModifiedEntry // Modified requests.
	Unchanged int             // Number of unchanged requests.
}

// DiffEntry represents a single entry in a diff.
type DiffEntry struct {
	Method string // HTTP method.
	URL    string // Request URL.
	Status int    // Response status code.
	Index  int    // Index in the original HAR file.
}

// ModifiedEntry represents a modified entry.
type ModifiedEntry struct {
	Method  string        // HTTP method.
	URL     string        // Request URL.
	Changes []FieldChange // List of field changes.
	Old     *Entries      // Old entry.
	New     *Entries      // New entry.
}

// FieldChange represents a change to a single field.
type FieldChange struct {
	Field    string      // Field name.
	OldValue interface{} // Old value.
	NewValue interface{} // New value.
}

// DiffOptions configures diff comparisons.
type DiffOptions struct {
	IgnoreHeaders []string // Header names to ignore.
	IgnoreTimings bool     // Ignore timing differences.
	IgnoreDates   bool     // Ignore date differences.
	IgnoreCache   bool     // Ignore cache differences.
	IgnoreComment bool     // Ignore comment differences.
	NormalizeURL  bool     // Normalize URLs by sorting query parameters.
	CompareByURL  bool     // Match by URL (default: index and URL).
	IncludeBody   bool     // Compare response bodies.
}

// DefaultDiffOptions returns the default diff options.
func DefaultDiffOptions() DiffOptions {
	return DiffOptions{
		IgnoreTimings: true,
		IgnoreDates:   true,
		IgnoreCache:   true,
	}
}

// Diff compares two HAR files.
func Diff(har1, har2 *Har, options DiffOptions) *HarDiff {
	result := &HarDiff{}

	if har1 == nil && har2 == nil {
		return result
	}
	if har1 == nil {
		for i, entry := range har2.Log.Entries {
			result.Added = append(result.Added, DiffEntry{
				Method: entry.Request.Method,
				URL:    entry.Request.URL,
				Status: entry.Response.Status,
				Index:  i,
			})
		}
		return result
	}
	if har2 == nil {
		for i, entry := range har1.Log.Entries {
			result.Removed = append(result.Removed, DiffEntry{
				Method: entry.Request.Method,
				URL:    entry.Request.URL,
				Status: entry.Response.Status,
				Index:  i,
			})
		}
		return result
	}

	// Build maps of entries.
	entries1 := buildEntryMap(har1, options)
	entries2 := buildEntryMap(har2, options)

	// Find added and modified entries.
	for key, entry2 := range entries2 {
		if entry1, ok := entries1[key]; ok {
			// Compare entry differences.
			changes := compareEntries(entry1, entry2, options)
			if len(changes) > 0 {
				result.Modified = append(result.Modified, ModifiedEntry{
					Method:  entry2.Request.Method,
					URL:     entry2.Request.URL,
					Changes: changes,
					Old:     entry1,
					New:     entry2,
				})
			} else {
				result.Unchanged++
			}
		} else {
			result.Added = append(result.Added, DiffEntry{
				Method: entry2.Request.Method,
				URL:    entry2.Request.URL,
				Status: entry2.Response.Status,
			})
		}
	}

	// Find removed entries.
	for key, entry1 := range entries1 {
		if _, ok := entries2[key]; !ok {
			result.Removed = append(result.Removed, DiffEntry{
				Method: entry1.Request.Method,
				URL:    entry1.Request.URL,
				Status: entry1.Response.Status,
			})
		}
	}

	return result
}

// entryKey generates a unique key for an entry.
func entryKey(entry *Entries, options DiffOptions) string {
	method := entry.Request.Method
	u := entry.Request.URL

	if options.NormalizeURL {
		u = normalizeURL(u, nil)
	}

	return method + " " + u
}

// buildEntryMap builds a map of entries.
func buildEntryMap(har *Har, options DiffOptions) map[string]*Entries {
	result := make(map[string]*Entries)

	for i := range har.Log.Entries {
		key := entryKey(&har.Log.Entries[i], options)
		// Handle duplicate keys.
		if _, exists := result[key]; exists {
			key = fmt.Sprintf("%s_%d", key, i)
		}
		result[key] = &har.Log.Entries[i]
	}

	return result
}

// compareEntries compares two entries.
func compareEntries(entry1, entry2 *Entries, options DiffOptions) []FieldChange {
	var changes []FieldChange

	// Compare response status codes.
	if entry1.Response.Status != entry2.Response.Status {
		changes = append(changes, FieldChange{
			Field:    "response.status",
			OldValue: entry1.Response.Status,
			NewValue: entry2.Response.Status,
		})
	}

	// Compare response status text.
	if entry1.Response.StatusText != entry2.Response.StatusText {
		changes = append(changes, FieldChange{
			Field:    "response.statusText",
			OldValue: entry1.Response.StatusText,
			NewValue: entry2.Response.StatusText,
		})
	}

	// Compare total duration.
	if !options.IgnoreTimings && entry1.Time != entry2.Time {
		changes = append(changes, FieldChange{
			Field:    "time",
			OldValue: entry1.Time,
			NewValue: entry2.Time,
		})
	}

	// Compare response content types.
	if entry1.Response.Content.MimeType != entry2.Response.Content.MimeType {
		changes = append(changes, FieldChange{
			Field:    "response.content.mimeType",
			OldValue: entry1.Response.Content.MimeType,
			NewValue: entry2.Response.Content.MimeType,
		})
	}

	// Compare response content sizes.
	if entry1.Response.Content.Size != entry2.Response.Content.Size {
		changes = append(changes, FieldChange{
			Field:    "response.content.size",
			OldValue: entry1.Response.Content.Size,
			NewValue: entry2.Response.Content.Size,
		})
	}

	// Compare response body content.
	if options.IncludeBody && entry1.Response.Content.Text != entry2.Response.Content.Text {
		changes = append(changes, FieldChange{
			Field:    "response.content.text",
			OldValue: entry1.Response.Content.Text,
			NewValue: entry2.Response.Content.Text,
		})
	}

	// Compare request headers.
	changes = append(changes, compareHeaders(entry1.Request.Headers, entry2.Request.Headers, "request.headers", options)...)

	// Compare response headers.
	changes = append(changes, compareHeaders(entry1.Response.Headers, entry2.Response.Headers, "response.headers", options)...)

	return changes
}

// compareHeaders compares header differences.
func compareHeaders(headers1, headers2 []Headers, prefix string, options DiffOptions) []FieldChange {
	var changes []FieldChange

	// Build a set of headers to ignore.
	ignoreSet := make(map[string]bool)
	for _, h := range options.IgnoreHeaders {
		ignoreSet[strings.ToLower(h)] = true
	}

	// Convert headers to maps for easier lookup.
	map1 := headersToMap(headers1, ignoreSet)
	map2 := headersToMap(headers2, ignoreSet)

	// Find added and modified headers.
	for name, value2 := range map2 {
		if value1, ok := map1[name]; ok {
			if value1 != value2 {
				changes = append(changes, FieldChange{
					Field:    fmt.Sprintf("%s.%s", prefix, name),
					OldValue: value1,
					NewValue: value2,
				})
			}
		} else {
			changes = append(changes, FieldChange{
				Field:    fmt.Sprintf("%s.%s", prefix, name),
				OldValue: nil,
				NewValue: value2,
			})
		}
	}

	// Find removed headers.
	for name, value1 := range map1 {
		if _, ok := map2[name]; !ok {
			changes = append(changes, FieldChange{
				Field:    fmt.Sprintf("%s.%s", prefix, name),
				OldValue: value1,
				NewValue: nil,
			})
		}
	}

	return changes
}

// headersToMap converts headers to a map and filters out ignored headers.
func headersToMap(headers []Headers, ignoreSet map[string]bool) map[string]string {
	result := make(map[string]string)
	for _, h := range headers {
		if ignoreSet[strings.ToLower(h.Name)] {
			continue
		}
		result[h.Name] = h.Value
	}
	return result
}

// HasChanges reports whether the diff contains any changes.
func (d *HarDiff) HasChanges() bool {
	if d == nil {
		return false
	}
	return len(d.Added) > 0 || len(d.Removed) > 0 || len(d.Modified) > 0
}

// TotalChanges returns the total number of changes.
func (d *HarDiff) TotalChanges() int {
	if d == nil {
		return 0
	}
	return len(d.Added) + len(d.Removed) + len(d.Modified)
}

// Report generates a diff report.
func (d *HarDiff) Report(format ConvertFormat) string {
	if d == nil {
		d = &HarDiff{}
	}

	var sb strings.Builder

	switch format {
	case FormatText:
		d.writeTextReport(&sb)
	case FormatMarkdown:
		d.writeMarkdownReport(&sb)
	case FormatCSV:
		d.writeCSVReport(&sb)
	default:
		d.writeTextReport(&sb)
	}

	return sb.String()
}

// writeTextReport writes a text-format report.
func (d *HarDiff) writeTextReport(sb *strings.Builder) {
	sb.WriteString("HAR Diff Report\n")
	sb.WriteString(strings.Repeat("=", 60) + "\n\n")

	sb.WriteString(fmt.Sprintf("Total changes: %d (added: %d, removed: %d, modified: %d, unchanged: %d)\n\n",
		d.TotalChanges(), len(d.Added), len(d.Removed), len(d.Modified), d.Unchanged))

	if len(d.Added) > 0 {
		sb.WriteString("Added Requests:\n")
		for _, a := range d.Added {
			sb.WriteString(fmt.Sprintf("  + [%d] %s %s (status: %d)\n", a.Index, a.Method, a.URL, a.Status))
		}
		sb.WriteString("\n")
	}

	if len(d.Removed) > 0 {
		sb.WriteString("Removed Requests:\n")
		for _, r := range d.Removed {
			sb.WriteString(fmt.Sprintf("  - [%d] %s %s (status: %d)\n", r.Index, r.Method, r.URL, r.Status))
		}
		sb.WriteString("\n")
	}

	if len(d.Modified) > 0 {
		sb.WriteString("Modified Requests:\n")
		for _, m := range d.Modified {
			sb.WriteString(fmt.Sprintf("  ~ %s %s\n", m.Method, m.URL))
			for _, c := range m.Changes {
				sb.WriteString(fmt.Sprintf("      %s: %v -> %v\n", c.Field, c.OldValue, c.NewValue))
			}
		}
	}
}

// writeMarkdownReport writes a Markdown-format report.
func (d *HarDiff) writeMarkdownReport(sb *strings.Builder) {
	sb.WriteString("# HAR Diff Report\n\n")
	sb.WriteString(fmt.Sprintf("**Total changes**: %d | **Added**: %d | **Removed**: %d | **Modified**: %d | **Unchanged**: %d\n\n",
		d.TotalChanges(), len(d.Added), len(d.Removed), len(d.Modified), d.Unchanged))

	if len(d.Added) > 0 {
		sb.WriteString("## Added Requests\n\n")
		sb.WriteString("| Method | URL | Status Code |\n")
		sb.WriteString("| --- | --- | --- |\n")
		for _, a := range d.Added {
			sb.WriteString(fmt.Sprintf("| %s | %s | %d |\n", a.Method, a.URL, a.Status))
		}
		sb.WriteString("\n")
	}

	if len(d.Removed) > 0 {
		sb.WriteString("## Removed Requests\n\n")
		sb.WriteString("| Method | URL | Status Code |\n")
		sb.WriteString("| --- | --- | --- |\n")
		for _, r := range d.Removed {
			sb.WriteString(fmt.Sprintf("| %s | %s | %d |\n", r.Method, r.URL, r.Status))
		}
		sb.WriteString("\n")
	}

	if len(d.Modified) > 0 {
		sb.WriteString("## Modified Requests\n\n")
		for _, m := range d.Modified {
			sb.WriteString(fmt.Sprintf("### %s %s\n\n", m.Method, m.URL))
			sb.WriteString("| Field | Old Value | New Value |\n")
			sb.WriteString("| --- | --- | --- |\n")
			for _, c := range m.Changes {
				sb.WriteString(fmt.Sprintf("| %s | %v | %v |\n", c.Field, c.OldValue, c.NewValue))
			}
			sb.WriteString("\n")
		}
	}
}

// writeCSVReport writes a CSV-format report.
func (d *HarDiff) writeCSVReport(sb *strings.Builder) {
	sb.WriteString("type,method,url,field,old_value,new_value\n")

	for _, a := range d.Added {
		sb.WriteString(fmt.Sprintf("added,%s,%s,,,\"%d\"\n", a.Method, a.URL, a.Status))
	}
	for _, r := range d.Removed {
		sb.WriteString(fmt.Sprintf("removed,%s,%s,,,\"%d\"\n", r.Method, r.URL, r.Status))
	}
	for _, m := range d.Modified {
		for _, c := range m.Changes {
			sb.WriteString(fmt.Sprintf("modified,%s,%s,%s,%v,%v\n", m.Method, m.URL, c.Field, c.OldValue, c.NewValue))
		}
	}
}
