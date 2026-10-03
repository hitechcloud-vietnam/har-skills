package har

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// DedupStrategy defines a deduplication strategy.
type DedupStrategy int

const (
	DedupExactURL    DedupStrategy = iota // Exact URL match.
	DedupURLPattern                       // URL-pattern match, ignoring specified parameters.
	DedupContentHash                      // Content-hash-based.
)

// DeduplicateOptions configures deduplication.
type DeduplicateOptions struct {
	Strategy       DedupStrategy // Deduplication strategy.
	IgnoreParams   []string      // Query parameters to ignore (such as cache busters).
	CompareHeaders bool          // Whether to include headers when comparing.
	CompareBody    bool          // Whether to include request bodies when comparing.
}

// DuplicateGroup represents a group of duplicate requests.
type DuplicateGroup struct {
	Key          string // Deduplication key (URL pattern, hash, etc.).
	EntryIndices []int  // Indices of duplicate entries.
	Count        int    // Number of duplicates.
}

// DefaultDeduplicateOptions returns the default deduplication options.
// Uses the DedupURLPattern strategy and ignores common cache-busting parameters.
func DefaultDeduplicateOptions() DeduplicateOptions {
	return DeduplicateOptions{
		Strategy:     DedupURLPattern,
		IgnoreParams: defaultCacheBusterParams(),
	}
}

// defaultCacheBusterParams returns common cache-busting parameter names.
func defaultCacheBusterParams() []string {
	return []string{"_", "cb", "cachebuster", "timestamp", "t", "rand", "random", "v"}
}

// IsCacheBusterParam checks whether a parameter name looks like a cache buster.
// A parameter is a cache buster if its name is common or if its name is "v" and its value is numeric.
func IsCacheBusterParam(name string) bool {
	lower := strings.ToLower(name)
	commonBusters := map[string]bool{
		"_":           true,
		"cb":          true,
		"cachebuster": true,
		"timestamp":   true,
		"t":           true,
		"rand":        true,
		"random":      true,
	}
	return commonBusters[lower]
}

// IsCacheBusterParamWithValue checks whether a parameter name and value look like a cache buster.
// For the "v" parameter, only numeric values are considered cache busters.
func IsCacheBusterParamWithValue(name, value string) bool {
	lower := strings.ToLower(name)
	if lower == "v" {
		_, err := strconv.Atoi(value)
		return err == nil
	}
	return IsCacheBusterParam(name)
}

// FindDuplicates finds duplicate or near-duplicate requests.
func (h *Har) FindDuplicates(opts DeduplicateOptions) []DuplicateGroup {
	if h == nil || len(h.Log.Entries) == 0 {
		return nil
	}

	groups := make(map[string][]int) // key -> entry indices

	for i, entry := range h.Log.Entries {
		key := computeDedupKey(entry, opts)
		groups[key] = append(groups[key], i)
	}

	var result []DuplicateGroup
	for key, indices := range groups {
		if len(indices) > 1 {
			result = append(result, DuplicateGroup{
				Key:          key,
				EntryIndices: indices,
				Count:        len(indices),
			})
		}
	}

	return result
}

// Deduplicate removes duplicate requests, keeping the first occurrence.
func (h *Har) Deduplicate(opts DeduplicateOptions) *Har {
	if h == nil {
		return nil
	}

	cloned := h.Clone()
	if cloned == nil {
		return nil
	}
	if len(cloned.Log.Entries) == 0 {
		return cloned
	}

	seen := make(map[string]bool)
	var deduped []Entries

	for _, entry := range cloned.Log.Entries {
		key := computeDedupKey(entry, opts)
		if !seen[key] {
			seen[key] = true
			deduped = append(deduped, entry)
		}
	}

	cloned.Log.Entries = deduped
	return cloned
}

// computeDedupKey calculates the deduplication key for the selected strategy.
func computeDedupKey(entry Entries, opts DeduplicateOptions) string {
	switch opts.Strategy {
	case DedupExactURL:
		return computeExactURLKey(entry, opts)
	case DedupURLPattern:
		return computeURLPatternKey(entry, opts)
	case DedupContentHash:
		return computeContentHashKey(entry, opts)
	default:
		return computeURLPatternKey(entry, opts)
	}
}

// computeExactURLKey returns the exact-URL-match key.
func computeExactURLKey(entry Entries, opts DeduplicateOptions) string {
	key := entry.Request.Method + " " + entry.Request.URL
	if opts.CompareHeaders {
		key += " " + headersKey(entry.Request.Headers)
	}
	if opts.CompareBody && entry.Request.PostData != nil {
		key += " " + entry.Request.PostData.Text
	}
	return key
}

// computeURLPatternKey returns the URL-pattern key, ignoring specified parameters.
func computeURLPatternKey(entry Entries, opts DeduplicateOptions) string {
	normalizedURL := normalizeURL(entry.Request.URL, opts.IgnoreParams)
	key := entry.Request.Method + " " + normalizedURL
	if opts.CompareHeaders {
		key += " " + headersKey(entry.Request.Headers)
	}
	if opts.CompareBody && entry.Request.PostData != nil {
		key += " " + entry.Request.PostData.Text
	}
	return key
}

// computeContentHashKey returns a content-hash-based key.
// The content-hash strategy compares response content and excludes URLs because different URLs may return the same content.
func computeContentHashKey(entry Entries, opts DeduplicateOptions) string {
	h := sha256.New()

	// Method.
	h.Write([]byte(entry.Request.Method))

	// Response status code.
	h.Write([]byte(fmt.Sprintf("%d", entry.Response.Status)))

	// Response MIME type.
	h.Write([]byte(entry.Response.Content.MimeType))

	// Request body.
	if opts.CompareBody && entry.Request.PostData != nil {
		h.Write([]byte(entry.Request.PostData.Text))
	}

	// Request headers.
	if opts.CompareHeaders {
		h.Write([]byte(headersKey(entry.Request.Headers)))
	}

	// Response body (content hashes generally compare response content).
	if entry.Response.Content.Text != "" {
		h.Write([]byte(entry.Response.Content.Text))
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

// normalizeURL normalizes a URL while ignoring specified parameters.
// When ignoreParams is nil, query parameters are sorted.
// When ignoreParams is not nil, specified parameters are removed and the remaining parameters are sorted.
func normalizeURL(rawURL string, ignoreParams []string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	q := u.Query()

	if len(ignoreParams) > 0 {
		ignoreSet := make(map[string]bool, len(ignoreParams))
		for _, p := range ignoreParams {
			ignoreSet[strings.ToLower(p)] = true
		}

		newQ := make(url.Values)
		for name, values := range q {
			if !ignoreSet[strings.ToLower(name)] {
				newQ[name] = values
			}
		}
		u.RawQuery = newQ.Encode()
	} else {
		// Sort query parameters for consistent normalization
		u.RawQuery = q.Encode()
	}

	return u.String()
}

// headersKey serializes a header list into a comparable string.
func headersKey(headers []Headers) string {
	var sb strings.Builder
	for _, h := range headers {
		sb.WriteString(h.Name)
		sb.WriteString(":")
		sb.WriteString(h.Value)
		sb.WriteString(";")
	}
	return sb.String()
}
