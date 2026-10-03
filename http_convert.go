package har

import (
	"io"
	"net/http"
	"strings"
)

// EntryMeta carries optional metadata for a HAR entry.
// An upstream instrumentation system may have additional information beyond *http.Request / *http.Response,
// such as server IP, connection ID, or page reference, which cannot be inferred directly from req/resp.
// Pass this structure to AddEntryFromHTTPWithMeta.
type EntryMeta struct {
	// ServerIPAddress is the peer server IP (HAR field serverIPAddress).
	ServerIPAddress string
	// Connection identifies entries that reuse the same connection (HAR field connection).
	Connection string
	// Pageref is the page reference and must match an id registered with HarBuilder.AddPage.
	Pageref string
	// InitiatorType / InitiatorURL / InitiatorLine describe the request initiator
	// (Chrome extension field _initiator), such as "script", "parser", or "other".
	InitiatorType string
	InitiatorURL  string
	InitiatorLine int
	// Priority is the resource priority (Chrome extension field _priority), such as "High" or "Low".
	Priority string
	// ResourceType is the resource type (Chrome extension field _resourceType), such as "xhr" or "script".
	ResourceType string
	// Comment is an entry comment.
	Comment string
}

// HeadersFromHTTP converts net/http request/response headers to HAR []Headers.
// Each header value (including multiple values) is expanded into a separate Headers entry, preserving original casing.
func HeadersFromHTTP(h http.Header) []Headers {
	if h == nil {
		return []Headers{}
	}
	out := make([]Headers, 0, len(h))
	for key, values := range h {
		for _, value := range values {
			out = append(out, Headers{Name: key, Value: value})
		}
	}
	return out
}

// CookiesFromHTTP converts a net/http cookie slice to HAR []Cookie.
// Only fields available directly from http.Cookie are populated: Name, Value, Path, Domain, HTTPOnly, and Secure.
// Expires and SameSite exist in http.Cookie but are not inferred here; callers can set them afterward if needed.
func CookiesFromHTTP(cookies []*http.Cookie) []Cookie {
	if len(cookies) == 0 {
		return []Cookie{}
	}
	out := make([]Cookie, 0, len(cookies))
	for _, cookie := range cookies {
		out = append(out, Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Path:     cookie.Path,
			Domain:   cookie.Domain,
			HTTPOnly: cookie.HttpOnly,
			Secure:   cookie.Secure,
		})
	}
	return out
}

// PostDataFromRequest reads and constructs the request body.
// This consumes req.Body (io.ReadAll followed by Close), so callers must cache a copy if they still need the body.
// The returned PostData may be nil (if there is no body or reading fails).
// The second return value is the body size in bytes, used to set Request.BodySize.
//
// Automatically detect Content-Type:
// - application/x-www-form-urlencoded → parse form fields into PostData.Params and leave Text empty
// - Otherwise, the raw string is stored in PostData.Text.
//
// If mimeType is empty, fall back to req.Header.Get("Content-Type").
func PostDataFromRequest(req *http.Request) (*PostData, int) {
	if req == nil || isNilReader(req.Body) {
		return nil, 0
	}
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil || len(bodyBytes) == 0 {
		return nil, 0
	}
	contentType := req.Header.Get("Content-Type")

	var postData *PostData
	if strings.HasPrefix(strings.ToLower(contentType), "application/x-www-form-urlencoded") {
		// Parse form fields while preserving raw values, matching form behavior without modifying req.Form.
		params := parseFormParams(string(bodyBytes))
		postData = &PostData{
			MimeType: contentType,
			Params:   params,
		}
	} else {
		postData = &PostData{
			MimeType: contentType,
			Text:     string(bodyBytes),
		}
	}
	return postData, len(bodyBytes)
}

// parseFormParams parses an application/x-www-form-urlencoded body into []Param.
func parseFormParams(body string) []Param {
	if body == "" {
		return []Param{}
	}
	out := make([]Param, 0)
	for _, pair := range strings.Split(body, "&") {
		if pair == "" {
			continue
		}
		key, value, found := strings.Cut(pair, "=")
		if !found {
			out = append(out, Param{Name: key})
			continue
		}
		out = append(out, Param{Name: key, Value: value})
	}
	return out
}

// isTextContentType roughly determines whether Content-Type is text, to decide whether the body needs base64 encoding.
// Non-text content (images, audio, video, fonts, and binary octet-stream) should be base64-encoded for lossless JSON round trips.
func isTextContentType(mimeType string) bool {
	m := strings.ToLower(strings.TrimSpace(mimeType))
	if m == "" {
		return true // Unknown types are treated as text for backward compatibility.
	}
	switch {
	case strings.HasPrefix(m, "text/"),
		strings.Contains(m, "json"),
		strings.Contains(m, "xml"),
		strings.Contains(m, "javascript"),
		strings.Contains(m, "urlencoded"),
		strings.Contains(m, "form-data"),
		strings.HasPrefix(m, "application/"):
		// Most application/* subtypes are text unless they are binary types such as images or fonts; here, application subtypes are
		// checked against known binary families and treated as text when none match.
		if strings.Contains(m, "image") ||
			strings.Contains(m, "audio") ||
			strings.Contains(m, "video") ||
			strings.Contains(m, "font") ||
			strings.Contains(m, "octet-stream") ||
			strings.Contains(m, "pdf") ||
			strings.Contains(m, "zip") ||
			strings.Contains(m, "gzip") {
			return false
		}
		return true
	}
	// Top-level types such as image/*, audio/*, video/*, and font/* are treated as binary.
	return false
}
