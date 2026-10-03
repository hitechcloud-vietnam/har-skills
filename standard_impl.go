package har

import "time"

// Implements the interfaces for standard HAR types.

// GetVersion implements the HARProvider interface.
func (h *Har) GetVersion() string {
	if h == nil {
		return ""
	}
	return h.Log.Version
}

// GetCreator implements the HARProvider interface.
func (h *Har) GetCreator() Creator {
	if h == nil {
		return Creator{}
	}
	return h.Log.Creator
}

// GetBrowser implements the HARProvider interface.
func (h *Har) GetBrowser() Browser {
	if h == nil {
		return Browser{}
	}
	return h.Log.Browser
}

// GetEntries implements the HARProvider interface.
func (h *Har) GetEntries() []EntryProvider {
	if h == nil {
		return nil
	}
	providers := make([]EntryProvider, len(h.Log.Entries))
	for i := range h.Log.Entries {
		providers[i] = &h.Log.Entries[i]
	}
	return providers
}

// GetPages implements the HARProvider interface.
func (h *Har) GetPages() []PageProvider {
	if h == nil {
		return nil
	}
	providers := make([]PageProvider, len(h.Log.Pages))
	for i := range h.Log.Pages {
		providers[i] = &h.Log.Pages[i]
	}
	return providers
}

// ToStandard implements the HARProvider interface.
func (h *Har) ToStandard() *Har {
	if h == nil {
		return nil
	}
	return h
}

// Entries interface implementation.

// GetStartedDateTime implements the EntryProvider interface.
func (e *Entries) GetStartedDateTime() time.Time {
	if e == nil {
		return time.Time{}
	}
	return e.StartedDateTime
}

// GetTime implements the EntryProvider interface.
func (e *Entries) GetTime() float64 {
	if e == nil {
		return 0
	}
	return e.Time
}

// GetRequest implements the EntryProvider interface.
func (e *Entries) GetRequest() RequestProvider {
	if e == nil {
		return nil
	}
	return &e.Request
}

// GetResponse implements the EntryProvider interface.
func (e *Entries) GetResponse() ResponseProvider {
	if e == nil {
		return nil
	}
	return &e.Response
}

// GetTimings implements the EntryProvider interface.
func (e *Entries) GetTimings() TimingsProvider {
	if e == nil {
		return nil
	}
	return &e.Timings
}

// GetPageref implements the EntryProvider interface.
func (e *Entries) GetPageref() string {
	if e == nil {
		return ""
	}
	return e.Pageref
}

// ToStandard implements the EntryProvider interface.
func (e *Entries) ToStandard() Entries {
	if e == nil {
		return Entries{}
	}
	return *e
}

// Request interface implementation.

// GetMethod implements the RequestProvider interface.
func (r *Request) GetMethod() string {
	if r == nil {
		return ""
	}
	return r.Method
}

// GetURL implements the RequestProvider interface.
func (r *Request) GetURL() string {
	if r == nil {
		return ""
	}
	return r.URL
}

// GetHTTPVersion implements the RequestProvider interface.
func (r *Request) GetHTTPVersion() string {
	if r == nil {
		return ""
	}
	return r.HTTPVersion
}

// GetHeaders implements the RequestProvider interface.
func (r *Request) GetHeaders() []HeaderProvider {
	if r == nil {
		return nil
	}
	providers := make([]HeaderProvider, len(r.Headers))
	for i := range r.Headers {
		providers[i] = &r.Headers[i]
	}
	return providers
}

// GetCookies implements the RequestProvider interface.
func (r *Request) GetCookies() []CookieProvider {
	if r == nil {
		return nil
	}
	providers := make([]CookieProvider, len(r.Cookies))
	for i := range r.Cookies {
		providers[i] = &r.Cookies[i]
	}
	return providers
}

// GetBodySize implements the RequestProvider interface.
func (r *Request) GetBodySize() int {
	if r == nil {
		return 0
	}
	return r.BodySize
}

// GetHeadersSize implements the RequestProvider interface.
func (r *Request) GetHeadersSize() int {
	if r == nil {
		return 0
	}
	return r.HeadersSize
}

