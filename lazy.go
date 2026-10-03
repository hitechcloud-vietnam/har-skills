package har

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// LazyContent represents lazily loaded content.
type LazyContent struct {
	// Basic information is always loaded.
	Size        int    `json:"size"`
	MimeType    string `json:"mimeType"`
	Compression int    `json:"compression,omitempty"`

	// Actual content is loaded lazily.
	Text     *string `json:"text,omitempty"`
	Encoding *string `json:"encoding,omitempty"`
	Comment  string  `json:"comment,omitempty"`

	// Raw data used for lazy loading.
	rawData   json.RawMessage `json:"-"`
	loaded    bool            `json:"-"`
	loadMutex sync.RWMutex    `json:"-"`
}

// LazyResponse represents a response with lazily loaded content.
type LazyResponse struct {
	Status       int          `json:"status"`
	StatusText   string       `json:"statusText"`
	HTTPVersion  string       `json:"httpVersion"`
	Cookies      []Cookie     `json:"cookies"`
	Headers      []Headers    `json:"headers"`
	RedirectURL  string       `json:"redirectURL"`
	HeadersSize  int          `json:"headersSize"`
	BodySize     int          `json:"bodySize"`
	Content      *LazyContent `json:"content"`
	TransferSize int          `json:"_transferSize"`
	Error        any          `json:"_error"`
	Comment      string       `json:"comment,omitempty"`
}

// LazyEntries represents an entry with lazily loaded content.
type LazyEntries struct {
	StartedDateTime time.Time    `json:"startedDateTime"`
	Time            float64      `json:"time"`
	Request         Request      `json:"request"`
	Response        LazyResponse `json:"response"`
	Cache           Cache        `json:"cache"`
	Timings         Timings      `json:"timings"`
	Pageref         string       `json:"pageref"`
	Initiator       Initiator    `json:"_initiator"`
	Priority        string       `json:"_priority"`
	ResourceType    string       `json:"_resourceType"`
	Connection      string       `json:"connection"`
	ServerIPAddress string       `json:"serverIPAddress"`
	Comment         string       `json:"comment,omitempty"`
}

// LazyHar is a HAR object with lazy-loading support.
type LazyHar struct {
	Log struct {
		Version string        `json:"version"`
		Creator Creator       `json:"creator"`
		Browser Browser       `json:"browser,omitempty"`
		Pages   []Pages       `json:"pages"`
		Entries []LazyEntries `json:"entries"`
	} `json:"log"`
}

// UnmarshalJSON customizes JSON parsing to load only basic information initially.
func (lc *LazyContent) UnmarshalJSON(data []byte) error {
	if lc == nil {
		return NewInvalidFormatError("Content is empty")
	}

	// Stores raw data for lazy loading.
	lc.rawData = make(json.RawMessage, len(data))
	copy(lc.rawData, data)

	// Parse basic information.
	type BasicContent struct {
		Size        int    `json:"size"`
		MimeType    string `json:"mimeType"`
		Compression int    `json:"compression"`
		Comment     string `json:"comment"`
	}

	var basic BasicContent
	if err := json.Unmarshal(data, &basic); err != nil {
		return WrapJSONUnmarshalError(err)
	}

	lc.Size = basic.Size
	lc.MimeType = basic.MimeType
	lc.Compression = basic.Compression
	lc.Comment = basic.Comment
	lc.loaded = false

	return nil
}

// Load loads the complete content data.
func (lc *LazyContent) Load() error {
	if lc == nil {
		return NewInvalidFormatError("Content is empty")
	}

	lc.loadMutex.Lock()
	defer lc.loadMutex.Unlock()

	if lc.loaded {
		return nil
	}

	// Temporary structure used to parse the complete content.
	type FullContent struct {
		Text     *string `json:"text,omitempty"`
		Encoding *string `json:"encoding,omitempty"`
	}

	var full FullContent
	if err := json.Unmarshal(lc.rawData, &full); err != nil {
		return NewJSONParseError("Unable to load lazily loaded content", err)
	}

	lc.Text = full.Text
	lc.Encoding = full.Encoding
	lc.loaded = true

	return nil
}

// GetText returns the content text, loading it first if necessary.
func (lc *LazyContent) GetText() (*string, error) {
	if lc == nil {
		return nil, NewInvalidFormatError("Content is empty")
	}

	lc.loadMutex.RLock()
	if lc.loaded {
		text := lc.Text
		lc.loadMutex.RUnlock()
		return text, nil
	}
	lc.loadMutex.RUnlock()

	if err := lc.Load(); err != nil {
		return nil, err
	}

	return lc.Text, nil
}

