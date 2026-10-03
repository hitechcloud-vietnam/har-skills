# Go-HAR Data Structures in Detail

This document describes the HTTP Archive (HAR) data structures and their implementation in Go-HAR.

## HAR File Overview

The HAR (HTTP Archive) file format stores HTTP request and response data as JSON. Browser developer tools export HAR files to analyze web performance, debug issues, and record network activity.

## Core Structures

### Har Structure

`Har` is the root structure of a HAR file and contains a single `Log` field:

```go
type Har struct {
    Log Log `json:"log"`
}
```

### Log Structure

`Log` is the main container for all HAR data:

```go
type Log struct {
    Version string   `json:"version"`     // HAR format version, usually "1.2"
    Creator Creator  `json:"creator"`     // Information about the tool that created the HAR file
    Browser Browser  `json:"browser,omitempty"` // Browser used for capture (optional)
    Pages   []Pages  `json:"pages,omitempty"`   // Captured page information (optional)
    Entries []Entries `json:"entries"`     // All HTTP request and response entries
    Comment string   `json:"comment,omitempty"` // User-provided comment (optional)
}
```

Field details:
- `Version`: HAR format version; the specification version is "1.2".
- `Creator`: Information about the application that created the HAR file.
- `Browser`: Information about the browser that generated the requests.
- `Pages`: Information about the recorded pages.
- `Entries`: All HTTP request and response entries.
- `Comment`: Optional user comment.

### Creator and Browser Structures

These structures describe the creator tool and browser:

```go
type Creator struct {
    Name    string `json:"name"`    // Application name
    Version string `json:"version"` // Application version
    Comment string `json:"comment,omitempty"` // Comment (optional)
}

type Browser struct {
    Name    string `json:"name"`    // Browser name
    Version string `json:"version"` // Browser version
    Comment string `json:"comment,omitempty"` // Comment (optional)
}
```

### Pages Structure

Describes information about a captured page:

```go
type Pages struct {
    StartedDateTime time.Time   `json:"startedDateTime"` // Page load start time (ISO 8601)
    ID              string      `json:"id"`              // Unique page identifier
    Title           string      `json:"title"`           // Page title
    PageTimings     PageTimings `json:"pageTimings"`     // Page load timing information
    Comment         string      `json:"comment,omitempty"` // Comment (optional)
}

type PageTimings struct {
    OnContentLoad float64 `json:"onContentLoad,omitempty"` // DOMContentLoaded event in milliseconds; -1 means unavailable
    OnLoad        float64 `json:"onLoad,omitempty"`        // load event in milliseconds; -1 means unavailable
    Comment       string  `json:"comment,omitempty"`       // Comment (optional)
}
```

### Entries Structure

`Entries` is the most important part of a HAR file and contains complete HTTP request and response information:

```go
type Entries struct {
    Pageref         string     `json:"pageref,omitempty"` // Referenced page ID
    StartedDateTime time.Time  `json:"startedDateTime"`   // Request start time (ISO 8601)
    Time            float64    `json:"time"`              // Total request duration (milliseconds)
    Request         Request    `json:"request"`           // Request information
    Response        Response   `json:"response"`          // Response information
    Cache           Cache      `json:"cache"`             // Cache information
    Timings         Timings    `json:"timings"`           // Timing for each request phase
    ServerIPAddress string     `json:"serverIPAddress,omitempty"` // Server IP address
    Connection      string     `json:"connection,omitempty"`      // Connection information
    Comment         string     `json:"comment,omitempty"`         // Comment.
}
```

Field details:
- `Pageref`: Reference to the associated page, used to identify which page a request belongs to.
- `StartedDateTime`: Exact request start time in ISO 8601 format.
- `Time`: Total request duration in milliseconds, from request initiation through completion of the response.
- `Request`: Complete HTTP request information.
- `Response`: Complete HTTP response information.
- `Cache`: Browser cache state information.
- `Timings`: Detailed duration for each request phase.
- `ServerIPAddress`: Optional server IP address.
- `Connection`: Optional connection identifier, such as "52492".

### Request Structure

Detailed information about an HTTP request:

```go
type Request struct {
    Method      string     `json:"method"`      // HTTP method (GET, POST, etc.)
    URL         string     `json:"url"`         // Full URL
    HTTPVersion string     `json:"httpVersion"` // HTTP version
    Cookies     []Cookie   `json:"cookies"`     // Cookie information
    Headers     []Headers  `json:"headers"`     // Request headers
    QueryString []QueryString `json:"queryString"` // URL query parameters
    PostData    PostData   `json:"postData,omitempty"` // POST data (optional)
    HeadersSize int        `json:"headersSize"` // Request header size (bytes)
    BodySize    int        `json:"bodySize"`    // Request body size (bytes)
    Comment     string     `json:"comment,omitempty"`  // Comment (optional)
}
```

### Response Structure

Detailed information about an HTTP response:

