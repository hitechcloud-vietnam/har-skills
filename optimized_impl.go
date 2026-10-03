package har

import "time"

// OptimizedHar interface implementation.

// GetVersion implements the HARProvider interface.
func (h *OptimizedHar) GetVersion() string {
	if h == nil {
		return ""
	}
	return h.Log.Version
}

// GetCreator implements the HARProvider interface.
func (h *OptimizedHar) GetCreator() Creator {
	if h == nil {
		return Creator{}
	}
	return h.Log.Creator
}

// GetBrowser implements the HARProvider interface.
func (h *OptimizedHar) GetBrowser() Browser {
	if h == nil {
		return Browser{}
	}
	return h.Log.Browser
}

// GetEntries implements the HARProvider interface.
func (h *OptimizedHar) GetEntries() []EntryProvider {
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
func (h *OptimizedHar) GetPages() []PageProvider {
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
func (h *OptimizedHar) ToStandard() *Har {
	if h == nil {
		return nil
	}

	// Convert from the optimized format to the standard format.
	standard := &Har{
		Log: Log{
			Version: h.Log.Version,
			Creator: h.Log.Creator,
			Browser: h.Log.Browser,
			Pages:   h.Log.Pages,
			Entries: make([]Entries, len(h.Log.Entries)),
		},
	}

	// Convert entries.
	for i, entry := range h.Log.Entries {
		standard.Log.Entries[i] = entry.ToStandard()
	}

	return standard
}

// OptimizedEntries interface implementation.

// GetStartedDateTime implements the EntryProvider interface.
func (e *OptimizedEntries) GetStartedDateTime() time.Time {
	if e == nil {
		return time.Time{}
	}
	return e.StartedDateTime
}

// GetTime implements the EntryProvider interface.
func (e *OptimizedEntries) GetTime() float64 {
	if e == nil {
		return 0
	}
	return e.Time
}

// GetRequest implements the EntryProvider interface.
func (e *OptimizedEntries) GetRequest() RequestProvider {
	if e == nil {
		return nil
	}
	return &e.Request
}

// GetResponse implements the EntryProvider interface.
func (e *OptimizedEntries) GetResponse() ResponseProvider {
	if e == nil {
		return nil
	}
	return &e.Response
}

// GetTimings implements the EntryProvider interface.
func (e *OptimizedEntries) GetTimings() TimingsProvider {
	if e == nil {
		return nil
	}
	return &e.Timings
}

// GetPageref implements the EntryProvider interface.
func (e *OptimizedEntries) GetPageref() string {
	if e == nil {
		return ""
	}
	if e.PageRef != nil {
		return *e.PageRef
	}
	return ""
}

// ToStandard implements the EntryProvider interface.
func (e *OptimizedEntries) ToStandard() Entries {
	if e == nil {
		return Entries{}
	}

	// Convert to the standard format.
	entry := Entries{
		StartedDateTime: e.StartedDateTime,
		Time:            e.Time,
		Request:         e.Request.ToStandard(),
		Response:        e.Response.ToStandard(),
		Timings:         e.Timings.ToStandard(),
	}

	// Optionally add Pageref if it is not empty.
	if e.PageRef != nil {
		entry.Pageref = *e.PageRef
	}

	// Optionally add ServerIP.
	if e.ServerIP != nil {
		entry.ServerIPAddress = *e.ServerIP
	}

	// Optionally add Connection.
	if e.Connection != nil {
		entry.Connection = *e.Connection
	}

	// Convert cache data.
	if e.Cache != nil {
		entry.Cache = *e.Cache
	}

	return entry
}

// OptimizedRequest interface implementation.

// GetMethod implements the RequestProvider interface.
func (r *OptimizedRequest) GetMethod() string {
	if r == nil {
		return MethodUnknown.String()
	}

	// Convert the HTTPMethod enum to a string.
	switch r.Method {
	case MethodGET:
		return "GET"
	case MethodPOST:
		return "POST"
	case MethodPUT:
		return "PUT"
	case MethodDELETE:
		return "DELETE"
	case MethodHEAD:
		return "HEAD"
	case MethodOPTIONS:
		return "OPTIONS"
	case MethodPATCH:
		return "PATCH"
	case MethodCONNECT:
		return "CONNECT"
	case MethodTRACE:
		return "TRACE"
	default:
		return "UNKNOWN"
	}
}

// GetURL implements the RequestProvider interface.
func (r *OptimizedRequest) GetURL() string {
	if r == nil {
		return ""
	}
	return r.URL
}

// GetHTTPVersion implements the RequestProvider interface.
func (r *OptimizedRequest) GetHTTPVersion() string {
	if r == nil {
		return ""
	}
	return r.HTTPVersion
}

// GetHeaders implements the RequestProvider interface.
func (r *OptimizedRequest) GetHeaders() []HeaderProvider {
	if r == nil {
		return nil
	}

	// Convert the map to a slice.
	headers := make([]HeaderProvider, 0, len(r.Headers))
	for name, value := range r.Headers {
		header := &Headers{
			Name:  name,
			Value: value,
		}
		headers = append(headers, header)
	}
	return headers
}

// GetCookies implements the RequestProvider interface.
func (r *OptimizedRequest) GetCookies() []CookieProvider {
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
func (r *OptimizedRequest) GetBodySize() int {
	if r == nil {
		return 0
	}
	if r.BodySize != nil {
		return *r.BodySize
	}
	return 0
}

// GetHeadersSize implements the RequestProvider interface.
func (r *OptimizedRequest) GetHeadersSize() int {
	if r == nil {
		return 0
	}
	if r.HeadersSize != nil {
		return *r.HeadersSize
	}
	return 0
}

// GetQueryString implements the RequestProvider interface.
func (r *OptimizedRequest) GetQueryString() []QueryString {
	if r == nil {
		return nil
	}

	params := make([]QueryString, 0, len(r.QueryString))
	for name, value := range r.QueryString {
		params = append(params, QueryString{
			Name:  name,
			Value: value,
		})
	}
	return params
}

// GetPostData implements the RequestProvider interface.
func (r *OptimizedRequest) GetPostData() *PostData {
	if r == nil {
		return nil
	}
	return r.PostData
}

// ToStandard implements the RequestProvider interface.
func (r *OptimizedRequest) ToStandard() Request {
	if r == nil {
		return Request{}
	}

	// Convert from the optimized format to the standard format.
	request := Request{
		Method:      r.GetMethod(),
		URL:         r.URL,
		HTTPVersion: r.HTTPVersion,
		PostData:    r.PostData,
	}

	// Convert headers.
	for name, value := range r.Headers {
		request.Headers = append(request.Headers, Headers{
			Name:  name,
			Value: value,
		})
	}

	// Convert query parameters.
	for name, value := range r.QueryString {
		request.QueryString = append(request.QueryString, QueryString{
			Name:  name,
			Value: value,
		})
	}

	// Convert cookies.
	request.Cookies = make([]Cookie, len(r.Cookies))
	copy(request.Cookies, r.Cookies)

	// Handle optional fields.
	if r.HeadersSize != nil {
		request.HeadersSize = *r.HeadersSize
	}

	if r.BodySize != nil {
		request.BodySize = *r.BodySize
	}

	return request
}

// OptimizedResponse interface implementation.

// GetStatus implements the ResponseProvider interface.
func (r *OptimizedResponse) GetStatus() int {
	if r == nil {
		return 0
	}
	return r.Status
}

// GetStatusText implements the ResponseProvider interface.
func (r *OptimizedResponse) GetStatusText() string {
	if r == nil {
		return ""
	}
	return r.StatusText
}

// GetHTTPVersion implements the ResponseProvider interface.
func (r *OptimizedResponse) GetHTTPVersion() string {
	if r == nil {
		return ""
	}
	return r.HTTPVersion
}

// GetHeaders implements the ResponseProvider interface.
func (r *OptimizedResponse) GetHeaders() []HeaderProvider {
	if r == nil {
		return nil
	}

	// Convert the map to a slice.
	headers := make([]HeaderProvider, 0, len(r.Headers))
	for name, value := range r.Headers {
		header := &Headers{
			Name:  name,
			Value: value,
		}
		headers = append(headers, header)
	}
	return headers
}

// GetCookies implements the ResponseProvider interface.
func (r *OptimizedResponse) GetCookies() []CookieProvider {
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
func (r *OptimizedResponse) GetContent() ContentProvider {
	if r == nil {
		return nil
	}
	if r.Content == nil {
		return nil
	}
	return r.Content
}

// GetBodySize implements the ResponseProvider interface.
func (r *OptimizedResponse) GetBodySize() int {
	if r == nil {
		return 0
	}
	if r.BodySize != nil {
		return *r.BodySize
	}
	return 0
}

// GetHeadersSize implements the ResponseProvider interface.
func (r *OptimizedResponse) GetHeadersSize() int {
	if r == nil {
		return 0
	}
	if r.HeadersSize != nil {
		return *r.HeadersSize
	}
	return 0
}

// ToStandard implements the ResponseProvider interface.
func (r *OptimizedResponse) ToStandard() Response {
	if r == nil {
		return Response{}
	}

	// Convert from the optimized format to the standard format.
	response := Response{
		Status:      r.Status,
		StatusText:  r.StatusText,
		HTTPVersion: r.HTTPVersion,
		RedirectURL: r.RedirectURL,
	}

	// Handle the Content field.
	if r.Content != nil {
		response.Content = r.Content.ToStandard()
	}

	// Convert headers.
	for name, value := range r.Headers {
		response.Headers = append(response.Headers, Headers{
			Name:  name,
			Value: value,
		})
	}

	// Convert cookies.
	response.Cookies = make([]Cookie, len(r.Cookies))
	copy(response.Cookies, r.Cookies)

	// Handle optional fields.
	if r.HeadersSize != nil {
		response.HeadersSize = *r.HeadersSize
	}

	if r.BodySize != nil {
		response.BodySize = *r.BodySize
	}

	return response
}

// OptimizedContent interface implementation.

// GetSize implements the ContentProvider interface.
func (c *OptimizedContent) GetSize() int {
	if c == nil {
		return 0
	}
	return c.Size
}

// GetMimeType implements the ContentProvider interface.
func (c *OptimizedContent) GetMimeType() string {
	if c == nil {
		return ""
	}
	return c.MimeType
}

// GetText implements the ContentProvider interface.
func (c *OptimizedContent) GetText() string {
	if c == nil {
		return ""
	}
	if c.Text != nil {
		return *c.Text
	}
	return ""
}

// GetEncoding implements the ContentProvider interface.
func (c *OptimizedContent) GetEncoding() string {
	if c == nil {
		return ""
	}
	if c.Encoding != nil {
		return *c.Encoding
	}
	return ""
}

// GetCompression implements the ContentProvider interface.
func (c *OptimizedContent) GetCompression() int {
	return 0 // OptimizedContent doesn't track compression
}

// ToStandard implements the ContentProvider interface.
func (c *OptimizedContent) ToStandard() Content {
	if c == nil {
		return Content{}
	}

	content := Content{
		Size:     c.Size,
		MimeType: c.MimeType,
	}
	if c.Text != nil {
		content.Text = *c.Text
	}
	if c.Encoding != nil {
		content.Encoding = *c.Encoding
	}
	if c.Comment != nil {
		content.Comment = *c.Comment
	}
	return content
}

// OptimizedTimings interface implementation.

// GetBlocked implements the TimingsProvider interface.
func (t *OptimizedTimings) GetBlocked() float64 {
	if t == nil {
		return -1
	}
	if t.Blocked != nil {
		return *t.Blocked
	}
	return -1
}

// GetDNS implements the TimingsProvider interface.
func (t *OptimizedTimings) GetDNS() float64 {
	if t == nil {
		return -1
	}
	if t.DNS != nil {
		return *t.DNS
	}
	return -1
}

// GetConnect implements the TimingsProvider interface.
func (t *OptimizedTimings) GetConnect() float64 {
	if t == nil {
		return -1
	}
	if t.Connect != nil {
		return *t.Connect
	}
	return -1
}

// GetSend implements the TimingsProvider interface.
func (t *OptimizedTimings) GetSend() float64 {
	if t == nil {
		return -1
	}
	if t.Send != nil {
		return *t.Send
	}
	return -1
}

// GetWait implements the TimingsProvider interface.
func (t *OptimizedTimings) GetWait() float64 {
	if t == nil {
		return -1
	}
	if t.Wait != nil {
		return *t.Wait
	}
	return -1
}

// GetReceive implements the TimingsProvider interface.
func (t *OptimizedTimings) GetReceive() float64 {
	if t == nil {
		return -1
	}
	if t.Receive != nil {
		return *t.Receive
	}
	return -1
}

// GetSSL implements the TimingsProvider interface.
func (t *OptimizedTimings) GetSSL() float64 {
	if t == nil {
		return -1
	}
	if t.Ssl != nil {
		return *t.Ssl
	}
	return -1
}

// ToStandard implements the TimingsProvider interface.
func (t *OptimizedTimings) ToStandard() Timings {
	if t == nil {
		return Timings{
			Blocked: -1,
			DNS:     -1,
			Connect: -1,
			Send:    -1,
			Wait:    -1,
			Receive: -1,
			Ssl:     -1,
		}
	}

	timings := Timings{}

	// Handle required fields to avoid nil pointers.
	if t.Blocked != nil {
		timings.Blocked = *t.Blocked
	} else {
		timings.Blocked = -1
	}

	if t.Send != nil {
		timings.Send = *t.Send
	} else {
		timings.Send = -1
	}

	if t.Wait != nil {
		timings.Wait = *t.Wait
	} else {
		timings.Wait = -1
	}

	if t.Receive != nil {
		timings.Receive = *t.Receive
	} else {
		timings.Receive = -1
	}

	// Handle optional fields.
	if t.DNS != nil {
		timings.DNS = *t.DNS
	} else {
		timings.DNS = -1
	}

	if t.Connect != nil {
		timings.Connect = *t.Connect
	} else {
		timings.Connect = -1
	}

	if t.Ssl != nil {
		timings.Ssl = *t.Ssl
	} else {
		timings.Ssl = -1
	}

	return timings
}
