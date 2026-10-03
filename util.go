package har

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"
)

// ========== Har methods ==========

// Clone makes a deep copy of the entire Har object.
// It deep-copies all slices (Pages, Entries, Headers, Cookies, QueryString, and PostData.Params)
// so that changes to the clone do not affect the original.
func (h *Har) Clone() *Har {
	if h == nil {
		return nil
	}

	data, err := json.Marshal(h)
	if err != nil {
		return nil
	}

	// data came from a successful json.Marshal(h), so unmarshaling it into the same type must succeed.
	clone := &Har{}
	_ = json.Unmarshal(data, clone)
	return clone
}

// GetEntryCount returns the number of entries in the HAR file.
func (h *Har) GetEntryCount() int {
	if h == nil {
		return 0
	}
	return len(h.Log.Entries)
}

// Walk visits each entry and calls the visitor function.
// If the visitor returns an error, iteration stops and that error is returned.
func (h *Har) Walk(fn func(*Entries) error) error {
	if h == nil {
		return nil
	}
	if fn == nil {
		return NewInvalidFormatError("visitor is nil")
	}
	for i := range h.Log.Entries {
		if err := fn(&h.Log.Entries[i]); err != nil {
			return err
		}
	}
	return nil
}

// GetUniqueDomains returns a sorted list of unique domains from entry URLs.
func (h *Har) GetUniqueDomains() []string {
	if h == nil {
		return nil
	}

	domainSet := make(map[string]bool)
	for _, entry := range h.Log.Entries {
		domain := extractDomain(entry.Request.URL)
		if domain != "" {
			domainSet[domain] = true
		}
	}

	domains := make([]string, 0, len(domainSet))
	for d := range domainSet {
		domains = append(domains, d)
	}
	sort.Strings(domains)
	return domains
}

// Equals reports whether two HAR objects are equal.
// It compares versions, creators, browsers, entry counts, and each entry's method, URL, and status.
func (h *Har) Equals(other *Har) bool {
	if h == nil && other == nil {
		return true
	}
	if h == nil || other == nil {
		return false
	}

	// Compare versions.
	if h.Log.Version != other.Log.Version {
		return false
	}

	// Compare creators.
	if h.Log.Creator.Name != other.Log.Creator.Name || h.Log.Creator.Version != other.Log.Creator.Version {
		return false
	}

	// Compare browsers.
	if h.Log.Browser.Name != other.Log.Browser.Name || h.Log.Browser.Version != other.Log.Browser.Version {
		return false
	}

	// Compare entry counts.
	if len(h.Log.Entries) != len(other.Log.Entries) {
		return false
	}

	// Compare each entry's method, URL, and status code.
	for i := range h.Log.Entries {
		if h.Log.Entries[i].Request.Method != other.Log.Entries[i].Request.Method {
			return false
		}
		if h.Log.Entries[i].Request.URL != other.Log.Entries[i].Request.URL {
			return false
		}
		if h.Log.Entries[i].Response.Status != other.Log.Entries[i].Response.Status {
			return false
		}
	}

	return true
}

// SaveToFileGzipped saves the HAR file using gzip compression.
func (h *Har) SaveToFileGzipped(filePath string, indent bool) error {
	if h == nil {
		return NewInvalidFormatError("HAR object is nil")
	}

	data, err := h.ToJSON(indent)
	if err != nil {
		return err
	}

	f, err := os.Create(filePath)
	if err != nil {
		return NewFileSystemError(fmt.Sprintf("unable to create file '%s'", filePath), err)
	}
	return writeGzippedDataToFile(f, filePath, data)
}

// SaveToWriter writes HAR JSON to an io.Writer.
func (h *Har) SaveToWriter(w io.Writer, indent bool) error {
	if h == nil {
		return NewInvalidFormatError("HAR object is nil")
	}
	if isNilWriter(w) {
		return NewInvalidFormatError("writer is nil")
	}

	data, err := h.ToJSON(indent)
	if err != nil {
		return err
	}

	return writeAllToWriter(w, data, "failed to write HAR data")
}

