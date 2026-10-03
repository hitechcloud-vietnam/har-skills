package har

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
)

// FormatJSON is the JSON format constant.
const FormatJSON ConvertFormat = "json"

// ---------------------------------------------------------------------------
// cURL export.
// ---------------------------------------------------------------------------

// ToCurl generates cURL commands for all entries.
func (h *Har) ToCurl() string {
	if h == nil || len(h.Log.Entries) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, entry := range h.Log.Entries {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(entryToCurl(&entry))
	}
	return sb.String()
}

// ToCurl generates a cURL command for a single entry.
func (e *Entries) ToCurl() string {
	if e == nil {
		return ""
	}
	return entryToCurl(e)
}

// entryToCurl converts a single HAR entry to a cURL command.
func entryToCurl(entry *Entries) string {
	if entry == nil {
		return ""
	}

	var parts []string

	// cURL command.
	parts = append(parts, "curl")

	// Use -X for non-GET methods.
	method := strings.ToUpper(entry.Request.Method)
	if method != "GET" {
		parts = append(parts, fmt.Sprintf("-X %s", method))
	}

	// Request headers.
	for _, h := range entry.Request.Headers {
		// Skip the Host header; cURL adds it automatically.
		if strings.EqualFold(h.Name, "Host") {
			continue
		}
		escaped := escapeSingleQuotes(h.Value)
		parts = append(parts, fmt.Sprintf("-H '%s: %s'", h.Name, escaped))
	}

	// POST data.
	if entry.Request.PostData != nil && entry.Request.PostData.Text != "" {
		escaped := escapeSingleQuotes(entry.Request.PostData.Text)
		parts = append(parts, fmt.Sprintf("--data '%s'", escaped))
	}

	// Check whether Accept-Encoding contains gzip or deflate.
	if hasAcceptEncoding(entry) {
		parts = append(parts, "--compressed")
	}

	// Check whether to skip SSL verification (based on whether the URL uses HTTPS).
	parsedURL, err := url.Parse(entry.Request.URL)
	if err == nil && parsedURL.Scheme == "https" {
		// Add -k if the _error field exists or the URL uses a self-signed certificate.
		// Conservatively check for SSL-related errors in the response.
		if entry.Response.Error != nil {
			parts = append(parts, "-k")
		}
	}

	// URL (wrapped in single quotes).
	parts = append(parts, fmt.Sprintf("'%s'", entry.Request.URL))

	return strings.Join(parts, " \\\n  ")
}

// ---------------------------------------------------------------------------
// Wget export.
// ---------------------------------------------------------------------------

// ToWget generates wget commands for all entries.
func (h *Har) ToWget() string {
	if h == nil || len(h.Log.Entries) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, entry := range h.Log.Entries {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(entryToWget(&entry))
	}
	return sb.String()
}

// ToWget generates a wget command for a single entry.
func (e *Entries) ToWget() string {
	if e == nil {
		return ""
	}
	return entryToWget(e)
}

// entryToWget converts a single HAR entry to a wget command.
func entryToWget(entry *Entries) string {
	if entry == nil {
		return ""
	}

	var parts []string

	parts = append(parts, "wget")

	method := strings.ToUpper(entry.Request.Method)
	// wget uses GET by default; use --method for other methods.
	if method != "GET" {
		parts = append(parts, fmt.Sprintf("--method=%s", method))
	}

	// Request headers.
	for _, h := range entry.Request.Headers {
		if strings.EqualFold(h.Name, "Host") {
			continue
		}
		escaped := escapeSingleQuotes(h.Value)
		parts = append(parts, fmt.Sprintf("--header='%s: %s'", h.Name, escaped))
	}

	// POST data.
	if entry.Request.PostData != nil && entry.Request.PostData.Text != "" {
		escaped := escapeSingleQuotes(entry.Request.PostData.Text)
		parts = append(parts, fmt.Sprintf("--post-data='%s'", escaped))
	}

	// Skip SSL verification.
	parsedURL, err := url.Parse(entry.Request.URL)
	if err == nil && parsedURL.Scheme == "https" {
		parts = append(parts, "--no-check-certificate")
	}

	// Quiet mode and write to stdout.
	parts = append(parts, "-qO-")

	// URL
	parts = append(parts, fmt.Sprintf("'%s'", entry.Request.URL))

	return strings.Join(parts, " \\\n  ")
}

