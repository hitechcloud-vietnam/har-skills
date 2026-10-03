package har

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
)

//MIMECategory represents a MIME type category.type MIMECategory string

const (
	MIMEImage      MIMECategory = "image"
	MIMEScript     MIMECategory = "script"
	MIMEStylesheet MIMECategory = "stylesheet"
	MIMEFont       MIMECategory = "font"
	MIMEMedia      MIMECategory = "media"
	MIMEDocument   MIMECategory = "document"
	MIMEAPI        MIMECategory = "api"
	MIMEData       MIMECategory = "data"
	MIMEOther      MIMECategory = "other"
)

//ContentSummary summarizes all content in a HAR file.type ContentSummary struct {
	TotalSize      int                  //Total size.	TextSize       int                  //Text content size.	BinarySize     int                  //Binary content size.	CompressedSize int                  //Compressed size.	ByCategory     map[MIMECategory]int //Size by category.	ByMIMEType     map[string]int       //Size by specific MIME type.}

//MIMECategory returns the MIME category of the content.//
//It uses the Content.MimeType field to determine the category.//Common MIME types are supported; unrecognized types are categorized as MIMEOther.func (c *Content) MIMECategory() MIMECategory {
	if c == nil {
		return MIMEOther
	}

	mime := strings.ToLower(c.MimeType)
	// Remove parameters (e.g., "text/html; charset=utf-8" -> "text/html")
	if idx := strings.Index(mime, ";"); idx >= 0 {
		mime = strings.TrimSpace(mime[:idx])
	}

	// Image
	if strings.HasPrefix(mime, "image/") {
		return MIMEImage
	}

	// Script
	if mime == "application/javascript" || mime == "application/x-javascript" ||
		mime == "text/javascript" || mime == "text/x-javascript" ||
		mime == "application/ecmascript" || mime == "text/ecmascript" ||
		strings.HasPrefix(mime, "application/vnd.dart") {
		return MIMEScript
	}

	// Stylesheet
	if mime == "text/css" || mime == "text/x-css" ||
		strings.HasPrefix(mime, "application/x-css") {
		return MIMEStylesheet
	}

	// Font
	if strings.HasPrefix(mime, "font/") ||
		mime == "application/x-font-ttf" || mime == "application/x-font-woff" ||
		mime == "application/font-woff" || mime == "application/font-woff2" ||
		mime == "application/x-font-opentype" || mime == "application/vnd.ms-fontobject" ||
		strings.HasPrefix(mime, "application/x-font-") {
		return MIMEFont
	}

	// Media (audio/video)
	if strings.HasPrefix(mime, "audio/") || strings.HasPrefix(mime, "video/") {
		return MIMEMedia
	}

	// Document
	if mime == "text/html" || mime == "application/xhtml+xml" ||
		mime == "text/xml" || mime == "application/xml" ||
		mime == "text/plain" || mime == "text/richtext" ||
		mime == "application/pdf" || mime == "application/msword" ||
		strings.HasPrefix(mime, "application/vnd.") && !strings.Contains(mime, "json") {
		return MIMEDocument
	}

	// API
	if mime == "application/json" || strings.HasSuffix(mime, "+json") ||
		mime == "text/json" || mime == "application/graphql" {
		return MIMEAPI
	}

	// Data
	if mime == "text/csv" || mime == "text/tab-separated-values" ||
		mime == "application/x-www-form-urlencoded" ||
		mime == "multipart/form-data" ||
		mime == "application/octet-stream" ||
		strings.HasPrefix(mime, "application/vnd.") && strings.Contains(mime, "json") {
		return MIMEData
	}

	// Other text types not yet classified
	if strings.HasPrefix(mime, "text/") {
		return MIMEDocument
	}

	return MIMEOther
}

//IsBinary checks whether the content is binary.//
//It checks the MIME type and content bytes.//Text MIME types (text/*, application/json, application/xml, application/javascript, etc.)//are considered non-binary. If the content text is available, http.DetectContentType() is also used.func (c *Content) IsBinary() bool {
	if c == nil {
		return false
	}

	// Check declared MIME type first
	if isTextMIME(c.MimeType) {
		return false
	}

	// If content text is available, use http.DetectContentType for further detection
	data, err := c.DecodeContent()
	if err == nil && len(data) > 0 {
		detected := http.DetectContentType(data)
		if isTextMIME(detected) {
			return false
		}
	}

	return true
}

//IsText checks whether the content is text.//
//It is the inverse of IsBinary: text content returns true.func (c *Content) IsText() bool {
	if c == nil {
		return false
	}
	return !c.IsBinary()
}

//DetectMIMEType uses http.DetectContentType to detect the actual MIME type of the content.//
//If the content text is available, it decodes the content bytes before detecting the MIME type.//If detection is not possible (the content is empty or decoding fails), it falls back to Content.MimeType.func (c *Content) DetectMIMEType() string {
	if c == nil {
		return ""
	}

	data, err := c.DecodeContent()
	if err == nil && len(data) > 0 {
		detected := http.DetectContentType(data)
		// http.DetectContentType returns "application/octet-stream" when it can't detect
		if detected != "application/octet-stream" {
			return detected
		}
	}

	return c.MimeType
}

//Hash calculates the SHA-256 hash of the content.//
//It hashes the decoded content bytes and returns the hash as a hexadecimal string.func (c *Content) Hash() (string, error) {
	if c == nil {
		return "", NewInvalidFormatError("Content is empty")
	}

	data, err := c.DecodeContent()
	if err != nil {
		return "", err
	}
	if data == nil {
		return "", NewInvalidFormatError("Content data is empty")
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash), nil
}