// GetQueryString implements the RequestProvider interface.
func (r *Request) GetQueryString() []QueryString {
	if r == nil {
		return nil
	}
	return r.QueryString
}

// GetPostData implements the RequestProvider interface.
func (r *Request) GetPostData() *PostData {
	if r == nil {
		return nil
	}
	return r.PostData
}

// ToStandard implements the RequestProvider interface.
func (r *Request) ToStandard() Request {
	if r == nil {
		return Request{}
	}
	return *r
}

// Response interface implementation.

// GetStatus implements the ResponseProvider interface.
func (r *Response) GetStatus() int {
	if r == nil {
		return 0
	}
	return r.Status
}

// GetStatusText implements the ResponseProvider interface.
func (r *Response) GetStatusText() string {
	if r == nil {
		return ""
	}
	return r.StatusText
}

// GetHTTPVersion implements the ResponseProvider interface.
func (r *Response) GetHTTPVersion() string {
	if r == nil {
		return ""
	}
	return r.HTTPVersion
}

// GetHeaders implements the ResponseProvider interface.
func (r *Response) GetHeaders() []HeaderProvider {
	if r == nil {
		return nil
	}
	providers := make([]HeaderProvider, len(r.Headers))
	for i := range r.Headers {
		providers[i] = &r.Headers[i]
	}
	return providers
}

// GetCookies implements the ResponseProvider interface.
func (r *Response) GetCookies() []CookieProvider {
	if r == nil {
		return nil
	}
	providers := make([]CookieProvider, len(r.Cookies))
	for i := range r.Cookies {
		providers[i] = &r.Cookies[i]
	}
	return providers
}

// GetContent implements the ResponseProvider interface.
func (r *Response) GetContent() ContentProvider {
	if r == nil {
		return nil
	}
	return &r.Content
}

// GetBodySize implements the ResponseProvider interface.
func (r *Response) GetBodySize() int {
	if r == nil {
		return 0
	}
	return r.BodySize
}

// GetHeadersSize implements the ResponseProvider interface.
func (r *Response) GetHeadersSize() int {
	if r == nil {
		return 0
	}
	return r.HeadersSize
}

// ToStandard implements the ResponseProvider interface.
func (r *Response) ToStandard() Response {
	if r == nil {
		return Response{}
	}
	return *r
}

// Headers interface implementation.

// GetName implements the HeaderProvider interface.
func (h *Headers) GetName() string {
	if h == nil {
		return ""
	}
	return h.Name
}

// GetValue implements the HeaderProvider interface.
func (h *Headers) GetValue() string {
	if h == nil {
		return ""
	}
	return h.Value
}

// ToStandard implements the HeaderProvider interface.
func (h *Headers) ToStandard() Headers {
	if h == nil {
		return Headers{}
	}
	return *h
}

// Cookie interface implementation.

// GetName implements the CookieProvider interface.
func (c *Cookie) GetName() string {
	if c == nil {
		return ""
	}
	return c.Name
}

// GetValue implements the CookieProvider interface.
func (c *Cookie) GetValue() string {
	if c == nil {
		return ""
	}
	return c.Value
}

// GetDomain implements the CookieProvider interface.
func (c *Cookie) GetDomain() string {
	if c == nil {
		return ""
	}
	return c.Domain
}

// GetPath implements the CookieProvider interface.
func (c *Cookie) GetPath() string {
	if c == nil {
		return ""
	}
	return c.Path
}

// GetExpires implements the CookieProvider interface.
func (c *Cookie) GetExpires() time.Time {
	if c == nil {
		return time.Time{}
	}
	return c.Expires
}

// IsHTTPOnly implements the CookieProvider interface.
func (c *Cookie) IsHTTPOnly() bool {
	if c == nil {
		return false
	}
	return c.HTTPOnly
}

// IsSecure implements the CookieProvider interface.
func (c *Cookie) IsSecure() bool {
	if c == nil {
		return false
	}
	return c.Secure
}