// ---------------------------------------------------------------------------
// Python Requests export.
// ---------------------------------------------------------------------------

// ToPythonRequests generates Python requests code for all entries.
func (h *Har) ToPythonRequests() string {
	if h == nil || len(h.Log.Entries) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("import requests\n\n")
	for i, entry := range h.Log.Entries {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(entryToPythonRequests(&entry))
		sb.WriteString("\n")
	}
	return sb.String()
}

// ToPythonRequests generates Python requests code for a single entry.
func (e *Entries) ToPythonRequests() string {
	if e == nil {
		return ""
	}
	return entryToPythonRequests(e)
}

// entryToPythonRequests converts a single HAR entry to Python requests code.
func entryToPythonRequests(entry *Entries) string {
	if entry == nil {
		return ""
	}

	var sb strings.Builder

	method := strings.ToLower(entry.Request.Method)

	// Build the headers dictionary.
	headers := buildHeadersDict(entry)

	// Build the request call.
	if len(headers) > 0 {
		sb.WriteString(fmt.Sprintf("headers = %s\n", headers))
	}

	// Build the request parameters.
	args := []string{fmt.Sprintf("'%s'", entry.Request.URL)}
	if len(headers) > 0 {
		args = append(args, "headers=headers")
	}

	// POST data.
	if entry.Request.PostData != nil && entry.Request.PostData.Text != "" {
		escaped := escapePythonString(entry.Request.PostData.Text)
		args = append(args, fmt.Sprintf("data='%s'", escaped))
	}

	sb.WriteString(fmt.Sprintf("response = requests.%s(%s)\n", method, strings.Join(args, ", ")))
	sb.WriteString("print(response.status_code)\n")
	sb.WriteString("print(response.text)\n")

	return sb.String()
}

// buildHeadersDict builds a Python-dictionary-formatted headers string.
func buildHeadersDict(entry *Entries) string {
	if entry == nil {
		return ""
	}
	if len(entry.Request.Headers) == 0 {
		return ""
	}
	var pairs []string
	for _, h := range entry.Request.Headers {
		key := escapePythonString(h.Name)
		val := escapePythonString(h.Value)
		pairs = append(pairs, fmt.Sprintf("'%s': '%s'", key, val))
	}
	return fmt.Sprintf("{%s}", strings.Join(pairs, ", "))
}

// ---------------------------------------------------------------------------
// Postman Collection v2.1 export.
// ---------------------------------------------------------------------------

// PostmanCollection represents the Postman Collection v2.1 format.
type PostmanCollection struct {
	Info PostmanInfo   `json:"info"`
	Item []PostmanItem `json:"item"`
}

// PostmanInfo contains Postman Collection information.
type PostmanInfo struct {
	Name   string `json:"name"`
	Schema string `json:"schema"`
}

// PostmanItem represents a request item in a Postman Collection.
type PostmanItem struct {
	Name    string         `json:"name"`
	Request PostmanRequest `json:"request"`
}

// PostmanRequest defines a Postman request.
type PostmanRequest struct {
	Method string          `json:"method"`
	Header []PostmanHeader `json:"header,omitempty"`
	URL    PostmanURL      `json:"url"`
	Body   *PostmanBody    `json:"body,omitempty"`
}

// PostmanHeader represents a Postman request header.
type PostmanHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// PostmanURL defines a Postman URL.
type PostmanURL struct {
	Raw      string         `json:"raw"`
	Protocol string         `json:"protocol"`
	Host     []string       `json:"host"`
	Path     []string       `json:"path"`
	Query    []PostmanQuery `json:"query,omitempty"`
}

// PostmanQuery represents a Postman query parameter.
type PostmanQuery struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// PostmanBody represents a Postman request body.
type PostmanBody struct {
	Mode string `json:"mode"`
	Raw  string `json:"raw"`
}

