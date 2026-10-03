package har

import (
	"fmt"
	"sort"
	"time"
)

// MergeOptions defines merge options.
type MergeOptions struct {
	SortByTime  bool // Sort merged entries by time.
	Deduplicate bool // Deduplicate by Method+URL, keeping the newest entry.
}

// DefaultMergeOptions returns the default merge options.
func DefaultMergeOptions() MergeOptions {
	return MergeOptions{
		SortByTime:  true,
		Deduplicate: false,
	}
}

// Merge combines multiple HAR files.
//
// Combine entries from multiple HAR files into one HAR file.
// The merged HAR file uses the version and creator information from the first HAR file.
func Merge(hars ...*Har) *Har {
	return MergeWithOptions(DefaultMergeOptions(), hars...)
}

// MergeWithOptions merges multiple HAR files using the specified options.
func MergeWithOptions(options MergeOptions, hars ...*Har) *Har {
	if len(hars) == 0 {
		return NewHar()
	}

	result := NewHar()

	// Use metadata from the first non-nil Har.
	for _, h := range hars {
		if h == nil {
			continue
		}
		result.Log.Version = h.Log.Version
		result.Log.Creator = h.Log.Creator
		result.Log.Browser = h.Log.Browser
		break
	}

	// Combine all entries and pages.
	for _, h := range hars {
		if h == nil {
			continue
		}
		result.Log.Entries = append(result.Log.Entries, h.Log.Entries...)
		result.Log.Pages = append(result.Log.Pages, h.Log.Pages...)
	}

	// Deduplicate.
	if options.Deduplicate {
		result.Log.Entries = deduplicateEntries(result.Log.Entries)
	}

	// Sort entries.
	if options.SortByTime {
		sortEntriesByTime(result.Log.Entries)
	}

	return result
}

// deduplicateEntries deduplicates by Method+URL, keeping the newest entry.
func deduplicateEntries(entries []Entries) []Entries {
	seen := make(map[string]int) // key -> index in result
	var result []Entries

	for _, entry := range entries {
		key := entry.Request.Method + " " + entry.Request.URL
		if idx, ok := seen[key]; ok {
			// Keep the newer entry.
			if entry.StartedDateTime.After(result[idx].StartedDateTime) {
				result[idx] = entry
			}
		} else {
			seen[key] = len(result)
			result = append(result, entry)
		}
	}

	return result
}

// sortEntriesByTime sorts entries by time.
func sortEntriesByTime(entries []Entries) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].StartedDateTime.Before(entries[j].StartedDateTime)
	})
}

// SplitByPage splits a HAR file by page reference.
//
// Group HAR entries by pageref and return a map of HAR files keyed by pageref.
// Entries without a pageref are grouped under an empty-string key.
func (h *Har) SplitByPage() map[string]*Har {
	result := make(map[string]*Har)

	if h == nil {
		return result
	}

	// Collect all pages.
	pagesMap := make(map[string]Pages)
	for _, page := range h.Log.Pages {
		pagesMap[page.ID] = page
	}

	// Group by pageref.
	groups := make(map[string][]Entries)
	for _, entry := range h.Log.Entries {
		ref := entry.Pageref
		groups[ref] = append(groups[ref], entry)
	}

	// Create a Har object for each group.
	for ref, entries := range groups {
		har := NewHar()
		har.Log.Version = h.Log.Version
		har.Log.Creator = h.Log.Creator

		if page, ok := pagesMap[ref]; ok {
			har.Log.Pages = []Pages{page}
		}

		har.Log.Entries = entries
		result[ref] = har
	}

	return result
}

// SplitByDomain splits a HAR file by domain.
//
// Group HAR entries by request domain and return a map of HAR files keyed by domain.
func (h *Har) SplitByDomain() map[string]*Har {
	result := make(map[string]*Har)

	if h == nil {
		return result
	}

	// Group by domain.
	groups := make(map[string][]Entries)
	for _, entry := range h.Log.Entries {
		domain := extractDomain(entry.Request.URL)
		groups[domain] = append(groups[domain], entry)
	}

	// Create a Har object for each group.
	for domain, entries := range groups {
		har := NewHar()
		har.Log.Version = h.Log.Version
		har.Log.Creator = h.Log.Creator
		har.Log.Entries = entries
		result[domain] = har
	}

	return result
}