func isNilWriter(w io.Writer) bool {
	if w == nil {
		return true
	}

	value := reflect.ValueOf(w)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func writeAllToWriter(w io.Writer, data []byte, message string) error {
	n, err := w.Write(data)
	if err != nil {
		return NewFileSystemError(message, err)
	}
	if n != len(data) {
		return NewFileSystemError(message, io.ErrShortWrite)
	}
	return nil
}

// ========== Entries methods ==========

// IsError reports whether the response is an error (status code >= 400).
func (e *Entries) IsError() bool {
	if e == nil {
		return false
	}
	return e.Response.Status >= 400
}

// IsRedirect reports whether the response is a redirect (3xx status code).
func (e *Entries) IsRedirect() bool {
	if e == nil {
		return false
	}
	return e.Response.Status >= 300 && e.Response.Status < 400
}

// IsSuccess reports whether the response is successful (2xx status code).
func (e *Entries) IsSuccess() bool {
	if e == nil {
		return false
	}
	return e.Response.Status >= 200 && e.Response.Status < 300
}

// GetElapsedTime converts the Time field from milliseconds to time.Duration.
func (e *Entries) GetElapsedTime() time.Duration {
	if e == nil {
		return 0
	}
	return time.Duration(e.Time * float64(time.Millisecond))
}

// GetURL parses and returns the request URL.
func (e *Entries) GetURL() *url.URL {
	if e == nil {
		return nil
	}
	u, err := url.Parse(e.Request.URL)
	if err != nil {
		return nil
	}
	return u
}

// GetDomain extracts the domain from the request URL.
func (e *Entries) GetDomain() string {
	if e == nil {
		return ""
	}
	return extractDomain(e.Request.URL)
}

// GetSize returns the total entry size (request headers, request body, response headers, and response body).
// Fields with a size of -1 (unknown) are treated as 0.
func (e *Entries) GetSize() int {
	if e == nil {
		return 0
	}

	reqHeadersSize := e.Request.HeadersSize
	if reqHeadersSize < 0 {
		reqHeadersSize = 0
	}
	reqBodySize := e.Request.BodySize
	if reqBodySize < 0 {
		reqBodySize = 0
	}
	respHeadersSize := e.Response.HeadersSize
	if respHeadersSize < 0 {
		respHeadersSize = 0
	}
	respBodySize := e.Response.BodySize
	if respBodySize < 0 {
		respBodySize = 0
	}

	return reqHeadersSize + reqBodySize + respHeadersSize + respBodySize
}

// GetRequestBody returns the request body bytes from PostData.Text.
func (e *Entries) GetRequestBody() []byte {
	if e == nil || e.Request.PostData == nil {
		return nil
	}
	return []byte(e.Request.PostData.Text)
}

// GetResponseBody returns the decoded response body.
// Base64-encoded content is decoded automatically; otherwise, the text bytes are returned.
func (e *Entries) GetResponseBody() ([]byte, error) {
	if e == nil {
		return nil, nil
	}

	text := e.Response.Content.Text
	if text == "" {
		return []byte{}, nil
	}

	// Decode base64-encoded content.
	if strings.EqualFold(e.Response.Content.Encoding, "base64") {
		data, err := base64.StdEncoding.DecodeString(text)
		if err != nil {
			return nil, NewHarError(ErrCodeInvalidFormat,
				fmt.Sprintf("failed to decode base64 response body: %v", err), err)
		}
		return data, nil
	}

	return []byte(text), nil
}

// ========== Request methods ==========

// GetHeader returns the first request header value with the specified name (case-insensitive).
func (r *Request) GetHeader(name string) string {
	if r == nil {
		return ""
	}
	for _, h := range r.Headers {
		if strings.EqualFold(h.Name, name) {
			return h.Value
		}
	}
	return ""
}

// GetHeaderValues returns all request header values with the specified name (case-insensitive).
func (r *Request) GetHeaderValues(name string) []string {
	if r == nil {
		return nil
	}
	var values []string
	for _, h := range r.Headers {
		if strings.EqualFold(h.Name, name) {
			values = append(values, h.Value)
		}
	}
	return values
}

// GetCookie returns the request cookie with the specified name (case-sensitive).
func (r *Request) GetCookie(name string) *Cookie {
	if r == nil {
		return nil
	}
	for i := range r.Cookies {
		if r.Cookies[i].Name == name {
			return &r.Cookies[i]
		}
	}
	return nil
}

// HasHeader reports whether a request header with the specified name exists (case-insensitive).
func (r *Request) HasHeader(name string) bool {
	if r == nil {
		return false
	}
	for _, h := range r.Headers {
		if strings.EqualFold(h.Name, name) {
			return true
		}
	}
	return false
}

// ========== Response methods ==========

// GetHeader returns the first response header value with the specified name (case-insensitive).
func (r *Response) GetHeader(name string) string {
	if r == nil {
		return ""
	}
	for _, h := range r.Headers {
		if strings.EqualFold(h.Name, name) {
			return h.Value
		}
	}
	return ""
}

// GetHeaderValues returns all response header values with the specified name (case-insensitive).
func (r *Response) GetHeaderValues(name string) []string {
	if r == nil {
		return nil
	}
	var values []string
	for _, h := range r.Headers {
		if strings.EqualFold(h.Name, name) {
			values = append(values, h.Value)
		}
	}
	return values
}

// GetCookie returns the response cookie with the specified name (case-sensitive).
func (r *Response) GetCookie(name string) *Cookie {
	if r == nil {
		return nil
	}
	for i := range r.Cookies {
		if r.Cookies[i].Name == name {
			return &r.Cookies[i]
		}
	}
	return nil
}

// HasHeader reports whether a response header with the specified name exists (case-insensitive).
func (r *Response) HasHeader(name string) bool {
	if r == nil {
		return false
	}
	for _, h := range r.Headers {
		if strings.EqualFold(h.Name, name) {
			return true
		}
	}
	return false
}

// GetContentType returns the Content-Type response header (case-insensitive).
func (r *Response) GetContentType() string {
	return r.GetHeader("Content-Type")
}

// ========== Content methods ==========

// EncodeContent base64-encodes binary data and sets the corresponding fields.
func (c *Content) EncodeContent(data []byte, mimeType string) {
	if c == nil {
		return
	}
	c.Text = base64.StdEncoding.EncodeToString(data)
	c.Encoding = "base64"
	c.MimeType = mimeType
	c.Size = len(data)
}

// SetText sets the text content and updates its size.
func (c *Content) SetText(text string) {
	if c == nil {
		return
	}
	c.Text = text
	c.Size = len(text)
}