```go
type Response struct {
    Status       int       `json:"status"`       // HTTP status code
    StatusText   string    `json:"statusText"`   // Status text
    HTTPVersion  string    `json:"httpVersion"`  // HTTP version
    Cookies      []Cookie  `json:"cookies"`      // Cookie information
    Headers      []Headers `json:"headers"`      // Response headers
    Content      Content   `json:"content"`      // Response content
    RedirectURL  string    `json:"redirectURL"`  // Redirect URL
    HeadersSize  int       `json:"headersSize"`  // Response header size (bytes)
    BodySize     int       `json:"bodySize"`     // Response body size (bytes)
    TransferSize int       `json:"_transferSize,omitempty"` // Transfer size (bytes)
    Error        string    `json:"_error,omitempty"`        // Error information
    Comment      string    `json:"comment,omitempty"`       // Comment (optional)
}
```

### Headers and Cookie Structures

HTTP header and Cookie information:

```go
type Headers struct {
    Name    string `json:"name"`    // Header name
    Value   string `json:"value"`   // Header value
    Comment string `json:"comment,omitempty"` // Comment (optional)
}

type Cookie struct {
    Name     string    `json:"name"`     // Cookie name
    Value    string    `json:"value"`    // Cookie value
    Path     string    `json:"path,omitempty"`     // Path
    Domain   string    `json:"domain,omitempty"`   // Domain
    Expires  time.Time `json:"expires,omitempty"`  // Expiration time
    HTTPOnly bool      `json:"httpOnly,omitempty"` // Whether HttpOnly
    Secure   bool      `json:"secure,omitempty"`   // Whether secure
    SameSite string    `json:"sameSite,omitempty"` // SameSite attribute
    Comment  string    `json:"comment,omitempty"`  // Comment (optional)
}
```

### QueryString and PostData Structures

URL query parameters and POST data:

```go
type QueryString struct {
    Name    string `json:"name"`    // Parameter name
    Value   string `json:"value"`   // Parameter value
    Comment string `json:"comment,omitempty"` // Comment (optional)
}

type PostData struct {
    MimeType string    `json:"mimeType"` // MIME type
    Params   []Params  `json:"params"`   // Parameters (for forms)
    Text     string    `json:"text"`     // Text content
    Comment  string    `json:"comment,omitempty"` // Comment (optional)
}

type Params struct {
    Name        string  `json:"name"`        // Parameter name
    Value       string  `json:"value,omitempty"` // Parameter value
    FileName    string  `json:"fileName,omitempty"` // File name (for file uploads)
    ContentType string  `json:"contentType,omitempty"` // Content type
    Comment     string  `json:"comment,omitempty"` // Comment (optional)
}
```

### Content Structure

HTTP response content:

```go
type Content struct {
    Size        int    `json:"size"`        // Content size (bytes)
    MimeType    string `json:"mimeType"`    // MIME type
    Compression int    `json:"compression,omitempty"` // Compressed size (bytes)
    Text        string `json:"text,omitempty"`        // Actual content (optional)
    Encoding    string `json:"encoding,omitempty"`    // Encoding (such as base64)
    Comment     string `json:"comment,omitempty"`     // Comment (optional)
}
```

### Cache Structure

Browser cache information:

```go
type Cache struct {
    BeforeRequest *BeforeRequest `json:"beforeRequest,omitempty"` // Cache state before the request
    AfterRequest  *AfterRequest  `json:"afterRequest,omitempty"`  // Cache state after the request
    Comment       string         `json:"comment,omitempty"`       // Comment (optional)
}

type BeforeRequest struct {
    Expires    time.Time `json:"expires,omitempty"`    // Expiration time
    LastAccess time.Time `json:"lastAccess"`           // Last access time.
    ETag       string    `json:"eTag"`                 // ETag
    HitCount   int       `json:"hitCount"`             // Hit count
    Comment    string    `json:"comment,omitempty"`    // Comment (optional)
}

type AfterRequest struct {
    Expires    time.Time `json:"expires,omitempty"`    // Expiration time
    LastAccess time.Time `json:"lastAccess"`           // Last access time.
    ETag       string    `json:"eTag"`                 // ETag
    HitCount   int       `json:"hitCount"`             // Hit count
    Comment    string    `json:"comment,omitempty"`    // Comment (optional)
}
```

### Timings Structure

Timing information for each request phase:

```go
type Timings struct {
    Blocked float64 `json:"blocked"`             // Blocked time
    DNS     float64 `json:"dns"`                 // DNS resolution time
    Connect float64 `json:"connect"`             // TCP connection time
    Send    float64 `json:"send"`                // Request send time
    Wait    float64 `json:"wait"`                // Response wait time
    Receive float64 `json:"receive"`             // Response receive time
    Ssl     float64 `json:"ssl,omitempty"`       // SSL/TLS negotiation time
    Comment string  `json:"comment,omitempty"`   // Comment (optional)
}
```

Field details:
- `Blocked`: Time spent blocked, for example while waiting for an available TCP connection.
- `DNS`: Time spent on DNS resolution; a value of -1 means DNS was cached or not applicable.
- `Connect`: Time spent establishing a TCP connection; a value of -1 means a reused connection or that the phase did not apply.
- `Send`: Time spent sending the HTTP request to the server.
- `Wait`: Time spent waiting for the first response byte from the server.
- `Receive`: Time spent receiving response data.
- `Ssl`: Time spent on SSL/TLS negotiation; applies only to HTTPS requests.