// SplitByTimeRange splits a HAR file by time range.
//
// Group HAR entries by the specified time interval.
// For example, an interval of one hour places all entries from that hour in each HAR file.
func (h *Har) SplitByTimeRange(interval time.Duration) []*Har {
	if h == nil || len(h.Log.Entries) == 0 || interval <= 0 {
		return nil
	}

	// Sort by time.
	sorted := make([]Entries, len(h.Log.Entries))
	copy(sorted, h.Log.Entries)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].StartedDateTime.Before(sorted[j].StartedDateTime)
	})

	// Group by time interval.
	var result []*Har
	var currentGroup []Entries
	var groupStart time.Time

	for i, entry := range sorted {
		if i == 0 {
			groupStart = entry.StartedDateTime
			currentGroup = append(currentGroup, entry)
			continue
		}

		if entry.StartedDateTime.Sub(groupStart) >= interval {
			// Create a Har object for the current group.
			har := NewHar()
			har.Log.Version = h.Log.Version
			har.Log.Creator = h.Log.Creator
			har.Log.Entries = currentGroup
			result = append(result, har)

			// Start a new group.
			currentGroup = []Entries{entry}
			groupStart = entry.StartedDateTime
		} else {
			currentGroup = append(currentGroup, entry)
		}
	}

	// Process the last group.
	if len(currentGroup) > 0 {
		har := NewHar()
		har.Log.Version = h.Log.Version
		har.Log.Creator = h.Log.Creator
		har.Log.Entries = currentGroup
		result = append(result, har)
	}

	return result
}

// SplitBySize splits a HAR file by entry count.
//
// Group HAR entries by the specified count; each HAR file contains at most maxEntries entries.
func (h *Har) SplitBySize(maxEntries int) []*Har {
	if h == nil || maxEntries <= 0 {
		return nil
	}

	total := len(h.Log.Entries)
	if total == 0 {
		return nil
	}

	numGroups := (total + maxEntries - 1) / maxEntries
	result := make([]*Har, 0, numGroups)

	for i := 0; i < total; i += maxEntries {
		end := i + maxEntries
		if end > total {
			end = total
		}

		har := NewHar()
		har.Log.Version = h.Log.Version
		har.Log.Creator = h.Log.Creator

		entries := make([]Entries, end-i)
		copy(entries, h.Log.Entries[i:end])
		har.Log.Entries = entries

		result = append(result, har)
	}

	return result
}

// SplitByStatusCode splits a HAR file by status-code range.
//
// Group HAR entries by status-code range: 2xx, 3xx, 4xx, and 5xx.
func (h *Har) SplitByStatusCode() map[string]*Har {
	result := make(map[string]*Har)

	if h == nil {
		return result
	}

	groups := make(map[string][]Entries)
	for _, entry := range h.Log.Entries {
		var group string
		status := entry.Response.Status
		switch {
		case status >= 200 && status < 300:
			group = "2xx"
		case status >= 300 && status < 400:
			group = "3xx"
		case status >= 400 && status < 500:
			group = "4xx"
		case status >= 500 && status < 600:
			group = "5xx"
		default:
			group = fmt.Sprintf("%dxx", status/100)
		}
		groups[group] = append(groups[group], entry)
	}

	for group, entries := range groups {
		har := NewHar()
		har.Log.Version = h.Log.Version
		har.Log.Creator = h.Log.Creator
		har.Log.Entries = entries
		result[group] = har
	}

	return result
}

// SplitByMethod splits HAR entries by HTTP method.
func (h *Har) SplitByMethod() map[string]*Har {
	result := make(map[string]*Har)

	if h == nil {
		return result
	}

	groups := make(map[string][]Entries)
	for _, entry := range h.Log.Entries {
		method := entry.Request.Method
		groups[method] = append(groups[method], entry)
	}

	for method, entries := range groups {
		har := NewHar()
		har.Log.Version = h.Log.Version
		har.Log.Creator = h.Log.Creator
		har.Log.Entries = entries
		result[method] = har
	}

	return result
}