// ToPostmanCollection converts HAR data to Postman Collection v2.1 JSON.
func (h *Har) ToPostmanCollection() ([]byte, error) {
	if h == nil {
		return nil, NewInvalidFormatError("HAR object is nil")
	}

	collection := PostmanCollection{
		Info: PostmanInfo{
			Name:   "HAR Export",
			Schema: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
		Item: make([]PostmanItem, 0, len(h.Log.Entries)),
	}

	for i := range h.Log.Entries {
		entry := &h.Log.Entries[i]
		item := entryToPostmanItem(entry)
		collection.Item = append(collection.Item, item)
	}

	return json.MarshalIndent(collection, "", "  ")
}

// SaveAsPostmanCollection saves HAR data as a Postman Collection file.
func (h *Har) SaveAsPostmanCollection(filePath string) error {
	data, err := h.ToPostmanCollection()
	if err != nil {
		return err
	}
	return writeToFile(filePath, data)
}

// entryToPostmanItem converts a HAR entry to a Postman request item.
func entryToPostmanItem(entry *Entries) PostmanItem {
	if entry == nil {
		return PostmanItem{}
	}

	// Parse the URL.
	parsedURL, err := url.Parse(entry.Request.URL)
	name := entry.Request.URL
	if err == nil {
		if parsedURL.Path != "" {
			name = parsedURL.Path
		}
	}

	item := PostmanItem{
		Name: name,
		Request: PostmanRequest{
			Method: entry.Request.Method,
			URL:    buildPostmanURL(entry.Request.URL, parsedURL),
		},
	}

	// Request headers.
	for _, h := range entry.Request.Headers {
		item.Request.Header = append(item.Request.Header, PostmanHeader{
			Key:   h.Name,
			Value: h.Value,
		})
	}

	// Request body.
	if entry.Request.PostData != nil && entry.Request.PostData.Text != "" {
		item.Request.Body = &PostmanBody{
			Mode: "raw",
			Raw:  entry.Request.PostData.Text,
		}
	}

	return item
}

// buildPostmanURL builds a Postman URL structure.
func buildPostmanURL(rawURL string, parsedURL *url.URL) PostmanURL {
	pmURL := PostmanURL{
		Raw: rawURL,
	}

	if parsedURL == nil {
		return pmURL
	}

	pmURL.Protocol = parsedURL.Scheme

	// Split the host.
	host := parsedURL.Host
	if h := strings.Split(host, "."); len(h) > 0 {
		pmURL.Host = h
	}

	// Split the path.
	path := parsedURL.Path
	if path != "" {
		segments := strings.Split(strings.TrimPrefix(path, "/"), "/")
		for _, seg := range segments {
			if seg != "" {
				pmURL.Path = append(pmURL.Path, seg)
			}
		}
	}

	// Query parameters.
	for key, values := range parsedURL.Query() {
		for _, v := range values {
			pmURL.Query = append(pmURL.Query, PostmanQuery{
				Key:   key,
				Value: v,
			})
		}
	}

	return pmURL
}

// ---------------------------------------------------------------------------
// XML export.
// ---------------------------------------------------------------------------

// XMLElement is a helper structure for generating simple XML.
type XMLElement struct {
	XMLName  xml.Name
	Attrs    []xml.Attr   `xml:",any,attr,omitempty"`
	Children []XMLElement `xml:",any,omitempty"`
	Content  string       `xml:",chardata"`
}

// HARXML represents HAR data in XML.
type HARXML struct {
	XMLName xml.Name `xml:"har"`
	Log     LogXML   `xml:"log"`
}

// LogXML represents a Log in XML.
type LogXML struct {
	Version string     `xml:"version"`
	Creator CreatorXML `xml:"creator"`
	Entries []EntryXML `xml:"entries>entry"`
}

// CreatorXML represents a Creator in XML.
type CreatorXML struct {
	Name    string `xml:"name"`
	Version string `xml:"version"`
}

// EntryXML represents Entries in XML.
type EntryXML struct {
	StartedDateTime string      `xml:"startedDateTime"`
	Time            float64     `xml:"time"`
	Request         RequestXML  `xml:"request"`
	Response        ResponseXML `xml:"response"`
}

// RequestXML represents a Request in XML.
type RequestXML struct {
	Method      string       `xml:"method"`
	URL         string       `xml:"url"`
	HTTPVersion string       `xml:"httpVersion"`
	Headers     []HeaderXML  `xml:"headers>header"`
	PostData    *PostDataXML `xml:"postData,omitempty"`
}

// ResponseXML represents a Response in XML.
type ResponseXML struct {
	Status      int         `xml:"status"`
	StatusText  string      `xml:"statusText"`
	HTTPVersion string      `xml:"httpVersion"`
	Headers     []HeaderXML `xml:"headers>header"`
	Content     ContentXML  `xml:"content"`
}

// HeaderXML represents Headers in XML.
type HeaderXML struct {
	Name  string `xml:"name"`
	Value string `xml:"value"`
}

// PostDataXML represents PostData in XML.
type PostDataXML struct {
	MimeType string `xml:"mimeType"`
	Text     string `xml:"text"`
}

// ContentXML represents Content in XML.
type ContentXML struct {
	Size     int    `xml:"size"`
	MimeType string `xml:"mimeType"`
	Text     string `xml:"text,omitempty"`
}

// ToXML converts HAR data to XML.
func (h *Har) ToXML() (string, error) {
	if h == nil {
		return "", NewInvalidFormatError("HAR object is nil")
	}

	harXML := HARXML{
		Log: LogXML{
			Version: h.Log.Version,
			Creator: CreatorXML{
				Name:    h.Log.Creator.Name,
				Version: h.Log.Creator.Version,
			},
			Entries: make([]EntryXML, 0, len(h.Log.Entries)),
		},
	}

	for i := range h.Log.Entries {
		entry := &h.Log.Entries[i]
		entryXML := EntryXML{
			StartedDateTime: entry.StartedDateTime.Format("2006-01-02T15:04:05.000Z"),
			Time:            entry.Time,
			Request: RequestXML{
				Method:      entry.Request.Method,
				URL:         entry.Request.URL,
				HTTPVersion: entry.Request.HTTPVersion,
				Headers:     make([]HeaderXML, 0, len(entry.Request.Headers)),
			},
			Response: ResponseXML{
				Status:      entry.Response.Status,
				StatusText:  entry.Response.StatusText,
				HTTPVersion: entry.Response.HTTPVersion,
				Headers:     make([]HeaderXML, 0, len(entry.Response.Headers)),
				Content: ContentXML{
					Size:     entry.Response.Content.Size,
					MimeType: entry.Response.Content.MimeType,
					Text:     entry.Response.Content.Text,
				},
			},
		}

		// Request headers.
		for _, hdr := range entry.Request.Headers {
			entryXML.Request.Headers = append(entryXML.Request.Headers, HeaderXML{
				Name:  hdr.Name,
				Value: hdr.Value,
			})
		}

		// Response headers.
		for _, hdr := range entry.Response.Headers {
			entryXML.Response.Headers = append(entryXML.Response.Headers, HeaderXML{
				Name:  hdr.Name,
				Value: hdr.Value,
			})
		}

		// POST data.
		if entry.Request.PostData != nil {
			entryXML.Request.PostData = &PostDataXML{
				MimeType: entry.Request.PostData.MimeType,
				Text:     entry.Request.PostData.Text,
			}
		}

		harXML.Log.Entries = append(harXML.Log.Entries, entryXML)
	}

	// HARXML fields are strings, numbers, slices, pointers, or other known XML-serializable types,
	// so xml.MarshalIndent cannot fail.
	data, _ := xml.MarshalIndent(harXML, "", "  ")

	return xml.Header + string(data), nil
}

// SaveAsXML saves HAR data as an XML file.
func (h *Har) SaveAsXML(filePath string) error {
	xmlData, err := h.ToXML()
	if err != nil {
		return err
	}
	return writeToFile(filePath, []byte(xmlData))
}

// ---------------------------------------------------------------------------
// Helper functions.
// ---------------------------------------------------------------------------

// escapeSingleQuotes escapes single quotes in single-quoted shell strings.
// A single quote cannot be escaped inside single quotes; close the quote, add an escaped quote, then reopen it.
func escapeSingleQuotes(s string) string {
	return strings.ReplaceAll(s, "'", "'\\''")
}

// escapePythonString escapes special characters in Python strings.
func escapePythonString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}

// hasAcceptEncoding checks whether the Accept-Encoding request header contains gzip or deflate.
func hasAcceptEncoding(entry *Entries) bool {
	if entry == nil {
		return false
	}
	for _, h := range entry.Request.Headers {
		if strings.EqualFold(h.Name, "Accept-Encoding") {
			val := strings.ToLower(h.Value)
			if strings.Contains(val, "gzip") || strings.Contains(val, "deflate") {
				return true
			}
		}
	}
	return false
}
