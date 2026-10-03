package har

import "time"

// LazyHar interface implementation.

// GetVersion implements the HARProvider interface.
func (h *LazyHar) GetVersion() string {
	if h == nil {
		return ""
	}
	return h.Log.Version
}

// GetCreator implements the HARProvider interface.
func (h *LazyHar) GetCreator() Creator {
	if h == nil {
		return Creator{}
	}
	return h.Log.Creator
}

// GetBrowser implements the HARProvider interface.
func (h *LazyHar) GetBrowser() Browser {
	if h == nil {
		return Browser{}
	}
	return h.Log.Browser
}

// GetEntries implements the HARProvider interface.
func (h *LazyHar) GetEntries() []EntryProvider {
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
func (h *LazyHar) GetPages() []PageProvider {
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
func (h *LazyHar) ToStandard() *Har {
	if h == nil {
		return nil
	}
	result := &Har{
		Log: Log{
			Version: h.Log.Version,
			Creator: h.Log.Creator,
			Browser: h.Log.Browser,
			Pages:   h.Log.Pages,
			Entries: make([]Entries, len(h.Log.Entries)),
		},
	}
	for i := range h.Log.Entries {
		result.Log.Entries[i] = h.Log.Entries[i].ToStandard()
	}
	return result
}

// LazyEntries interface implementation.

// GetStartedDateTime implements the EntryProvider interface.
func (e *LazyEntries) GetStartedDateTime() time.Time {
	if e == nil {
		return time.Time{}
	}
	return e.StartedDateTime
}

// GetTime implements the EntryProvider interface.
func (e *LazyEntries) GetTime() float64 {
	if e == nil {
		return 0
	}
	return e.Time
}

// GetRequest implements the EntryProvider interface.
func (e *LazyEntries) GetRequest() RequestProvider {
	if e == nil {
		return nil
	}
	return &e.Request
}

// GetResponse implements the EntryProvider interface.
func (e *LazyEntries) GetResponse() ResponseProvider {
	if e == nil {
		return nil
	}
	return &e.Response
}

// GetTimings implements the EntryProvider interface.
func (e *LazyEntries) GetTimings() TimingsProvider {
	if e == nil {
		return nil
	}
	return &e.Timings
}

// GetPageref implements the EntryProvider interface.
func (e *LazyEntries) GetPageref() string {
	if e == nil {
		return ""
	}
	return e.Pageref
}

// ToStandard implements the EntryProvider interface.
func (e *LazyEntries) ToStandard() Entries {
	if e == nil {
		return Entries{}
	}
	return Entries{
		StartedDateTime: e.StartedDateTime,
		Time:            e.Time,
		Request:         e.Request,
		Response:        e.Response.ToStandard(),
		Cache:           e.Cache,
		Timings:         e.Timings,
		Pageref:         e.Pageref,
		ServerIPAddress: e.ServerIPAddress,
		Connection:      e.Connection,
		Comment:         e.Comment,
	}
}

// LazyResponse interface implementation.

// GetStatus implements the ResponseProvider interface.
func (r *LazyResponse) GetStatus() int {
	if r == nil {
		return 0
	}
	return r.Status
}

// GetStatusText implements the ResponseProvider interface.
func (r *LazyResponse) GetStatusText() string {
	if r == nil {
		return ""
	}
	return r.StatusText
}

// GetHTTPVersion implements the ResponseProvider interface.
func (r *LazyResponse) GetHTTPVersion() string {
	if r == nil {
		return ""
	}
	return r.HTTPVersion
}

// GetHeaders implements the ResponseProvider interface.
func (r *LazyResponse) GetHeaders() []HeaderProvider {
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
func (r *LazyResponse) GetCookies() []CookieProvider {
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
func (r *LazyResponse) GetContent() ContentProvider {
	if r == nil {
		return nil
	}
	// A LazyContent pointer does not directly implement the ContentProvider interface.
	// Therefore, a wrapper must be created.
	return &lazyContentWrapper{content: r.Content}
}

// GetBodySize implements the ResponseProvider interface.
func (r *LazyResponse) GetBodySize() int {
	if r == nil {
		return 0
	}
	return r.BodySize
}

// GetHeadersSize implements the ResponseProvider interface.
func (r *LazyResponse) GetHeadersSize() int {
	if r == nil {
		return 0
	}
	return r.HeadersSize
}

// ToStandard implements the ResponseProvider interface.
func (r *LazyResponse) ToStandard() Response {
	if r == nil {
		return Response{}
	}
	var content Content

	// Create a standard Content object, preserving all fields.
	if r.Content != nil {
		content = r.Content.ToStandard()
	}

	return Response{
		Status:       r.Status,
		StatusText:   r.StatusText,
		HTTPVersion:  r.HTTPVersion,
		Cookies:      r.Cookies,
		Headers:      r.Headers,
		RedirectURL:  r.RedirectURL,
		HeadersSize:  r.HeadersSize,
		BodySize:     r.BodySize,
		Content:      content,
		TransferSize: r.TransferSize,
		Error:        r.Error,
	}
}

// lazyContentWrapper wraps LazyContent.
// Implements the ContentProvider interface.
type lazyContentWrapper struct {
	content *LazyContent
}

// GetSize implements the ContentProvider  interface.
func (w *lazyContentWrapper) GetSize() int {
	if w == nil || w.content == nil {
		return 0
	}
	return w.content.Size
}

// GetMimeType implements the ContentProvider interface.
func (w *lazyContentWrapper) GetMimeType() string {
	if w == nil || w.content == nil {
		return ""
	}
	return w.content.MimeType
}

// GetText implements the ContentProvider  interface.
func (w *lazyContentWrapper) GetText() string {
	if w == nil || w.content == nil {
		return ""
	}
	// LazyContent.GetText() returns (*string, error) - defined in lazy.go
	text, err := w.content.GetText()
	if err != nil || text == nil {
		return ""
	}
	return *text
}

// GetEncoding implements the ContentProvider  interface.
func (w *lazyContentWrapper) GetEncoding() string {
	if w == nil || w.content == nil {
		return ""
	}

	// Ensure the content is loaded.
	_ = w.content.Load()
	if w.content.Encoding == nil {
		return ""
	}
	return *w.content.Encoding
}

// GetCompression implements the ContentProvider  interface.
func (w *lazyContentWrapper) GetCompression() int {
	if w == nil || w.content == nil {
		return 0
	}
	return w.content.Compression
}

// ToStandard implements the ContentProvider  interface.
func (w *lazyContentWrapper) ToStandard() Content {
	if w == nil || w.content == nil {
		return Content{}
	}

	return w.content.ToStandard()
}

// LazyContent interface implementation.

// GetSize implements the ContentProvider interface.
func (c *LazyContent) GetSize() int {
	if c == nil {
		return 0
	}
	return c.Size
}

// GetMimeType implements the ContentProvider interface.
func (c *LazyContent) GetMimeType() string {
	if c == nil {
		return ""
	}
	return c.MimeType
}

// GetEncoding implements the ContentProvider interface.
func (c *LazyContent) GetEncoding() string {
	if c == nil {
		return ""
	}
	_ = c.Load()
	if c.Encoding == nil {
		return ""
	}
	return *c.Encoding
}

// GetCompression implements the ContentProvider interface.
func (c *LazyContent) GetCompression() int {
	if c == nil {
		return 0
	}
	return c.Compression
}

// ToStandard implements the ContentProvider interface.
func (c *LazyContent) ToStandard() Content {
	if c == nil {
		return Content{}
	}
	_ = c.Load()
	content := Content{
		Size:        c.Size,
		MimeType:    c.MimeType,
		Compression: c.Compression,
		Comment:     c.Comment,
	}
	if c.Text != nil {
		content.Text = *c.Text
	}
	if c.Encoding != nil {
		content.Encoding = *c.Encoding
	}
	return content
}
