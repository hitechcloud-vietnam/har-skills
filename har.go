package har

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"time"
)

// Error definitions.
var (
	// ErrInvalidHar indicates that the HAR object is missing required fields.
	ErrInvalidHar = NewValidationError("HAR object is missing required fields", "")

	// ErrInvalidURL indicates that a HAR entry contains an invalid URL.
	ErrInvalidURL = NewValidationError("HAR entry contains an invalid URL", "")

	// ErrNotJsonContent indicates that the content is not JSON.
	ErrNotJsonContent = NewInvalidFormatError("content is not JSON")
)

// ParseHarFile parses a file in HAR format.
//
// ParseHarFile is a convenience wrapper around ParseHar that reads the file before parsing it.
// It follows error-handling best practices by converting all errors to HarError for consistent handling.
//
// Example:
//
//	har, err := ParseHarFile("example.har")
//	if err != nil {
//	    log.Fatalf("failed to parse HAR file: %v", err)
//	}
func ParseHarFile(harFilePath string) (*Har, error) {
	harFileBytes, err := os.ReadFile(harFilePath)
	if err != nil {
		return nil, NewFileSystemError(fmt.Sprintf("unable to read file '%s'", harFilePath), err)
	}
	return ParseHar(harFileBytes)
}

// ParseHar parses byte data in HAR format.
//
// ParseHar converts byte data in HAR format into a Har struct.
// It performs full validation to ensure the Har object conforms to the specification.
//
// Example:
//
//	harBytes, _ := ioutil.ReadFile("example.har")
//	har, err := ParseHar(harBytes)
//	if err != nil {
//	    log.Fatalf("failed to parse HAR data: %v", err)
//	}
func ParseHar(harFileBytes []byte) (*Har, error) {
	// Check for empty input.
	if len(harFileBytes) == 0 {
		return nil, NewInvalidFormatError("input is empty")
	}

	// Check that the input is JSON.
	if !isJSONContent(harFileBytes) {
		return nil, ErrNotJsonContent
	}

	// Parse the JSON.
	har := new(Har)
	err := json.Unmarshal(harFileBytes, har)
	if err != nil {
		return nil, WrapJSONUnmarshalError(err)
	}

	// Validate the HAR object.
	if err := ValidateHarFile(har); err != nil {
		return nil, err
	}

	return har, nil
}

// Har represents the root structure of an HTTP Archive (HAR) file.
//
// Har is the root object in HAR format and contains a Log field.
// All HAR data is stored in Log.
type Har struct {
	Log          Log          `json:"log"` // HAR log object.
	CustomFields CustomFields `json:"-"`
}

// Log represents the HAR log object.
//
// Log contains the main HAR data, including the version, creator information,
// page information, and HTTP entries.
type Log struct {
	Version      string       `json:"version"`           // HAR specification version.
	Creator      Creator      `json:"creator"`           // Information about the creating tool.
	Browser      Browser      `json:"browser,omitempty"` // Browser information (optional).
	Pages        []Pages      `json:"pages,omitempty"`   // Page information.
	Entries      []Entries    `json:"entries"`           // HTTP request/response entries.
	Comment      string       `json:"comment,omitempty"` // Optional comment.
	CustomFields CustomFields `json:"-"`
}

// Creator represents information about the tool that created the HAR file.
type Creator struct {
	Name    string `json:"name"`              // Name of the creating tool.
	Version string `json:"version"`           // Version of the creating tool.
	Comment string `json:"comment,omitempty"` // Optional comment.
}

// Browser represents browser information.
type Browser struct {
	Name    string `json:"name"`              // Browser name.
	Version string `json:"version"`           // Browser version.
	Comment string `json:"comment,omitempty"` // Optional comment.
}

// PageTimings represents page load timings.
type PageTimings struct {
	OnContentLoad float64 `json:"onContentLoad"`     // Time when the DOMContentLoaded event fired (ms).
	OnLoad        float64 `json:"onLoad"`            // Time when the load event fired (ms).
	Comment       string  `json:"comment,omitempty"` // Optional comment.
}

// Pages represents page information in a HAR file.
type Pages struct {
	StartedDateTime time.Time    `json:"startedDateTime"`   // Page load start time.
	ID              string       `json:"id"`                // Unique page identifier.
	Title           string       `json:"title"`             // Page title.
	PageTimings     PageTimings  `json:"pageTimings"`       // Page load timings.
	Comment         string       `json:"comment,omitempty"` // Optional comment.
	CustomFields    CustomFields `json:"-"`
}

// Headers represents an HTTP header.
type Headers struct {
	Name    string `json:"name"`              // Header name.
	Value   string `json:"value"`             // Header value.
	Comment string `json:"comment,omitempty"` // Optional comment.
}

