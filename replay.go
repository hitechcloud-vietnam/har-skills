package har

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ReplayOptions configures request replay.
type ReplayOptions struct {
	Timeout         time.Duration     // Request timeout.
	FollowRedirects bool              // Whether to follow redirects.
	MaxRedirects    int               // Maximum number of redirects.
	SkipSSLVerify   bool              // Whether to skip SSL certificate verification.
	OverrideHeaders map[string]string // Request headers to override.
	Transport       http.RoundTripper // Custom transport.
}

// DefaultReplayOptions returns the default replay options.
func DefaultReplayOptions() ReplayOptions {
	return ReplayOptions{
		Timeout:         30 * time.Second,
		FollowRedirects: true,
		MaxRedirects:    10,
		SkipSSLVerify:   false,
	}
}

// ReplayResult represents the result of replaying a single request.
type ReplayResult struct {
	Entry    *Entries       // Original HAR entry.
	Response *http.Response // HTTP response.
	Duration time.Duration  // Request duration.
	Error    error          // Error, if any.
	Index    int            // Entry index.
}

// ToHTTPRequest converts a HAR entry to a standard-library http.Request.
//
// It builds a complete http.Request from the request information in the HAR entry,
// including the method, URL, headers, cookies, and body.
func (e *Entries) ToHTTPRequest() (*http.Request, error) {
	if e == nil {
		return nil, NewInvalidFormatError("entry is nil")
	}

	// Parse the URL.
	parsedURL, err := url.Parse(e.Request.URL)
	if err != nil {
		return nil, NewInvalidValueError("request.url", e.Request.URL,
			fmt.Sprintf("failed to parse URL: %v", err))
	}

	// Build the request body.
	var body io.Reader
	if e.Request.PostData != nil && e.Request.PostData.Text != "" {
		body = strings.NewReader(e.Request.PostData.Text)
	}

	// Create the request.
	req, err := http.NewRequest(e.Request.Method, parsedURL.String(), body)
	if err != nil {
		return nil, NewHarError(ErrCodeInvalidFormat,
			fmt.Sprintf("failed to create HTTP request: %v", err), err)
	}

	// Set request headers.
	for _, header := range e.Request.Headers {
		req.Header.Set(header.Name, header.Value)
	}

	// Set cookies.
	for _, cookie := range e.Request.Cookies {
		req.AddCookie(&http.Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Path:     cookie.Path,
			Domain:   cookie.Domain,
			HttpOnly: cookie.HTTPOnly,
			Secure:   cookie.Secure,
		})
	}

	// Set Content-Type if PostData is present.
	if e.Request.PostData != nil && e.Request.PostData.MimeType != "" {
		req.Header.Set("Content-Type", e.Request.PostData.MimeType)
	}

	return req, nil
}

// Replay replays the HTTP request from a single HAR entry.
//
// It converts the HAR entry to an HTTP request, executes it, and returns the replay result.
func (e *Entries) Replay(options ReplayOptions) (*ReplayResult, error) {
	if e == nil {
		return nil, NewInvalidFormatError("entry is nil")
	}

	// Build the HTTP request.
	req, err := e.ToHTTPRequest()
	if err != nil {
		return nil, err
	}

	// Apply header overrides.
	for name, value := range options.OverrideHeaders {
		req.Header.Set(name, value)
	}

	// Create the HTTP client.
	client := createHTTPClient(options)

	// Execute and time the request.
	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)

	if err != nil {
		replayErr := normalizeReplayError(err)
		return &ReplayResult{
			Entry:    e,
			Duration: duration,
			Error:    replayErr,
		}, replayErr
	}

	return &ReplayResult{
		Entry:    e,
		Response: resp,
		Duration: duration,
	}, nil
}