## Optimized Go-HAR Structures

In addition to the standard HAR structures, Go-HAR provides several optimized structures for different use cases:

### OptimizedHar

Memory-optimized version for processing large HAR files:

```go
type OptimizedHar struct {
    Log struct {
        Version string
        Creator Creator
        Pages   []Pages
        Entries []OptimizedEntries
    }
}

type OptimizedEntries struct {
    StartedDateTime time.Time
    Time            float64
    PageRef         *string              // Uses a pointer to allow nil values
    Request         OptimizedRequest
    Response        OptimizedResponse
    Timings         OptimizedTimings
}

type OptimizedRequest struct {
    Method      HTTPMethod        // Uses an enum instead of a string
    URL         string
    HTTPVersion string
    Cookies     []Cookie
    Headers     map[string]string // Uses a map instead of an array for faster lookups
    QueryString map[string]string // Uses a map instead of an array
    HeadersSize *int              // Uses a pointer to allow nil values
    BodySize    *int              // Uses a pointer to allow nil values
}
```

Key optimizations:
- Uses enums instead of strings.
- Uses maps instead of arrays to improve lookup performance.
- Uses pointers for optional values to avoid unnecessary memory use.

### LazyHar

Lazy-loading version for scenarios where large content should be loaded on demand:

```go
type LazyHar struct {
    Log struct {
        Version string
        Creator Creator
        Pages   []Pages
        Entries []LazyEntries
    }
}

type LazyEntries struct {
    StartedDateTime time.Time
    Time            float64
    Pageref         string
    Request         Request
    Response        LazyResponse
    Timings         Timings
}

type LazyResponse struct {
    // Basic fields are loaded directly.
    Status       int
    StatusText   string
    HTTPVersion  string
    // Content is loaded lazily.
    Content      *LazyContent
    // Other fields...
}

type LazyContent struct {
    // Basic information is loaded directly.
    Size        int
    MimeType    string
    // Large content is loaded lazily.
    Text        *string
    Encoding    *string
    // Internal state.
    loaded      bool
    dataSource  ContentDataSource
}
```

Key features:
- Basic information (such as size and type) is loaded directly.
- Large content (such as response bodies) is loaded only when needed.
- Provides a unified interface with usage consistent with the standard structures.

## Interfaces

Go-HAR defines a set of interfaces to ensure interoperability among different implementations:

```go
// HARProvider is the common interface for all HAR implementations.
type HARProvider interface {
    GetVersion() string
    GetCreator() Creator
    GetEntries() []EntryProvider
    GetPages() []PageProvider
    ToStandard() *Har
}

// EntryProvider is the interface for an individual entry.
type EntryProvider interface {
    GetStartedDateTime() time.Time
    GetTime() float64
    GetRequest() RequestProvider
    GetResponse() ResponseProvider
    GetTimings() TimingsProvider
    GetPageref() string
    ToStandard() Entries
}

// Other interfaces include RequestProvider, ResponseProvider, and ContentProvider.
```

These interfaces let standard, optimized, lazy-loading, and streaming HAR implementations be used consistently, improving code reuse and flexibility.

## Best Practices

### Choose the Appropriate Parsing Mode

- **Small HAR files**: Use standard parsing.
  ```go
  har, err := har.ParseFile("small.har")
  ```

- **Large HAR files**: Use memory-optimized mode to reduce memory use.
  ```go
  har, err := har.ParseFile("large.har", har.WithMemoryOptimized())
  ```

- **HAR files with large responses**: Use lazy loading to defer loading large content.
  ```go
  har, err := har.ParseFile("large_content.har", har.WithLazyLoading())
  ```

- **Very large HAR files**: Use streaming parsing to process entries one at a time.
  ```go
  iterator, err := har.NewStreamingParserFromFile("huge.har")
  ```

### Handle Errors

Always check parsing errors and consider using enhanced error handling:

```go
result, err := har.ParseHarFileWithWarnings("problematic.har")
if err != nil {
    log.Fatalf("Parsing failed completely: %v", err)
} else if len(result.Warnings) > 0 {
    log.Printf("Parsing succeeded with %d warnings", len(result.Warnings))
    for _, w := range result.Warnings {
        log.Printf("Warning: %s", w.Error())
    }
}
```

### Program to Interfaces

Write functions to accept interface types rather than concrete implementations:

```go
// Good: accept any type that implements HARProvider.
func ProcessHAR(harFile har.HARProvider) {
    // Processing logic.
}

// Not recommended: accept only the concrete type.
func ProcessHAR(harFile *har.Har) {
    // Processing logic.
}
```

## Performance Considerations

- **Memory use**: Standard parsing may exhaust memory with gigabyte-scale HAR files; consider streaming parsing.
- **Parsing speed**: Skipping validation can improve parsing speed but may miss format issues.
- **Lazy-loading trade-off**: Lazy loading reduces initial memory use, but accessing content later may incur a performance cost.