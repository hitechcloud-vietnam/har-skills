package har

import "time"

// HARProvider defines the interface implemented by all HAR providers.
type HARProvider interface {
	// GetVersion returns the HAR version.
	GetVersion() string

	// GetCreator returns creator information.
	GetCreator() Creator

	// GetBrowser returns browser information.
	GetBrowser() Browser

	// GetEntries returns all entries.
	GetEntries() []EntryProvider

	// GetPages returns all pages.
	GetPages() []PageProvider

	// converts to a standard Har object.
	ToStandard() *Har
}

// EntryProvider defines the interface for a single entry.
type EntryProvider interface {
	// GetStartedDateTime returns the start time.
	GetStartedDateTime() time.Time

	// GetTime returns the total duration.
	GetTime() float64

	// GetRequest returns request information.
	GetRequest() RequestProvider

	// GetResponse returns response information.
	GetResponse() ResponseProvider

	// GetTimings returns timing information.
	GetTimings() TimingsProvider

	// GetPageref returns the page reference.
	GetPageref() string

	// ToStandard converts to a standard Entry object.
	ToStandard() Entries
}

// RequestProvider defines the interface for a request.
type RequestProvider interface {
	// GetMethod returns the HTTP method.
	GetMethod() string

	// GetURL returns the URL.
	GetURL() string

	// GetHTTPVersion returns the HTTP version.
	GetHTTPVersion() string

	// returns header information.
	GetHeaders() []HeaderProvider

	// GetCookies returns cookie information.
	GetCookies() []CookieProvider

	// GetQueryString returns query parameters.
	GetQueryString() []QueryString

	// GetPostData returns POST data.
	GetPostData() *PostData

	// GetBodySize returns the request body size.
	GetBodySize() int

	// GetHeadersSize returns the header size.
	GetHeadersSize() int

	// converts to a standard Request object.
	ToStandard() Request
}

// ResponseProvider defines the interface for a response.
type ResponseProvider interface {
	// GetStatus returns the status code.
	GetStatus() int

	// GetStatusText returns the status text.
	GetStatusText() string

	// GetHTTPVersion returns the HTTP version.
	GetHTTPVersion() string

	// returns header information.
	GetHeaders() []HeaderProvider

	// GetCookies returns cookie information.
	GetCookies() []CookieProvider

	// returns the content.
	GetContent() ContentProvider

	// GetBodySize returns the response body size.
	GetBodySize() int

	// GetHeadersSize returns the header size.
	GetHeadersSize() int

	// converts to a standard Response object.
	ToStandard() Response
}

// HeaderProvider defines the interface for an HTTP header.
type HeaderProvider interface {
	// GetName returns the name.
	GetName() string

	// returns the value.
	GetValue() string

	// converts to a standard Header object.
	ToStandard() Headers
}

// CookieProvider defines the interface for a cookie.
type CookieProvider interface {
	// GetName returns the name.
	GetName() string

	// returns the value.
	GetValue() string

	// returns the domain.
	GetDomain() string

	// returns the path.
	GetPath() string

	// returns the expiration time.
	GetExpires() time.Time

	// reports whether the cookie is HttpOnly.
	IsHTTPOnly() bool

	// reports whether the cookie is secure.
	IsSecure() bool

	// GetSameSite returns the SameSite value.
	GetSameSite() string

	// converts to a standard Cookie object.
	ToStandard() Cookie
}

// ContentProvider defines the interface for content.
type ContentProvider interface {
	// returns the size.
	GetSize() int

	// GetMimeType returns the MIME type.
	GetMimeType() string

	// returns the text content, if available.
	GetText() string

	// returns the encoding, if available.
	GetEncoding() string

	// GetCompression returns the number of bytes saved by compression.
	GetCompression() int

	// converts to a standard Content object.
	ToStandard() Content
}

// TimingsProvider defines the interface for timing information.
type TimingsProvider interface {
	// returns the blocked duration.
	GetBlocked() float64

	// returns the DNS resolution duration.
	GetDNS() float64

	// GetConnect returns the connection duration.
	GetConnect() float64

	// GetSend returns the send duration.
	GetSend() float64

	// GetWait returns the wait duration.
	GetWait() float64

	// GetReceive returns the receive duration.
	GetReceive() float64

	// GetSSL returns the SSL handshake duration.
	GetSSL() float64

	// converts to standard Timings.
	ToStandard() Timings
}

// PageProvider defines the interface for a page.
type PageProvider interface {
	// GetID returns the ID.
	GetID() string

	// GetTitle returns the title.
	GetTitle() string

	// GetStartedDateTime returns the start time.
	GetStartedDateTime() time.Time

	// GetPageTimings returns page timing information.
	GetPageTimings() PageTimingsProvider

	// converts to a standard Page object.
	ToStandard() Pages
}

// PageTimingsProvider defines the interface for page timing data.
type PageTimingsProvider interface {
	// GetOnContentLoad returns the content load time.
	GetOnContentLoad() float64

	// GetOnLoad returns the page load time.
	GetOnLoad() float64

	// ToStandard converts to a standard PageTimings object.
	ToStandard() PageTimings
}
