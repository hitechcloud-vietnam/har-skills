package har

import (
	"regexp"
	"sort"
	"strings"
	"time"
)

// FilterResult contains filtered entries.
type FilterResult struct {
	Entries []Entries
}

// FilterOptions defines entry filtering criteria.
type FilterOptions struct {
	URL             string    // Substring or regular expression to match in the URL.
	Method          string    // Request method.
	StatusCode      int       // Response status code.
	StatusCodeMin   int       // Minimum status code.
	StatusCodeMax   int       // Maximum status code.
	ContentType     string    // Content type.
	StartTime       time.Time // Start time.
	EndTime         time.Time // End time.
	MinDuration     float64   // Minimum request duration (ms).
	MaxDuration     float64   // Maximum request duration (ms).
	ResourceType    string    // Resource type.
	HasError        bool      // Whether an error occurred.
	HeaderName      string    // Request header name.
	HeaderValue     string    // Request header value.
	RespHeaderName  string    // Response header name.
	RespHeaderValue string    // Response header value.
	UseRegex        bool      // Use regular expression matching.
}

// Filter filters entries by criteria.
func (h *Har) Filter(options FilterOptions) *FilterResult {
	var result []Entries
	if h == nil {
		return &FilterResult{Entries: result}
	}

	for _, entry := range h.Log.Entries {
		if matchesFilter(entry, options) {
			result = append(result, entry)
		}
	}

	return &FilterResult{
		Entries: result,
	}
}