//ParseJSON parses the content text as a JSON value.//
//It returns the parsed JSON value, which may be an object, array, string, and so on.//It returns an error if the content is empty or is not valid JSON.func (c *Content) ParseJSON() (interface{}, error) {
	if c == nil {
		return nil, NewInvalidFormatError("Content is empty")
	}

	data, err := c.DecodeContent()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, NewInvalidFormatError("Content data is empty")
	}

	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, NewHarError(ErrCodeJSONParse,
			fmt.Sprintf("JSON parsing failed: %v", err), err)
	}

	return result, nil
}

//ParseAsMap parses the content text as a JSON object (map).//
//It attempts to parse the content as a JSON object (map[string]interface{}).//It returns an error if the content is not a JSON object (for example, if it is an array or string).func (c *Content) ParseAsMap() (map[string]interface{}, error) {
	if c == nil {
		return nil, NewInvalidFormatError("Content is empty")
	}

	data, err := c.DecodeContent()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, NewInvalidFormatError("Content data is empty")
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, NewHarError(ErrCodeJSONParse,
			fmt.Sprintf("JSON object parsing failed: %v", err), err)
	}
	if result == nil {
		return nil, NewInvalidFormatError("JSON content is not an object")
	}

	return result, nil
}

//ContentLength returns the value of the response Content-Length header.//
//It looks up the Content-Length header in Response.Headers and returns its integer value.//It returns -1 if the header is not present.func (e *Entries) ContentLength() int {
	if e == nil {
		return -1
	}

	for _, header := range e.Response.Headers {
		if strings.EqualFold(header.Name, "Content-Length") {
			val, err := strconv.Atoi(strings.TrimSpace(header.Value))
			if err != nil {
				return -1
			}
			return val
		}
	}

	return -1
}

//HasContentLengthMismatch checks whether the Content-Length header differs from the actual content size.//
//It compares the Content-Length header value with Response.Content.Size and returns true if they differ.func (e *Entries) HasContentLengthMismatch() bool {
	if e == nil {
		return false
	}

	contentLen := e.ContentLength()
	if contentLen < 0 {
		return false
	}

	return contentLen != e.Response.Content.Size
}

//EstimateTransferSize estimates the actual transfer size.//
//It estimates the content actual network transfer size, accounting for compression and other factors.//It prefers Response.TransferSize (a Chrome extension field),//then Response.BodySize minus Compression, and finally Content.Size.func (e *Entries) EstimateTransferSize() int {
	if e == nil {
		return 0
	}

	// Prefer TransferSize (Chrome extension) if available
	if e.Response.TransferSize > 0 {
		return e.Response.TransferSize
	}

	// Use BodySize minus Compression savings
	if e.Response.BodySize > 0 {
		compression := e.Response.Content.Compression
		if compression > 0 && e.Response.BodySize > compression {
			return e.Response.BodySize - compression
		}
		return e.Response.BodySize
	}

	// Fall back to Content.Size
	return e.Response.Content.Size
}

//ContentSummary returns a summary of all content in the HAR file.//
//It summarizes content types and sizes for all entries, including total, text, and binary sizes,//compressed sizes, and sizes grouped by MIME category and specific MIME type.func (h *Har) ContentSummary() *ContentSummary {
	if h == nil {
		return nil
	}

	summary := &ContentSummary{
		ByCategory: make(map[MIMECategory]int),
		ByMIMEType: make(map[string]int),
	}

	for _, entry := range h.Log.Entries {
		content := entry.Response.Content
		size := content.Size
		if size < 0 {
			size = 0
		}

		summary.TotalSize += size

		category := content.MIMECategory()
		summary.ByCategory[category] += size

		mimeKey := content.MimeType
		if mimeKey == "" {
			mimeKey = "unknown"
		}
		summary.ByMIMEType[mimeKey] += size

		if content.IsText() {
			summary.TextSize += size
		} else {
			summary.BinarySize += size
		}

		compression := content.Compression
		if compression > 0 {
			summary.CompressedSize += compression
		}
	}

	return summary
}

//SaveToFile saves the decoded content to a file.//
//It automatically decodes base64 and decompresses content, then writes the resulting raw data to the specified path.func (c *Content) SaveToFile(path string) error {
	if c == nil {
		return NewInvalidFormatError("Content is empty")
	}

	data, err := c.DecodeContent()
	if err != nil {
		return err
	}
	if data == nil {
		// Empty content — write zero-length file
		data = []byte{}
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return NewFileSystemError(fmt.Sprintf("Failed to write file '%s'", path), err)
	}

	return nil
}

//isTextMIME checks whether the MIME type is a text type.func isTextMIME(mime string) bool {
	if mime == "" {
		return false
	}

	lower := strings.ToLower(mime)
	// Remove parameters
	if idx := strings.Index(lower, ";"); idx >= 0 {
		lower = strings.TrimSpace(lower[:idx])
	}

	// text/* is always text
	if strings.HasPrefix(lower, "text/") {
		return true
	}

	// Common application types that are actually text
	textApplicationTypes := []string{
		"application/json",
		"application/xml",
		"application/javascript",
		"application/x-javascript",
		"application/ecmascript",
		"application/graphql",
		"application/xhtml+xml",
		"application/atom+xml",
		"application/rss+xml",
		"application/soap+xml",
		"application/x-yaml",
		"application/yaml",
		"application/toml",
		"application/ld+json",
		"application/manifest+json",
		"application/schema+json",
		"application/vnd.api+json",
	}

	for _, t := range textApplicationTypes {
		if lower == t {
			return true
		}
	}

	// Any type ending with +json or +xml is text
	if strings.HasSuffix(lower, "+json") || strings.HasSuffix(lower, "+xml") {
		return true
	}

	return false
}