// QueryString represents a URL query parameter.
type QueryString struct {
	Name    string `json:"name"`              // Parameter name.
	Value   string `json:"value"`             // Parameter value.
	Comment string `json:"comment,omitempty"` // Optional comment.
}

// Cookie represents an HTTP cookie.
type Cookie struct {
	Name         string       `json:"name"`               // Cookie name.
	Value        string       `json:"value"`              // Cookie value.
	Path         string       `json:"path,omitempty"`     // Cookie path.
	Domain       string       `json:"domain,omitempty"`   // Cookie domain.
	Expires      time.Time    `json:"expires,omitempty"`  // Expiration time.
	HTTPOnly     bool         `json:"httpOnly,omitempty"` // Whether HttpOnly is enabled.
	Secure       bool         `json:"secure,omitempty"`   // Whether Secure is enabled.
	SameSite     string       `json:"sameSite,omitempty"` // SameSite policy.
	Comment      string       `json:"comment,omitempty"`  // Optional comment.
	CustomFields CustomFields `json:"-"`
}

// PostData represents POST data in an HTTP request.
type PostData struct {
	MimeType     string       `json:"mimeType"`          // MIME type.
	Params       []Param      `json:"params,omitempty"`  // Parameter list (used for form submissions).
	Text         string       `json:"text,omitempty"`    // Request body text.
	Comment      string       `json:"comment,omitempty"` // Optional comment.
	CustomFields CustomFields `json:"-"`
}

// Param represents a form parameter in a POST request.
type Param struct {
	Name         string       `json:"name"`                  // Parameter name.
	Value        string       `json:"value,omitempty"`       // Parameter value.
	FileName     string       `json:"fileName,omitempty"`    // File name (for file uploads).
	ContentType  string       `json:"contentType,omitempty"` // Content type.
	Comment      string       `json:"comment,omitempty"`     // Optional comment.
	CustomFields CustomFields `json:"-"`
}

// Content represents HTTP response content.
type Content struct {
	Size         int          `json:"size"`                  // Content size (bytes).
	MimeType     string       `json:"mimeType"`              // MIME type.
	Compression  int          `json:"compression,omitempty"` // Bytes saved by compression (optional).
	Text         string       `json:"text,omitempty"`        // Text content (optional).
	Encoding     string       `json:"encoding,omitempty"`    // Encoding (optional, e.g. base64).
	Comment      string       `json:"comment,omitempty"`     // Optional comment.
	CustomFields CustomFields `json:"-"`
}

// Request represents an HTTP request.
type Request struct {
	Method       string        `json:"method"`             // HTTP method (GET, POST, etc.).
	URL          string        `json:"url"`                // Request URL.
	HTTPVersion  string        `json:"httpVersion"`        // HTTP version.
	Cookies      []Cookie      `json:"cookies"`            // Cookies.
	Headers      []Headers     `json:"headers"`            // Headers.
	QueryString  []QueryString `json:"queryString"`        // Query parameters.
	PostData     *PostData     `json:"postData,omitempty"` // POST data (optional).
	HeadersSize  int           `json:"headersSize"`        // Header size (bytes).
	BodySize     int           `json:"bodySize"`           // Request body size (bytes).
	Comment      string        `json:"comment,omitempty"`  // Optional comment.
	CustomFields CustomFields  `json:"-"`
}

// Response represents an HTTP response.
type Response struct {
	Status       int          `json:"status"`                  // Status code.
	StatusText   string       `json:"statusText"`              // Status description.
	HTTPVersion  string       `json:"httpVersion"`             // HTTP version.
	Cookies      []Cookie     `json:"cookies"`                 // Cookies.
	Headers      []Headers    `json:"headers"`                 // Headers.
	Content      Content      `json:"content"`                 // Response content.
	RedirectURL  string       `json:"redirectURL"`             // Redirect URL.
	HeadersSize  int          `json:"headersSize"`             // Header size (bytes).
	BodySize     int          `json:"bodySize"`                // Response body size (bytes).
	TransferSize int          `json:"_transferSize,omitempty"` // Transfer size (Chrome extension).
	Error        any          `json:"_error,omitempty"`        // Error information (Chrome extension).
	Comment      string       `json:"comment,omitempty"`       // Optional comment.
	CustomFields CustomFields `json:"-"`
}

// BeforeRequest represents cache state before the request.
type BeforeRequest struct {
	Expires      time.Time    `json:"expires,omitempty"` // Expiration time.
	LastAccess   time.Time    `json:"lastAccess"`        // Last access time.
	ETag         string       `json:"eTag"`              // ETag.
	HitCount     int          `json:"hitCount"`          // Number of cache hits.
	Comment      string       `json:"comment,omitempty"` // Optional comment.
	CustomFields CustomFields `json:"-"`
}