// Check whether an entry matches the filter criteria.
func matchesFilter(entry Entries, options FilterOptions) bool {
	// URL filter.
	if options.URL != "" {
		if options.UseRegex {
			re, err := regexp.Compile(options.URL)
			if err == nil && !re.MatchString(entry.Request.URL) {
				return false
			}
		} else if !strings.Contains(entry.Request.URL, options.URL) {
			return false
		}
	}

	// Request method filter.
	if options.Method != "" && entry.Request.Method != options.Method {
		return false
	}

	// Status code filter.
	if options.StatusCode > 0 && entry.Response.Status != options.StatusCode {
		return false
	}

	// Status code range filter.
	if options.StatusCodeMin > 0 && entry.Response.Status < options.StatusCodeMin {
		return false
	}
	if options.StatusCodeMax > 0 && entry.Response.Status > options.StatusCodeMax {
		return false
	}

	// Content-type filter (prefer the MimeType field, falling back to the response header).
	if options.ContentType != "" {
		matched := false
		// First check the Content.MimeType field.
		if strings.Contains(strings.ToLower(entry.Response.Content.MimeType), strings.ToLower(options.ContentType)) {
			matched = true
		}
		// If MimeType does not match, check the Content-Type response header.
		if !matched {
			for _, header := range entry.Response.Headers {
				if strings.EqualFold(header.Name, "Content-Type") && strings.Contains(strings.ToLower(header.Value), strings.ToLower(options.ContentType)) {
					matched = true
					break
				}
			}
		}
		if !matched {
			return false
		}
	}

	// Time-range filter.
	if !options.StartTime.IsZero() && entry.StartedDateTime.Before(options.StartTime) {
		return false
	}
	if !options.EndTime.IsZero() && entry.StartedDateTime.After(options.EndTime) {
		return false
	}

	// Duration filter.
	if options.MinDuration > 0 && entry.Time < options.MinDuration {
		return false
	}
	if options.MaxDuration > 0 && entry.Time > options.MaxDuration {
		return false
	}

	// Resource-type filter.
	if options.ResourceType != "" && entry.ResourceType != options.ResourceType {
		return false
	}

	// Error filter.
	if options.HasError && (entry.Response.Status < 400 || entry.Response.Status >= 600) {
		return false
	}

	// Request-header filter.
	if options.HeaderName != "" {
		matched := false
		for _, header := range entry.Request.Headers {
			if strings.EqualFold(header.Name, options.HeaderName) {
				if options.HeaderValue == "" || strings.Contains(header.Value, options.HeaderValue) {
					matched = true
					break
				}
			}
		}
		if !matched {
			return false
		}
	}

	// Response-header filter.
	if options.RespHeaderName != "" {
		matched := false
		for _, header := range entry.Response.Headers {
			if strings.EqualFold(header.Name, options.RespHeaderName) {
				if options.RespHeaderValue == "" || strings.Contains(header.Value, options.RespHeaderValue) {
					matched = true
					break
				}
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// Convenience filtering methods.

// FindByURL finds entries by URL.
func (h *Har) FindByURL(urlStr string, useRegex bool) *FilterResult {
	return h.Filter(FilterOptions{
		URL:      urlStr,
		UseRegex: useRegex,
	})
}

// FindByMethod finds entries by HTTP method.
func (h *Har) FindByMethod(method string) *FilterResult {
	return h.Filter(FilterOptions{
		Method: method,
	})
}

// FindByStatusCode finds entries by status code.
func (h *Har) FindByStatusCode(statusCode int) *FilterResult {
	return h.Filter(FilterOptions{
		StatusCode: statusCode,
	})
}

// FindErrors finds all failed requests.
func (h *Har) FindErrors() *FilterResult {
	return h.Filter(FilterOptions{
		HasError: true,
	})
}

// FindByTimeRange finds entries by time range.
func (h *Har) FindByTimeRange(start, end time.Time) *FilterResult {
	return h.Filter(FilterOptions{
		StartTime: start,
		EndTime:   end,
	})
}

// FindByContentType finds entries by content type.
func (h *Har) FindByContentType(contentType string) *FilterResult {
	return h.Filter(FilterOptions{
		ContentType: contentType,
	})
}

// FindSlowRequests finds slow requests.
func (h *Har) FindSlowRequests(minDuration float64) *FilterResult {
	return h.Filter(FilterOptions{
		MinDuration: minDuration,
	})
}

// FindByDomain finds entries by domain.
func (h *Har) FindByDomain(domain string) *FilterResult {
	var result []Entries
	if h == nil {
		return &FilterResult{Entries: result}
	}
	for _, entry := range h.Log.Entries {
		if d := extractDomain(entry.Request.URL); d == domain {
			result = append(result, entry)
		}
	}
	return &FilterResult{Entries: result}
}

// FindByHeader finds entries by request header.
func (h *Har) FindByHeader(name, value string) *FilterResult {
	return h.Filter(FilterOptions{
		HeaderName:  name,
		HeaderValue: value,
	})
}

// FindByResponseHeader finds entries by response header.
func (h *Har) FindByResponseHeader(name, value string) *FilterResult {
	return h.Filter(FilterOptions{
		RespHeaderName:  name,
		RespHeaderValue: value,
	})
}

// FindByCookie finds entries by cookie name in both request and response cookies.
func (h *Har) FindByCookie(name string) *FilterResult {
	var result []Entries
	if h == nil {
		return &FilterResult{Entries: result}
	}
	for _, entry := range h.Log.Entries {
		found := false
		// Search request cookies.
		for _, cookie := range entry.Request.Cookies {
			if cookie.Name == name {
				found = true
				break
			}
		}
		// Search response cookies.
		if !found {
			for _, cookie := range entry.Response.Cookies {
				if cookie.Name == name {
					found = true
					break
				}
			}
		}
		if found {
			result = append(result, entry)
		}
	}
	return &FilterResult{Entries: result}
}

// FindByStatusCodeRange finds entries by status-code range.
func (h *Har) FindByStatusCodeRange(min, max int) *FilterResult {
	return h.Filter(FilterOptions{
		StatusCodeMin: min,
		StatusCodeMax: max,
	})
}

// FindRedirects finds all redirect requests (3xx).
func (h *Har) FindRedirects() *FilterResult {
	return h.Filter(FilterOptions{
		StatusCodeMin: 300,
		StatusCodeMax: 399,
	})
}

// FindCacheHits finds all requests with cache hits.
// A cache hit is indicated by HitCount > 0 in BeforeRequest or AfterRequest.
func (h *Har) FindCacheHits() *FilterResult {
	var result []Entries
	if h == nil {
		return &FilterResult{Entries: result}
	}
	for _, entry := range h.Log.Entries {
		hit := false
		if entry.Cache.BeforeRequest != nil && entry.Cache.BeforeRequest.HitCount > 0 {
			hit = true
		}
		if entry.Cache.AfterRequest != nil && entry.Cache.AfterRequest.HitCount > 0 {
			hit = true
		}
		if hit {
			result = append(result, entry)
		}
	}
	return &FilterResult{Entries: result}
}

// FindByResourceType finds entries by resource type.
func (h *Har) FindByResourceType(resourceType string) *FilterResult {
	return h.Filter(FilterOptions{
		ResourceType: resourceType,
	})
}

// FindByServerIP finds entries by server IP address.
func (h *Har) FindByServerIP(ip string) *FilterResult {
	var result []Entries
	if h == nil {
		return &FilterResult{Entries: result}
	}
	for _, entry := range h.Log.Entries {
		if entry.ServerIPAddress == ip {
			result = append(result, entry)
		}
	}
	return &FilterResult{Entries: result}
}

// FindByConnection finds entries by connection ID.
func (h *Har) FindByConnection(connectionID string) *FilterResult {
	var result []Entries
	if h == nil {
		return &FilterResult{Entries: result}
	}
	for _, entry := range h.Log.Entries {
		if entry.Connection == connectionID {
			result = append(result, entry)
		}
	}
	return &FilterResult{Entries: result}
}

// ExtractDomain extracts the domain from a URL (public API).
// Supports URLs with ports, user information, and other components.
var ExtractDomain = extractDomain

// Count returns the number of filtered results.
func (fr *FilterResult) Count() int {
	if fr == nil {
		return 0
	}
	return len(fr.Entries)
}

// First returns the first result.
func (fr *FilterResult) First() *Entries {
	if fr == nil {
		return nil
	}
	if len(fr.Entries) > 0 {
		return &fr.Entries[0]
	}
	return nil
}

// Last returns the last result.
func (fr *FilterResult) Last() *Entries {
	if fr == nil {
		return nil
	}
	if len(fr.Entries) > 0 {
		return &fr.Entries[len(fr.Entries)-1]
	}
	return nil
}

// At returns the result at the specified index.
func (fr *FilterResult) At(index int) *Entries {
	if fr == nil {
		return nil
	}
	if index >= 0 && index < len(fr.Entries) {
		return &fr.Entries[index]
	}
	return nil
}

// SortByTime sorts by request start time.
func (fr *FilterResult) SortByTime() *FilterResult {
	if fr == nil {
		return nil
	}
	sort.Slice(fr.Entries, func(i, j int) bool {
		return fr.Entries[i].StartedDateTime.Before(fr.Entries[j].StartedDateTime)
	})
	return fr
}

// SortByDuration sorts by request duration, fastest first.
func (fr *FilterResult) SortByDuration() *FilterResult {
	if fr == nil {
		return nil
	}
	sort.Slice(fr.Entries, func(i, j int) bool {
		return fr.Entries[i].Time < fr.Entries[j].Time
	})
	return fr
}

// SortByDurationDesc sorts by request duration, slowest first.
func (fr *FilterResult) SortByDurationDesc() *FilterResult {
	if fr == nil {
		return nil
	}
	sort.Slice(fr.Entries, func(i, j int) bool {
		return fr.Entries[i].Time > fr.Entries[j].Time
	})
	return fr
}

// SortBySize sorts by response size, smallest first.
func (fr *FilterResult) SortBySize() *FilterResult {
	if fr == nil {
		return nil
	}
	sort.Slice(fr.Entries, func(i, j int) bool {
		return fr.Entries[i].Response.Content.Size < fr.Entries[j].Response.Content.Size
	})
	return fr
}

// SortBySizeDesc sorts by response size, largest first.
func (fr *FilterResult) SortBySizeDesc() *FilterResult {
	if fr == nil {
		return nil
	}
	sort.Slice(fr.Entries, func(i, j int) bool {
		return fr.Entries[i].Response.Content.Size > fr.Entries[j].Response.Content.Size
	})
	return fr
}

// Limit restricts the number of results.
func (fr *FilterResult) Limit(n int) *FilterResult {
	if fr == nil {
		return nil
	}
	if n <= 0 {
		fr.Entries = nil
		return fr
	}
	if n >= len(fr.Entries) {
		return fr
	}
	fr.Entries = fr.Entries[:n]
	return fr
}

// Offset skips the first N results.
func (fr *FilterResult) Offset(n int) *FilterResult {
	if fr == nil {
		return nil
	}
	if n <= 0 {
		return fr
	}
	if n >= len(fr.Entries) {
		fr.Entries = nil
		return fr
	}
	fr.Entries = fr.Entries[n:]
	return fr
}

// Chain applies additional filters to the current results.
func (fr *FilterResult) Chain(options FilterOptions) *FilterResult {
	var result []Entries
	if fr == nil {
		return &FilterResult{Entries: result}
	}
	for _, entry := range fr.Entries {
		if matchesFilter(entry, options) {
			result = append(result, entry)
		}
	}
	return &FilterResult{Entries: result}
}

// ToHar converts the filtered results into a new Har object.
func (fr *FilterResult) ToHar() *Har {
	har := NewHar()
	if fr == nil {
		return har
	}
	har.Log.Entries = fr.Entries
	return har
}
