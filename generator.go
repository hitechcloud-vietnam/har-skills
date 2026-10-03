package har

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// NewHar creates a new Har object.
func NewHar() *Har {
	return &Har{
		Log: Log{
			Version: "1.2",
			Creator: Creator{
				Name:    "go-har",
				Version: "1.0",
			},
			Pages:   []Pages{},
			Entries: []Entries{},
		},
	}
}

// SetBrowser sets the browser information for a HAR file.
func (h *Har) SetBrowser(name, version string) *Har {
	if h == nil {
		return nil
	}
	h.Log.Browser = Browser{
		Name:    name,
		Version: version,
	}
	return h
}

// SetVersion sets the HAR specification version.
func (h *Har) SetVersion(version string) *Har {
	if h == nil {
		return nil
	}
	h.Log.Version = version
	return h
}

// SetCreator sets the creator information for a HAR file.
func (h *Har) SetCreator(name, version string) *Har {
	if h == nil {
		return nil
	}
	h.Log.Creator.Name = name
	h.Log.Creator.Version = version
	return h
}

// AddPage adds page information.
func (h *Har) AddPage(id, title string) *Pages {
	if h == nil {
		return nil
	}

	page := Pages{
		StartedDateTime: time.Now(),
		ID:              id,
		Title:           title,
		PageTimings: PageTimings{
			OnContentLoad: -1,
			OnLoad:        -1,
		},
	}
	h.Log.Pages = append(h.Log.Pages, page)
	return &h.Log.Pages[len(h.Log.Pages)-1]
}

// SetPageTimings sets page load timings.
func (p *Pages) SetPageTimings(onContentLoad, onLoad float64) *Pages {
	if p == nil {
		return nil
	}
	p.PageTimings.OnContentLoad = onContentLoad
	p.PageTimings.OnLoad = onLoad
	return p
}

// AddEntry adds a request/response entry.
func (h *Har) AddEntry(method, url, httpVersion string, pageref string) *Entries {
	if h == nil {
		return nil
	}

	entry := Entries{
		StartedDateTime: time.Now(),
		Time:            0,
		Request: Request{
			Method:      method,
			URL:         url,
			HTTPVersion: httpVersion,
			Headers:     []Headers{},
			Cookies:     []Cookie{},
			QueryString: []QueryString{},
			HeadersSize: -1,
			BodySize:    -1,
		},
		Response: Response{
			Status:      0,
			StatusText:  "",
			HTTPVersion: httpVersion,
			Headers:     []Headers{},
			Cookies:     []Cookie{},
			Content: Content{
				Size:     0,
				MimeType: "",
			},
			RedirectURL:  "",
			HeadersSize:  -1,
			BodySize:     -1,
			TransferSize: -1,
		},
		Cache: Cache{},
		Timings: Timings{
			Blocked: -1,
			DNS:     -1,
			Connect: -1,
			Send:    -1,
			Wait:    -1,
			Receive: -1,
			Ssl:     -1,
		},
		Pageref: pageref,
	}
	h.Log.Entries = append(h.Log.Entries, entry)
	return &h.Log.Entries[len(h.Log.Entries)-1]
}

// AddRequestHeader adds a request header.
func (e *Entries) AddRequestHeader(name, value string) *Entries {
	if e == nil {
		return nil
	}
	e.Request.Headers = append(e.Request.Headers, Headers{
		Name:  name,
		Value: value,
	})
	return e
}

// AddResponseHeader adds a response header.
func (e *Entries) AddResponseHeader(name, value string) *Entries {
	if e == nil {
		return nil
	}
	e.Response.Headers = append(e.Response.Headers, Headers{
		Name:  name,
		Value: value,
	})
	return e
}

// SetResponseStatus sets the response status.
func (e *Entries) SetResponseStatus(status int, statusText string) *Entries {
	if e == nil {
		return nil
	}
	e.Response.Status = status
	e.Response.StatusText = statusText
	return e
}

