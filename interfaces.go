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

	// ToStandard 转换为标准HAR对象
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

	// GetHeaders 获取头部信息
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

	// ToStandard 转换为标准Request对象
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

	// GetHeaders 获取头部信息
	GetHeaders() []HeaderProvider

	// GetCookies returns cookie information.
	GetCookies() []CookieProvider

	// GetContent 获取内容
	GetContent() ContentProvider

	// GetBodySize returns the response body size.
	GetBodySize() int

	// GetHeadersSize returns the header size.
	GetHeadersSize() int

	// ToStandard 转换为标准Response对象
	ToStandard() Response
}

// HeaderProvider 定义HTTP头部的接口
type HeaderProvider interface {
	// GetName returns the name.
	GetName() string

	// GetValue 获取值
	GetValue() string

	// ToStandard 转换为标准Header对象
	ToStandard() Headers
}

// CookieProvider defines the interface for a cookie.
type CookieProvider interface {
	// GetName returns the name.
	GetName() string

	// GetValue 获取值
	GetValue() string

	// GetDomain 获取域
	GetDomain() string

	// GetPath 获取路径
	GetPath() string

	// GetExpires 获取过期时间
	GetExpires() time.Time

	// IsHTTPOnly 是否为HTTPOnly
	IsHTTPOnly() bool

	// IsSecure 是否为Secure
	IsSecure() bool

	// GetSameSite returns the SameSite value.
	GetSameSite() string

	// ToStandard 转换为标准Cookie对象
	ToStandard() Cookie
}

// ContentProvider defines the interface for content.
type ContentProvider interface {
	// GetSize 获取大小
	GetSize() int

	// GetMimeType returns the MIME type.
	GetMimeType() string

	// GetText 获取文本内容（如果有）
	GetText() string

	// GetEncoding 获取编码（如果有）
	GetEncoding() string

	// GetCompression returns the number of bytes saved by compression.
	GetCompression() int

	// ToStandard 转换为标准Content对象
	ToStandard() Content
}

// TimingsProvider 定义计时信息的接口
type TimingsProvider interface {
	// GetBlocked 获取被阻塞时间
	GetBlocked() float64

	// GetDNS 获取DNS解析时间
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

	// ToStandard 转换为标准Timings对象
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

	// ToStandard 转换为标准Page对象
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
