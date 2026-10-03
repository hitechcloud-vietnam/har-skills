package har

import (
	"regexp"
	"sort"
	"time"
)

// HarIndex indexes HAR entries for fast lookups.
type HarIndex struct {
	byURL      map[string][]int // URL -> entry indices.
	byMethod   map[string][]int // HTTP method -> entry indices.
	byStatus   map[int][]int    // Status code -> entry indices.
	byDomain   map[string][]int // Domain -> entry indices.
	byMimeType map[string][]int // MIME type -> entry indices.
	har        *Har             // Associated Har object.
}

// IndexStats contains index statistics.
type IndexStats struct {
	UniqueURLs    int      // Number of unique URLs.
	UniqueDomains int      // Number of unique domains.
	StatusCodes   []int    // Status codes encountered.
	Methods       []string // HTTP methods encountered.
}

// BuildIndex builds all indexes for a Har object.
func (h *Har) BuildIndex() *HarIndex {
	if h == nil {
		return &HarIndex{
			byURL:      make(map[string][]int),
			byMethod:   make(map[string][]int),
			byStatus:   make(map[int][]int),
			byDomain:   make(map[string][]int),
			byMimeType: make(map[string][]int),
			har:        h,
		}
	}

	idx := &HarIndex{
		byURL:      make(map[string][]int),
		byMethod:   make(map[string][]int),
		byStatus:   make(map[int][]int),
		byDomain:   make(map[string][]int),
		byMimeType: make(map[string][]int),
		har:        h,
	}

	for i, entry := range h.Log.Entries {
		// URL index.
		idx.byURL[entry.Request.URL] = append(idx.byURL[entry.Request.URL], i)

		// Method index.
		idx.byMethod[entry.Request.Method] = append(idx.byMethod[entry.Request.Method], i)

		// Status-code index.
		idx.byStatus[entry.Response.Status] = append(idx.byStatus[entry.Response.Status], i)

		// Domain index.
		domain := extractDomain(entry.Request.URL)
		if domain != "" {
			idx.byDomain[domain] = append(idx.byDomain[domain], i)
		}

		// MIME-type index.
		mime := entry.Response.Content.MimeType
		if mime != "" {
			idx.byMimeType[mime] = append(idx.byMimeType[mime], i)
		}
	}

	return idx
}

// ByURL finds entries by exact URL.
func (idx *HarIndex) ByURL(urlStr string) []*Entries {
	if idx == nil {
		return nil
	}
	return idx.entriesByIndices(idx.byURL[urlStr])
}

// ByMethod finds entries by HTTP method.
func (idx *HarIndex) ByMethod(method string) []*Entries {
	if idx == nil {
		return nil
	}
	return idx.entriesByIndices(idx.byMethod[method])
}

// ByStatus finds entries by status code.
func (idx *HarIndex) ByStatus(code int) []*Entries {
	if idx == nil {
		return nil
	}
	return idx.entriesByIndices(idx.byStatus[code])
}

// ByDomain finds entries by domain.
func (idx *HarIndex) ByDomain(domain string) []*Entries {
	if idx == nil {
		return nil
	}
	return idx.entriesByIndices(idx.byDomain[domain])
}

// ByMimeType finds entries by MIME type.
func (idx *HarIndex) ByMimeType(mime string) []*Entries {
	if idx == nil {
		return nil
	}
	return idx.entriesByIndices(idx.byMimeType[mime])
}

// ByURLPattern finds entries using a URL regular expression.
func (idx *HarIndex) ByURLPattern(pattern string) []*Entries {
	if idx == nil {
		return nil
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}

	var result []*Entries
	for urlStr, indices := range idx.byURL {
		if re.MatchString(urlStr) {
			result = append(result, idx.entriesByIndices(indices)...)
		}
	}
	return result
}

// ByTimeRange finds entries by time range.
func (idx *HarIndex) ByTimeRange(start, end time.Time) []*Entries {
	if idx == nil || idx.har == nil {
		return nil
	}

	var result []*Entries
	for i := range idx.har.Log.Entries {
		entry := &idx.har.Log.Entries[i]
		if !entry.StartedDateTime.Before(start) && !entry.StartedDateTime.After(end) {
			result = append(result, entry)
		}
	}
	return result
}

// Size returns the total number of entries in the index.
func (idx *HarIndex) Size() int {
	if idx == nil || idx.har == nil || idx.har.Log.Entries == nil {
		return 0
	}
	return len(idx.har.Log.Entries)
}

// Stats returns index statistics.
func (idx *HarIndex) Stats() IndexStats {
	if idx == nil {
		return IndexStats{}
	}

	stats := IndexStats{
		UniqueURLs:    len(idx.byURL),
		UniqueDomains: len(idx.byDomain),
	}

	// Collect status codes.
	for code := range idx.byStatus {
		stats.StatusCodes = append(stats.StatusCodes, code)
	}
	sort.Ints(stats.StatusCodes)

	// Collect methods.
	for method := range idx.byMethod {
		stats.Methods = append(stats.Methods, method)
	}
	sort.Strings(stats.Methods)

	return stats
}

// entriesByIndices returns entry pointers for a list of indices.
func (idx *HarIndex) entriesByIndices(indices []int) []*Entries {
	if idx == nil || idx.har == nil || len(indices) == 0 {
		return nil
	}

	result := make([]*Entries, 0, len(indices))
	for _, i := range indices {
		if i >= 0 && i < len(idx.har.Log.Entries) {
			result = append(result, &idx.har.Log.Entries[i])
		}
	}
	return result
}