// AfterRequest represents cache state after the request.
type AfterRequest struct {
	Expires      time.Time    `json:"expires,omitempty"` // Expiration time.
	LastAccess   time.Time    `json:"lastAccess"`        // Last access time.
	ETag         string       `json:"eTag"`              // ETag.
	HitCount     int          `json:"hitCount"`          // Number of cache hits.
	Comment      string       `json:"comment,omitempty"` // Optional comment.
	CustomFields CustomFields `json:"-"`
}

// Cache represents HTTP cache information.
type Cache struct {
	BeforeRequest *BeforeRequest `json:"beforeRequest,omitempty"` // Cache state before the request.
	AfterRequest  *AfterRequest  `json:"afterRequest,omitempty"`  // Cache state after the request.
	Comment       string         `json:"comment,omitempty"`       // Comment.
	CustomFields  CustomFields   `json:"-"`
}

// Timings represents timing measurements during an HTTP request/response.
type Timings struct {
	Blocked         float64      `json:"blocked"`                     // Blocked time (ms).
	DNS             float64      `json:"dns"`                         // DNS lookup time (ms).
	Connect         float64      `json:"connect"`                     // TCP connection time (ms).
	Ssl             float64      `json:"ssl"`                         // SSL/TLS negotiation time (ms).
	Send            float64      `json:"send"`                        // Request send time (ms).
	Wait            float64      `json:"wait"`                        // Response wait time (ms).
	Receive         float64      `json:"receive"`                     // Response receive time (ms).
	BlockedQueueing float64      `json:"_blocked_queueing,omitempty"` // Queue blocking time (Chrome extension, ms).
	BlockedProxy    float64      `json:"_blocked_proxy,omitempty"`    // Proxy blocking time (Chrome extension, ms).
	Comment         string       `json:"comment,omitempty"`           // Optional comment.
	CustomFields    CustomFields `json:"-"`
}

// Entries represents a single HTTP request/response entry in a HAR file.
type Entries struct {
	StartedDateTime time.Time    `json:"startedDateTime"`           // Request start time.
	Time            float64      `json:"time"`                      // Total duration (ms).
	Request         Request      `json:"request"`                   // Request information.
	Response        Response     `json:"response"`                  // Response information.
	Cache           Cache        `json:"cache"`                     // Cache information.
	Timings         Timings      `json:"timings"`                   // Detailed timings.
	Pageref         string       `json:"pageref,omitempty"`         // Associated page ID.
	ServerIPAddress string       `json:"serverIPAddress,omitempty"` // Server IP address.
	Connection      string       `json:"connection,omitempty"`      // Connection ID.
	Initiator       Initiator    `json:"_initiator,omitempty"`      // Request initiator (Chrome extension).
	Priority        string       `json:"_priority,omitempty"`       // Request priority (Chrome extension).
	ResourceType    string       `json:"_resourceType,omitempty"`   // Resource type (Chrome extension).
	Comment         string       `json:"comment,omitempty"`         // Optional comment.
	CustomFields    CustomFields `json:"-"`
}

// Initiator represents the request initiator (Chrome DevTools extension).
type Initiator struct {
	Type       string `json:"type"`       // Initiator type.
	URL        string `json:"url"`        // Initiator URL.
	LineNumber int    `json:"lineNumber"` // Source line number.
	Stack      Stack  `json:"stack"`      // Call stack.
}

// Stack represents a call stack (Chrome DevTools extension).
type Stack struct {
	CallFrames []CallFrame `json:"callFrames"` // Call frames.
	Parent     Parent      `json:"parent"`     // Parent call stack.
}

// Parent represents the parent call stack (Chrome DevTools extension).
type Parent struct {
	Parent      *Parent     `json:"parent"`      // Nested parent.
	Description string      `json:"description"` // Description.
	CallFrames  []CallFrame `json:"callFrames"`  // Call frames.
	ParentID    ParentID    `json:"parentId"`    // Parent ID.
}

// ParentID represents the parent ID (Chrome DevTools extension).
type ParentID struct {
	ID         string `json:"id"`         // ID
	DebuggerID string `json:"debuggerId"` // Debugger ID.
}

// CallFrame represents a call frame (Chrome DevTools extension).
type CallFrame struct {
	FunctionName string `json:"functionName"` // Function name.
	ScriptID     string `json:"scriptId"`     // Script ID.
	URL          string `json:"url"`          // URL
	LineNumber   int    `json:"lineNumber"`   // Line number.
	ColumnNumber int    `json:"columnNumber"` // Column number.
}

// IsValidURL reports whether a URL is valid.
//
// It checks whether the given URL string conforms to URL syntax.
// It returns true if the URL is valid and false otherwise.
func IsValidURL(rawURL string) bool {
	_, err := url.Parse(rawURL)
	return err == nil
}