// ParseHarWithLazyLoading parses HAR data, lazily loading large fields.
func ParseHarWithLazyLoading(harFileBytes []byte) (*LazyHar, error) {
	if err := validateInput(harFileBytes); err != nil {
		return nil, err
	}

	har := new(LazyHar)
	err := json.Unmarshal(harFileBytes, har)
	if err != nil {
		return nil, WrapJSONUnmarshalError(err)
	}
	return har, nil
}

// ParseHarFileWithLazyLoading parses a HAR file, lazily loading large fields.
func ParseHarFileWithLazyLoading(harFilePath string) (*LazyHar, error) {
	harFileBytes, err := os.ReadFile(harFilePath)
	if err != nil {
		return nil, NewFileSystemError("Unable to read HAR file", err)
	}
	return ParseHarWithLazyLoading(harFileBytes)
}

// ToStandardHar converts a LazyHar to a standard Har object.
func (lh *LazyHar) ToStandardHar() (*Har, error) {
	if lh == nil {
		return nil, NewInvalidFormatError("HAR object is nil")
	}

	// Create a standard HAR object.
	result := &Har{
		Log: Log{
			Version: lh.Log.Version,
			Creator: lh.Log.Creator,
			Browser: lh.Log.Browser,
			Pages:   lh.Log.Pages,
			Entries: make([]Entries, len(lh.Log.Entries)),
		},
	}

	// Convert entries.
	for i, lazyEntry := range lh.Log.Entries {
		// Copy basic fields.
		entry := Entries{
			StartedDateTime: lazyEntry.StartedDateTime,
			Time:            lazyEntry.Time,
			Request:         lazyEntry.Request,
			Cache:           lazyEntry.Cache,
			Timings:         lazyEntry.Timings,
			Pageref:         lazyEntry.Pageref,
			Initiator:       lazyEntry.Initiator,
			Priority:        lazyEntry.Priority,
			ResourceType:    lazyEntry.ResourceType,
			Connection:      lazyEntry.Connection,
			ServerIPAddress: lazyEntry.ServerIPAddress,
			Comment:         lazyEntry.Comment,
		}

		// Copy response fields.
		entry.Response = Response{
			Status:       lazyEntry.Response.Status,
			StatusText:   lazyEntry.Response.StatusText,
			HTTPVersion:  lazyEntry.Response.HTTPVersion,
			Cookies:      lazyEntry.Response.Cookies,
			Headers:      lazyEntry.Response.Headers,
			RedirectURL:  lazyEntry.Response.RedirectURL,
			HeadersSize:  lazyEntry.Response.HeadersSize,
			BodySize:     lazyEntry.Response.BodySize,
			TransferSize: lazyEntry.Response.TransferSize,
			Error:        lazyEntry.Response.Error,
			Comment:      lazyEntry.Response.Comment,
		}

		// Copy content.
		if lazyEntry.Response.Content != nil {
			// Ensure the content is loaded.
			if err := lazyEntry.Response.Content.Load(); err != nil {
				return nil, NewJSONParseError("Unable to load lazily loaded content", err)
			}

			entry.Response.Content = lazyEntry.Response.Content.ToStandard()
		}

		result.Log.Entries[i] = entry
	}

	return result, nil
}

// GetEntry returns the entry at the specified index.
func (lh *LazyHar) GetEntry(index int) (*LazyEntries, error) {
	if lh == nil {
		return nil, NewInvalidFormatError("HAR object is nil")
	}
	if index < 0 || index >= len(lh.Log.Entries) {
		return nil, NewInvalidValueError("index", index, "index out of range")
	}
	return &lh.Log.Entries[index], nil
}

// GetEntriesCount returns the number of entries.
func (lh *LazyHar) GetEntriesCount() int {
	if lh == nil {
		return 0
	}
	return len(lh.Log.Entries)
}

// GetResponseContent returns the response content for the entry at the specified index.
func (lh *LazyHar) GetResponseContent(index int) (*LazyContent, error) {
	entry, err := lh.GetEntry(index)
	if err != nil {
		return nil, err
	}
	return entry.Response.Content, nil
}

// GetResponseText returns the response text for the entry at the specified index.
func (lh *LazyHar) GetResponseText(index int) (*string, error) {
	content, err := lh.GetResponseContent(index)
	if err != nil {
		return nil, err
	}
	if content == nil {
		return nil, nil
	}
	return content.GetText()
}