// SetResponseContent sets the response content.
func (e *Entries) SetResponseContent(size int, mimeType string) *Entries {
	if e == nil {
		return nil
	}
	e.Response.Content.Size = size
	e.Response.Content.MimeType = mimeType
	return e
}

// SetTimings sets timing data.
func (e *Entries) SetTimings(blocked, dns, connect, send, wait, receive, ssl float64) *Entries {
	if e == nil {
		return nil
	}
	e.Timings.Blocked = blocked
	e.Timings.DNS = dns
	e.Timings.Connect = connect
	e.Timings.Send = send
	e.Timings.Wait = wait
	e.Timings.Receive = receive
	e.Timings.Ssl = ssl

	// Calculate the total time.
	// Note: according to the HAR specification, SSL time is included in connect time and must not be counted twice.
	e.Time = blocked + dns + connect + send + wait + receive
	return e
}

// AddCookie adds a request cookie.
func (e *Entries) AddCookie(name, value string) *Entries {
	if e == nil {
		return nil
	}
	e.Request.Cookies = append(e.Request.Cookies, Cookie{
		Name:  name,
		Value: value,
	})
	return e
}

// AddResponseCookie adds a response cookie.
func (e *Entries) AddResponseCookie(name, value string) *Entries {
	if e == nil {
		return nil
	}
	e.Response.Cookies = append(e.Response.Cookies, Cookie{
		Name:  name,
		Value: value,
	})
	return e
}

// AddQueryParameter adds a query parameter.
func (e *Entries) AddQueryParameter(name, value string) *Entries {
	if e == nil {
		return nil
	}
	e.Request.QueryString = append(e.Request.QueryString, QueryString{
		Name:  name,
		Value: value,
	})
	return e
}

// SetPostData sets the POST request body.
func (e *Entries) SetPostData(mimeType, text string) *Entries {
	if e == nil {
		return nil
	}
	e.Request.PostData = &PostData{
		MimeType: mimeType,
		Text:     text,
	}
	return e
}

// SetPostDataParams sets POST form parameters.
func (e *Entries) SetPostDataParams(mimeType string, params []Param) *Entries {
	if e == nil {
		return nil
	}
	e.Request.PostData = &PostData{
		MimeType: mimeType,
		Params:   params,
	}
	return e
}

// SetResponseContentText sets the response content text.
func (e *Entries) SetResponseContentText(text string) *Entries {
	if e == nil {
		return nil
	}
	e.Response.Content.Text = text
	e.Response.Content.Size = len(text)
	return e
}

// SetServerIP sets the server IP address.
func (e *Entries) SetServerIP(ip string) *Entries {
	if e == nil {
		return nil
	}
	e.ServerIPAddress = ip
	return e
}

// SetConnection sets the connection ID.
func (e *Entries) SetConnection(id string) *Entries {
	if e == nil {
		return nil
	}
	e.Connection = id
	return e
}

// SetPageref sets the page reference.
func (e *Entries) SetPageref(ref string) *Entries {
	if e == nil {
		return nil
	}
	e.Pageref = ref
	return e
}

// ToJSON converts a Har object to JSON bytes.
func (h *Har) ToJSON(indent bool) ([]byte, error) {
	if h == nil {
		return nil, NewInvalidFormatError("HAR object is nil")
	}

	var (
		data []byte
		err  error
	)
	if indent {
		data, err = json.MarshalIndent(h, "", "  ")
	} else {
		data, err = json.Marshal(h)
	}
	if err != nil {
		return nil, NewJSONParseError("JSON serialization failed", err)
	}
	return data, nil
}

// SaveToFile saves a Har object to a file.
func (h *Har) SaveToFile(filePath string, indent bool) error {
	if h == nil {
		return NewInvalidFormatError("HAR object is nil")
	}

	data, err := h.ToJSON(indent)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return NewFileSystemError(fmt.Sprintf("Unable to write file '%s'", filePath), err)
	}

	return nil
}