// ReplayAll replays the HTTP requests from every entry in the HAR file.
//
// It executes requests sequentially and returns the replay result for each entry.
func (h *Har) ReplayAll(options ReplayOptions) ([]*ReplayResult, error) {
	if h == nil {
		return nil, NewInvalidFormatError("HAR object is nil")
	}

	results := make([]*ReplayResult, len(h.Log.Entries))
	var firstErr error

	for i := range h.Log.Entries {
		entry := &h.Log.Entries[i]
		result, err := entry.Replay(options)
		if result == nil {
			result = &ReplayResult{
				Entry: entry,
				Error: err,
			}
		}
		result.Index = i
		results[i] = result
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return results, firstErr
}

// ReplaySelective replays entries that match the specified criteria.
func (h *Har) ReplaySelective(options ReplayOptions, filterOptions FilterOptions) ([]*ReplayResult, error) {
	if h == nil {
		return nil, NewInvalidFormatError("HAR object is nil")
	}

	filtered := h.Filter(filterOptions)
	if filtered.Count() == 0 {
		return nil, nil
	}

	results := make([]*ReplayResult, filtered.Count())
	var firstErr error

	for i := range filtered.Entries {
		entry := &filtered.Entries[i]
		result, err := entry.Replay(options)
		if result == nil {
			result = &ReplayResult{
				Entry: entry,
				Error: err,
			}
		}
		result.Index = i
		results[i] = result
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return results, firstErr
}

// HTTPResponseToEntries converts an http.Response to HAR Entries.
//
// This helper converts a standard-library HTTP response to a HAR entry
// for use with the replay functionality.
func HTTPResponseToEntries(req *Entries, resp *http.Response, duration time.Duration) *Entries {
	if resp == nil {
		return nil
	}

	entry := &Entries{
		StartedDateTime: time.Now(),
		Time:            float64(duration.Milliseconds()),
	}
	if req != nil {
		entry.Request = req.Request
	}

	// Build the response.
	entry.Response = Response{
		Status:      resp.StatusCode,
		StatusText:  resp.Status,
		HTTPVersion: resp.Proto,
		HeadersSize: -1,
		BodySize:    -1,
	}

	// Read response headers.
	for key, values := range resp.Header {
		for _, value := range values {
			entry.Response.Headers = append(entry.Response.Headers, Headers{
				Name:  key,
				Value: value,
			})
		}
	}

	// Read response cookies.
	for _, cookie := range resp.Cookies() {
		entry.Response.Cookies = append(entry.Response.Cookies, Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Path:     cookie.Path,
			Domain:   cookie.Domain,
			HTTPOnly: cookie.HttpOnly,
			Secure:   cookie.Secure,
		})
	}

	// Read the response body.
	if !isNilReader(resp.Body) {
		bodyBytes, readErr, closeErr := readAndCloseResponseBody(resp.Body)
		if bodyErr := responseBodyErrorMessage(readErr, closeErr); bodyErr != "" {
			entry.Response.Error = bodyErr
		}
		if readErr == nil {
			entry.Response.Content = Content{
				Size:     len(bodyBytes),
				MimeType: resp.Header.Get("Content-Type"),
				Text:     string(bodyBytes),
			}
			entry.Response.BodySize = len(bodyBytes)
		}
	}

	return entry
}

// createHTTPClient creates an HTTP client using the specified options.
func createHTTPClient(options ReplayOptions) *http.Client {
	transport := options.Transport
	if isNilReplayTransport(transport) {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: options.SkipSSLVerify,
			},
		}
	}

	client := &http.Client{
		Timeout:   options.Timeout,
		Transport: transport,
	}

	if !options.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	} else if options.MaxRedirects > 0 {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) >= options.MaxRedirects {
				return newMaxRedirectsError(options.MaxRedirects)
			}
			return nil
		}
	}

	return client
}

func normalizeReplayError(err error) error {
	if err == nil {
		return nil
	}

	var harErr *HarError
	if errors.As(err, &harErr) {
		return harErr
	}
	return err
}

func newMaxRedirectsError(maxRedirects int) *HarError {
	return NewHarError(
		ErrCodeInvalidValue,
		"maximum redirects exceeded",
		fmt.Errorf("stopped after %d redirects", maxRedirects),
	).WithMetadata("maxRedirects", maxRedirects)
}

// ReplayResultsToHar converts replay results back to a HAR object.
func ReplayResultsToHar(results []*ReplayResult) *Har {
	h := NewHar()
	h.SetCreator("go-har-replay", "1.0")

	for _, result := range results {
		if result == nil {
			continue
		}

		if result.Response != nil {
			entry := HTTPResponseToEntries(result.Entry, result.Response, result.Duration)
			h.Log.Entries = append(h.Log.Entries, *entry)
		} else if result.Entry != nil {
			// Preserve the original entry even if the request failed.
			h.Log.Entries = append(h.Log.Entries, *result.Entry)
		}
	}

	return h
}