// GetSameSite implements the CookieProvider interface.
func (c *Cookie) GetSameSite() string {
	if c == nil {
		return ""
	}
	return c.SameSite
}

// ToStandard implements the CookieProvider interface.
func (c *Cookie) ToStandard() Cookie {
	if c == nil {
		return Cookie{}
	}
	return *c
}

// Content interface implementation.

// GetSize implements the ContentProvider interface.
func (c *Content) GetSize() int {
	if c == nil {
		return 0
	}
	return c.Size
}

// GetMimeType implements the ContentProvider interface.
func (c *Content) GetMimeType() string {
	if c == nil {
		return ""
	}
	return c.MimeType
}

// GetText implements the ContentProvider interface.
func (c *Content) GetText() string {
	if c == nil {
		return ""
	}
	return c.Text
}

// GetEncoding implements the ContentProvider interface.
func (c *Content) GetEncoding() string {
	if c == nil {
		return ""
	}
	return c.Encoding
}

// GetCompression implements the ContentProvider interface.
func (c *Content) GetCompression() int {
	if c == nil {
		return 0
	}
	return c.Compression
}

// ToStandard implements the ContentProvider interface.
func (c *Content) ToStandard() Content {
	if c == nil {
		return Content{}
	}
	return *c
}

// Timings interface implementation.

// GetBlocked implements the TimingsProvider interface.
func (t *Timings) GetBlocked() float64 {
	if t == nil {
		return 0
	}
	return t.Blocked
}

// GetDNS implements the TimingsProvider interface.
func (t *Timings) GetDNS() float64 {
	if t == nil {
		return 0
	}
	return t.DNS
}

// GetConnect implements the TimingsProvider interface.
func (t *Timings) GetConnect() float64 {
	if t == nil {
		return 0
	}
	return t.Connect
}

// GetSend implements the TimingsProvider interface.
func (t *Timings) GetSend() float64 {
	if t == nil {
		return 0
	}
	return t.Send
}

// GetWait implements the TimingsProvider interface.
func (t *Timings) GetWait() float64 {
	if t == nil {
		return 0
	}
	return t.Wait
}

// GetReceive implements the TimingsProvider interface.
func (t *Timings) GetReceive() float64 {
	if t == nil {
		return 0
	}
	return t.Receive
}

// GetSSL implements the TimingsProvider interface.
func (t *Timings) GetSSL() float64 {
	if t == nil {
		return 0
	}
	return t.Ssl
}

// ToStandard implements the TimingsProvider interface.
func (t *Timings) ToStandard() Timings {
	if t == nil {
		return Timings{}
	}
	return *t
}

// Pages interface implementation.

// GetID implements the PageProvider interface.
func (p *Pages) GetID() string {
	if p == nil {
		return ""
	}
	return p.ID
}

// GetTitle implements the PageProvider interface.
func (p *Pages) GetTitle() string {
	if p == nil {
		return ""
	}
	return p.Title
}

// GetStartedDateTime implements the PageProvider interface.
func (p *Pages) GetStartedDateTime() time.Time {
	if p == nil {
		return time.Time{}
	}
	return p.StartedDateTime
}

// GetPageTimings implements the PageProvider interface.
func (p *Pages) GetPageTimings() PageTimingsProvider {
	if p == nil {
		return nil
	}
	return &p.PageTimings
}

// ToStandard implements the PageProvider interface.
func (p *Pages) ToStandard() Pages {
	if p == nil {
		return Pages{}
	}
	return *p
}

// PageTimings interface implementation.

// GetOnContentLoad implements the PageTimingsProvider interface.
func (pt *PageTimings) GetOnContentLoad() float64 {
	if pt == nil {
		return 0
	}
	return pt.OnContentLoad
}

// GetOnLoad implements the PageTimingsProvider interface.
func (pt *PageTimings) GetOnLoad() float64 {
	if pt == nil {
		return 0
	}
	return pt.OnLoad
}

// ToStandard implements the PageTimingsProvider interface.
func (pt *PageTimings) ToStandard() PageTimings {
	if pt == nil {
		return PageTimings{}
	}
	return *pt
}