// BuildQueryStringFromURL parses query parameters from a URL string.
func BuildQueryStringFromURL(rawURL string) []QueryString {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}

	var params []QueryString
	for key, values := range parsedURL.Query() {
		for _, value := range values {
			params = append(params, QueryString{
				Name:  key,
				Value: value,
			})
		}
	}

	return params
}

// ParseResponseHeaders parses a raw HTTP response header string.
func ParseResponseHeaders(headerStr string) []Headers {
	var headers []Headers
	lines := strings.Split(headerStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			headers = append(headers, Headers{
				Name:  strings.TrimSpace(parts[0]),
				Value: strings.TrimSpace(parts[1]),
			})
		}
	}
	return headers
}

// EstimateHeaderSize estimates the size of HTTP headers.
func EstimateHeaderSize(headers []Headers) int {
	size := 0
	for _, h := range headers {
		size += len(h.Name) + len(h.Value) + 4 // name: value\r\n
	}
	return size
}

// FormatBytes formats a byte count.
func FormatBytes(size int) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case size >= GB:
		return strconv.FormatFloat(float64(size)/float64(GB), 'f', 2, 64) + " GB"
	case size >= MB:
		return strconv.FormatFloat(float64(size)/float64(MB), 'f', 2, 64) + " MB"
	case size >= KB:
		return strconv.FormatFloat(float64(size)/float64(KB), 'f', 2, 64) + " KB"
	default:
		return strconv.Itoa(size) + " B"
	}
}

// ReadBody reads the request body content.
func ReadBody(entry *Entries) ([]byte, error) {
	if entry == nil || entry.Request.PostData == nil {
		return nil, nil
	}
	return []byte(entry.Request.PostData.Text), nil
}

// WriteRequestToWriter writes an HTTP request to an io.Writer for debugging.
func WriteRequestToWriter(entry *Entries, w io.Writer) error {
	if entry == nil {
		return NewInvalidFormatError("entry is nil")
	}
	if isNilWriter(w) {
		return NewInvalidFormatError("writer is nil")
	}

	var builder strings.Builder
	_, _ = fmt.Fprintf(&builder, "%s %s %s\r\n", entry.Request.Method, entry.Request.URL, entry.Request.HTTPVersion)

	for _, h := range entry.Request.Headers {
		_, _ = fmt.Fprintf(&builder, "%s: %s\r\n", h.Name, h.Value)
	}

	builder.WriteString("\n")

	if entry.Request.PostData != nil && entry.Request.PostData.Text != "" {
		builder.WriteString(entry.Request.PostData.Text)
	}

	return writeAllToWriter(w, []byte(builder.String()), "failed to write HTTP request")
}

// CloneEntry makes a deep copy of a HAR entry.
func CloneEntry(entry *Entries) *Entries {
	if entry == nil {
		return nil
	}

	cloned := *entry

	// Copy slices.
	cloned.Request.Headers = make([]Headers, len(entry.Request.Headers))
	copy(cloned.Request.Headers, entry.Request.Headers)

	cloned.Request.Cookies = make([]Cookie, len(entry.Request.Cookies))
	copy(cloned.Request.Cookies, entry.Request.Cookies)

	cloned.Request.QueryString = make([]QueryString, len(entry.Request.QueryString))
	copy(cloned.Request.QueryString, entry.Request.QueryString)

	if entry.Request.PostData != nil {
		pd := *entry.Request.PostData
		if len(entry.Request.PostData.Params) > 0 {
			pd.Params = make([]Param, len(entry.Request.PostData.Params))
			copy(pd.Params, entry.Request.PostData.Params)
		}
		cloned.Request.PostData = &pd
	}

	cloned.Response.Headers = make([]Headers, len(entry.Response.Headers))
	copy(cloned.Response.Headers, entry.Response.Headers)

	cloned.Response.Cookies = make([]Cookie, len(entry.Response.Cookies))
	copy(cloned.Response.Cookies, entry.Response.Cookies)

	return &cloned
}
