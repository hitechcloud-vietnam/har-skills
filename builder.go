package har

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// HarBuilder provides a fluent API for building HAR files.
type HarBuilder struct {
	har *Har
}

// EntryBuilder provides a fluent API for building HAR entries.
type EntryBuilder struct {
	entry  *Entries
	parent *HarBuilder
}

func (b *HarBuilder) ensureHar() *Har {
	if b == nil {
		return nil
	}
	if b.har == nil {
		b.har = NewHar()
	}
	return b.har
}

func (r *Recorder) ensureBuilder() *HarBuilder {
	if r == nil {
		return nil
	}
	if r.builder == nil {
		r.builder = NewHarBuilder().SetCreator("go-har-recorder", "1.0")
	}
	return r.builder
}

// NewHarBuilder creates a new HAR builder.
func NewHarBuilder() *HarBuilder {
	return &HarBuilder{
		har: NewHar(),
	}
}

// SetVersion sets the HAR specification version.
func (b *HarBuilder) SetVersion(version string) *HarBuilder {
	if har := b.ensureHar(); har != nil {
		har.Log.Version = version
	}
	return b
}

// SetCreator sets creator information.
func (b *HarBuilder) SetCreator(name, version string) *HarBuilder {
	if har := b.ensureHar(); har != nil {
		har.Log.Creator = Creator{
			Name:    name,
			Version: version,
		}
	}
	return b
}

// SetBrowser sets browser information.
func (b *HarBuilder) SetBrowser(name, version string) *HarBuilder {
	if har := b.ensureHar(); har != nil {
		har.Log.Browser = Browser{
			Name:    name,
			Version: version,
		}
	}
	return b
}

// SetComment sets a comment.
func (b *HarBuilder) SetComment(comment string) *HarBuilder {
	if har := b.ensureHar(); har != nil {
		har.Log.Comment = comment
	}
	return b
}

// AddPage adds page information.
func (b *HarBuilder) AddPage(id, title string) *HarBuilder {
	if har := b.ensureHar(); har != nil {
		har.AddPage(id, title)
	}
	return b
}

// AddEntry adds an entry and returns an EntryBuilder for further configuration.
func (b *HarBuilder) AddEntry(method, url string) *EntryBuilder {
	har := b.ensureHar()
	if har == nil {
		return nil
	}
	entry := har.AddEntry(method, url, "HTTP/1.1", "")
	return &EntryBuilder{
		entry:  entry,
		parent: b,
	}
}

// AddEntryWithHTTPVersion adds an entry with the specified HTTP version.
func (b *HarBuilder) AddEntryWithHTTPVersion(method, url, httpVersion string) *EntryBuilder {
	har := b.ensureHar()
	if har == nil {
		return nil
	}
	entry := har.AddEntry(method, url, httpVersion, "")
	return &EntryBuilder{
		entry:  entry,
		parent: b,
	}
}

// AddEntryForPage adds an entry and associates it with the specified page.
func (b *HarBuilder) AddEntryForPage(method, url, pageref string) *EntryBuilder {
	har := b.ensureHar()
	if har == nil {
		return nil
	}
	entry := har.AddEntry(method, url, "HTTP/1.1", pageref)
	return &EntryBuilder{
		entry:  entry,
		parent: b,
	}
}

// AddEntryFromHTTP creates an entry from an HTTP request/response.
//
// Compatibility wrapper: startedDateTime is set to the current time, with no metadata.
// To provide the actual request start time or metadata such as server IP or connection ID, use AddEntryFromHTTPWithMeta.
//
// Note: this method consumes and closes req.Body and resp.Body. If the caller still
// needs the response body, cache a copy first (for example, with io.ReadAll, then
// restore it with io.NopCloser(bytes.NewReader(...))).
func (b *HarBuilder) AddEntryFromHTTP(req *http.Request, resp *http.Response, duration time.Duration) *HarBuilder {
	return b.addEntryFromHTTPImpl(req, resp, time.Now(), duration, EntryMeta{})
}

// AddEntryFromHTTPWithMeta creates an entry from an HTTP request/response with the actual start time and optional metadata.
//
// This is useful when wrapping the builder as a low-level library for network security
// or network-mapping systems:
//   - startedAt is the actual request start time, stored as HAR startedDateTime,
//     avoiding timing inconsistencies caused by the old wrapper's fixed time.Now().
//   - meta carries metadata that cannot be inferred from req/resp, such as serverIP,
//     connection, pageref, initiator, priority, and resourceType.
//   - Binary response bodies (images, fonts, video, etc.) are base64-encoded automatically
//     to preserve them in JSON round trips.
//   - HeadersSize is estimated automatically.
//
// It returns an *EntryBuilder for additional customization of the generated entry
// (such as adding headers or cookies); call EndEntry() to return to HarBuilder.
// Like AddEntryFromHTTP, it consumes and closes req.Body and resp.Body.
func (b *HarBuilder) AddEntryFromHTTPWithMeta(req *http.Request, resp *http.Response, startedAt time.Time, duration time.Duration, meta EntryMeta) *EntryBuilder {
	b.addEntryFromHTTPImpl(req, resp, startedAt, duration, meta)
	har := b.ensureHar()
	if har == nil || len(har.Log.Entries) == 0 {
		return nil
	}
	// Return an EntryBuilder for the appended entry so it can be customized.
	return &EntryBuilder{
		entry:  &har.Log.Entries[len(har.Log.Entries)-1],
		parent: b,
	}
}

// addEntryFromHTTPImpl is shared by AddEntryFromHTTP and AddEntryFromHTTPWithMeta.
func (b *HarBuilder) addEntryFromHTTPImpl(req *http.Request, resp *http.Response, startedAt time.Time, duration time.Duration, meta EntryMeta) *HarBuilder {
	har := b.ensureHar()
	if har == nil {
		return b
	}
	if req == nil {
		return b
	}

	requestURL := ""
	if req.URL != nil {
		requestURL = req.URL.String()
	}

	entry := Entries{
		StartedDateTime: startedAt,
		Time:            float64(duration.Milliseconds()),
		Request: Request{
			Method:      req.Method,
			URL:         requestURL,
			HTTPVersion: req.Proto,
			Headers:     HeadersFromHTTP(req.Header),
			Cookies:     CookiesFromHTTP(req.Cookies()),
			QueryString: BuildQueryStringFromURL(requestURL),
			HeadersSize: EstimateHeaderSize(HeadersFromHTTP(req.Header)),
			BodySize:    -1,
		},
		Response: Response{
			HeadersSize: -1,
			BodySize:    -1,
		},
		Timings: Timings{
			Blocked: -1,
			DNS:     -1,
			Connect: -1,
			Send:    -1,
			Wait:    float64(duration.Milliseconds()),
			Receive: -1,
			Ssl:     -1,
		},
	}

	// Read the request body (consumes req.Body).
	postData, bodySize := PostDataFromRequest(req)
	if postData != nil {
		entry.Request.PostData = postData
		entry.Request.BodySize = bodySize
	}

	// Convert the response.
	if resp != nil {
		entry.Response.Status = resp.StatusCode
		entry.Response.StatusText = resp.Status
		entry.Response.HTTPVersion = resp.Proto
		entry.Response.Headers = HeadersFromHTTP(resp.Header)
		entry.Response.HeadersSize = EstimateHeaderSize(entry.Response.Headers)

		entry.Response.Cookies = CookiesFromHTTP(resp.Cookies())

		if !isNilReader(resp.Body) {
			bodyBytes, readErr, closeErr := readAndCloseResponseBody(resp.Body)
			if bodyErr := responseBodyErrorMessage(readErr, closeErr); bodyErr != "" {
				entry.Response.Error = bodyErr
			}
			if readErr == nil {
				mimeType := resp.Header.Get("Content-Type")
				content := Content{
					Size:     len(bodyBytes),
					MimeType: mimeType,
				}
				// Base64-encode non-text bodies to preserve them in JSON round trips.
				if isTextContentType(mimeType) {
					content.Text = string(bodyBytes)
				} else {
					content.Text = base64.StdEncoding.EncodeToString(bodyBytes)
					content.Encoding = "base64"
				}
				entry.Response.Content = content
				entry.Response.BodySize = len(bodyBytes)
			}
		}
	}

	// Apply optional metadata.
	applyEntryMeta(&entry, meta)

	har.Log.Entries = append(har.Log.Entries, entry)
	return b
}

// applyEntryMeta writes non-zero fields from EntryMeta to the entry.
func applyEntryMeta(entry *Entries, meta EntryMeta) {
	if entry == nil {
		return
	}
	if meta.ServerIPAddress != "" {
		entry.ServerIPAddress = meta.ServerIPAddress
	}
	if meta.Connection != "" {
		entry.Connection = meta.Connection
	}
	if meta.Pageref != "" {
		entry.Pageref = meta.Pageref
	}
	if meta.Comment != "" {
		entry.Comment = meta.Comment
	}
	if meta.InitiatorType != "" || meta.InitiatorURL != "" {
		entry.Initiator = Initiator{
			Type: meta.InitiatorType,
			URL:  meta.InitiatorURL,
		}
		if meta.InitiatorLine > 0 {
			entry.Initiator.LineNumber = meta.InitiatorLine
		}
	}
	if meta.Priority != "" {
		entry.Priority = meta.Priority
	}
	if meta.ResourceType != "" {
		entry.ResourceType = meta.ResourceType
	}
}

// Build constructs and returns the HAR object.
func (b *HarBuilder) Build() *Har {
	return b.ensureHar()
}

// BuildJSON builds the HAR file and returns its JSON representation.
func (b *HarBuilder) BuildJSON(indent bool) ([]byte, error) {
	har := b.ensureHar()
	if har == nil {
		return nil, NewInvalidFormatError("HAR builder is nil")
	}
	return har.ToJSON(indent)
}

// BuildAndSave builds the HAR file and saves it to a file.
func (b *HarBuilder) BuildAndSave(filePath string, indent bool) error {
	har := b.ensureHar()
	if har == nil {
		return NewInvalidFormatError("HAR builder is nil")
	}
	return har.SaveToFile(filePath, indent)
}

// EntryBuilder methods.

// WithHTTPVersion sets the HTTP version.
func (eb *EntryBuilder) WithHTTPVersion(version string) *EntryBuilder {
	if eb == nil || eb.entry == nil {
		return eb
	}
	eb.entry.Request.HTTPVersion = version
	eb.entry.Response.HTTPVersion = version
	return eb
}

// WithStartedDateTime sets the request start time.
func (eb *EntryBuilder) WithStartedDateTime(t time.Time) *EntryBuilder {
	if eb == nil || eb.entry == nil {
		return eb
	}
	eb.entry.StartedDateTime = t
	return eb
}

// WithPageref sets the page reference.
func (eb *EntryBuilder) WithPageref(ref string) *EntryBuilder {
	if eb == nil || eb.entry == nil {
		return eb
	}
	eb.entry.Pageref = ref
	return eb
}

// WithServerIP sets the server IP address.
func (eb *EntryBuilder) WithServerIP(ip string) *EntryBuilder {
	if eb == nil || eb.entry == nil {
		return eb
	}
	eb.entry.ServerIPAddress = ip
	return eb
}

// WithConnection sets the connection ID.
func (eb *EntryBuilder) WithConnection(id string) *EntryBuilder {
	if eb == nil || eb.entry == nil {
		return eb
	}
	eb.entry.Connection = id
	return eb
}

// WithComment sets a comment.
func (eb *EntryBuilder) WithComment(comment string) *EntryBuilder {
	if eb == nil || eb.entry == nil {
		return eb
	}
	eb.entry.Comment = comment
	return eb
}

// AddRequestHeader adds a request header.
func (eb *EntryBuilder) AddRequestHeader(name, value string) *EntryBuilder {
	if eb == nil || eb.entry == nil {
		return eb
	}
	eb.entry.AddRequestHeader(name, value)
	return eb
}

// AddResponseHeader adds a response header.
func (eb *EntryBuilder) AddResponseHeader(name, value string) *EntryBuilder {
	if eb == nil || eb.entry == nil {
		return eb
	}
	eb.entry.AddResponseHeader(name, value)
	return eb
}

// AddCookie adds a request cookie.
func (eb *EntryBuilder) AddCookie(name, value string) *EntryBuilder {
	if eb == nil || eb.entry == nil {
		return eb
	}
	eb.entry.AddCookie(name, value)
	return eb
}

// AddResponseCookie adds a response cookie.
func (eb *EntryBuilder) AddResponseCookie(name, value string) *EntryBuilder {
	if eb == nil || eb.entry == nil {
		return eb
	}
	eb.entry.AddResponseCookie(name, value)
	return eb
}

// AddQueryParam adds a query parameter.
func (eb *EntryBuilder) AddQueryParam(name, value string) *EntryBuilder {
	if eb == nil || eb.entry == nil {
		return eb
	}
	eb.entry.AddQueryParameter(name, value)
	return eb
}

// WithPostData sets POST data.
func (eb *EntryBuilder) WithPostData(mimeType, text string) *EntryBuilder {
	if eb == nil || eb.entry == nil {
		return eb
	}
	eb.entry.SetPostData(mimeType, text)
	return eb
}

// WithPostDataParams sets POST form parameters.
func (eb *EntryBuilder) WithPostDataParams(mimeType string, params []Param) *EntryBuilder {
	if eb == nil || eb.entry == nil {
		return eb
	}
	eb.entry.SetPostDataParams(mimeType, params)
	return eb
}

// WithResponseStatus sets the response status.
func (eb *EntryBuilder) WithResponseStatus(status int, statusText string) *EntryBuilder {
	if eb == nil || eb.entry == nil {
		return eb
	}
	eb.entry.SetResponseStatus(status, statusText)
	return eb
}

// WithResponseContent sets the response content.
func (eb *EntryBuilder) WithResponseContent(size int, mimeType string) *EntryBuilder {
	if eb == nil || eb.entry == nil {
		return eb
	}
	eb.entry.SetResponseContent(size, mimeType)
	return eb
}

// WithResponseContentText sets the response content, including its text.
func (eb *EntryBuilder) WithResponseContentText(size int, mimeType, text string) *EntryBuilder {
	if eb == nil || eb.entry == nil {
		return eb
	}
	eb.entry.SetResponseContent(size, mimeType)
	eb.entry.Response.Content.Text = text
	return eb
}

// WithTimings sets timing data.
func (eb *EntryBuilder) WithTimings(blocked, dns, connect, send, wait, receive, ssl float64) *EntryBuilder {
	if eb == nil || eb.entry == nil {
		return eb
	}
	eb.entry.SetTimings(blocked, dns, connect, send, wait, receive, ssl)
	return eb
}

// WithCache sets cache data.
func (eb *EntryBuilder) WithCache(cache Cache) *EntryBuilder {
	if eb == nil || eb.entry == nil {
		return eb
	}
	eb.entry.Cache = cache
	return eb
}

// WithInitiator sets the request initiator.
func (eb *EntryBuilder) WithInitiator(initiatorType, initiatorURL string, lineNumber int) *EntryBuilder {
	if eb == nil || eb.entry == nil {
		return eb
	}
	eb.entry.Initiator = Initiator{
		Type:       initiatorType,
		URL:        initiatorURL,
		LineNumber: lineNumber,
	}
	return eb
}

// WithPriority sets the request priority.
func (eb *EntryBuilder) WithPriority(priority string) *EntryBuilder {
	if eb == nil || eb.entry == nil {
		return eb
	}
	eb.entry.Priority = priority
	return eb
}

// WithResourceType sets the resource type.
func (eb *EntryBuilder) WithResourceType(resourceType string) *EntryBuilder {
	if eb == nil || eb.entry == nil {
		return eb
	}
	eb.entry.ResourceType = resourceType
	return eb
}

// EndEntry finishes building the entry and returns the HarBuilder.
func (eb *EntryBuilder) EndEntry() *HarBuilder {
	if eb == nil {
		return nil
	}
	return eb.parent
}

// Recorder records HTTP interactions and generates HAR files.
type Recorder struct {
	builder *HarBuilder
}

// NewRecorder creates a Recorder.
func NewRecorder() *Recorder {
	return &Recorder{
		builder: NewHarBuilder().SetCreator("go-har-recorder", "1.0"),
	}
}

// SetCreator sets creator information for the recorder.
func (r *Recorder) SetCreator(name, version string) *Recorder {
	if builder := r.ensureBuilder(); builder != nil {
		builder.SetCreator(name, version)
	}
	return r
}

// SetBrowser sets browser information.
func (r *Recorder) SetBrowser(name, version string) *Recorder {
	if builder := r.ensureBuilder(); builder != nil {
		builder.SetBrowser(name, version)
	}
	return r
}

// Capture records an HTTP request/response.
func (r *Recorder) Capture(req *http.Request, resp *http.Response, duration time.Duration) *Recorder {
	if builder := r.ensureBuilder(); builder != nil {
		builder.AddEntryFromHTTP(req, resp, duration)
	}
	return r
}

// CaptureEntry records a prebuilt HAR entry.
func (r *Recorder) CaptureEntry(entry Entries) *Recorder {
	if builder := r.ensureBuilder(); builder != nil {
		builder.ensureHar().Log.Entries = append(builder.ensureHar().Log.Entries, entry)
	}
	return r
}

// EntryCount returns the number of recorded entries.
func (r *Recorder) EntryCount() int {
	builder := r.ensureBuilder()
	if builder == nil {
		return 0
	}
	// When builder is non-nil, ensureHar must return a non-nil Har because
	// NewHarBuilder initializes b.har.
	return len(builder.ensureHar().Log.Entries)
}

// ToHar generates a HAR object.
func (r *Recorder) ToHar() *Har {
	builder := r.ensureBuilder()
	if builder == nil {
		return nil
	}
	return builder.Build()
}

// SaveToFile saves the recording to a file.
func (r *Recorder) SaveToFile(path string) error {
	builder := r.ensureBuilder()
	if builder == nil {
		return NewInvalidFormatError("Recorder is nil")
	}
	return builder.BuildAndSave(path, true)
}

// ToJSON generates JSON output.
func (r *Recorder) ToJSON(indent bool) ([]byte, error) {
	builder := r.ensureBuilder()
	if builder == nil {
		return nil, NewInvalidFormatError("Recorder is nil")
	}
	return builder.BuildJSON(indent)
}

// WriteToWriter writes the HAR file to the specified Writer.
func WriteToWriter(har *Har, w io.Writer, indent bool) error {
	if har == nil {
		return NewInvalidFormatError("HAR object is nil")
	}
	if isNilWriter(w) {
		return NewInvalidFormatError("writer is nil")
	}

	data, err := har.ToJSON(indent)
	if err != nil {
		return err
	}

	return writeAllToWriter(w, data, "failed to write HAR data")
}

// WriteEntriesToWriter writes HAR entries to a Writer in JSON Lines format.
// Each line contains one entry as a JSON object, suitable for streaming.
func WriteEntriesToWriter(har *Har, w io.Writer) error {
	if har == nil {
		return NewInvalidFormatError("HAR object is nil")
	}
	if isNilWriter(w) {
		return NewInvalidFormatError("writer is nil")
	}

	for _, entry := range har.Log.Entries {
		var buf bytes.Buffer
		entryEncoder := json.NewEncoder(&buf)
		if err := entryEncoder.Encode(entry); err != nil {
			return NewJSONParseError("failed to encode HAR entry", err)
		}
		if err := writeAllToWriter(w, buf.Bytes(), "failed to write HAR entries"); err != nil {
			return err
		}
	}

	return nil
}

// ReadEntriesFromReader reads entries in JSON Lines format from a Reader.
func ReadEntriesFromReader(r io.Reader) ([]Entries, error) {
	if isNilReader(r) {
		return nil, NewInvalidFormatError("reader is nil")
	}

	var entries []Entries
	decoder := json.NewDecoder(r)

	for {
		var entry Entries
		if err := decoder.Decode(&entry); err != nil {
			if err == io.EOF {
				break
			}
			return entries, WrapJSONUnmarshalError(err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// ToJSONLines converts HAR entries to a JSON Lines string.
func (h *Har) ToJSONLines() (string, error) {
	if h == nil {
		return "", NewInvalidFormatError("HAR object is nil")
	}

	var buf bytes.Buffer
	err := WriteEntriesToWriter(h, &buf)
	return buf.String(), err
}

// WriteEntryToWriter writes a single HAR entry to a Writer in JSON Lines format
// (one JSON object per line). It supports low-memory, long-term archiving by
// writing each captured request immediately instead of buffering a complete *Har.
func WriteEntryToWriter(w io.Writer, entry Entries) error {
	if isNilWriter(w) {
		return NewInvalidFormatError("writer is nil")
	}
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	if err := encoder.Encode(entry); err != nil {
		return NewJSONParseError("failed to encode HAR entry", err)
	}
	return writeAllToWriter(w, buf.Bytes(), "failed to write HAR entry")
}

// AppendEntryToJSONLFile appends a single HAR entry to a file in JSON Lines format.
// The file is created if it does not exist; otherwise, O_APPEND writes to it without
// reading existing content, keeping memory usage constant. This is suitable for
// long-term archiving: each request occupies one line and can later be read with
// ForEachEntryFromReader or ReadEntriesFromReader, or split with commands such as split --by.
func AppendEntryToJSONLFile(path string, entry Entries) error {
	if path == "" {
		return NewInvalidFormatError("path is empty")
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return NewFileSystemError("failed to open HAR JSONL file", err)
	}
	defer f.Close()
	return WriteEntryToWriter(f, entry)
}

// ForEachEntryFromReader streams entries in JSON Lines format and calls fn for each entry.
// Unlike ReadEntriesFromReader, it does not load all entries into memory at once;
// it decodes each entry and immediately passes it to the callback, making it suitable
// for very large archives. Iteration stops and returns the error if fn returns a non-nil error.
func ForEachEntryFromReader(r io.Reader, fn func(entry Entries) error) error {
	if isNilReader(r) {
		return NewInvalidFormatError("reader is nil")
	}
	if fn == nil {
		return NewInvalidFormatError("callback is nil")
	}
	decoder := json.NewDecoder(r)
	for {
		var entry Entries
		if err := decoder.Decode(&entry); err != nil {
			if err == io.EOF {
				return nil
			}
			return WrapJSONUnmarshalError(err)
		}
		if err := fn(entry); err != nil {
			return err
		}
	}
}

// SafeRecorder is a concurrency-safe Recorder suitable for concurrent capture and
// archiving by network-mapping systems. It uses sync.Mutex to protect all reads and writes.
// Use it directly for continuous accumulation with one-time export, or concurrent Capture
// calls followed by periodic SaveToFile calls; callers do not need to add their own locks.
type SafeRecorder struct {
	mu       sync.Mutex
	recorder *Recorder
}

// NewSafeRecorder creates a concurrency-safe Recorder.
func NewSafeRecorder() *SafeRecorder {
	return &SafeRecorder{recorder: NewRecorder()}
}

// SetCreator sets creator information for the recorder.
func (s *SafeRecorder) SetCreator(name, version string) *SafeRecorder {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recorder.SetCreator(name, version)
	return s
}

// SetBrowser sets browser information.
func (s *SafeRecorder) SetBrowser(name, version string) *SafeRecorder {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recorder.SetBrowser(name, version)
	return s
}

// Capture safely records an HTTP request/response concurrently.
// Note: it consumes and closes req.Body and resp.Body. Cache a copy first if the caller still needs the response body.
func (s *SafeRecorder) Capture(req *http.Request, resp *http.Response, duration time.Duration) *SafeRecorder {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recorder.Capture(req, resp, duration)
	return s
}

// CaptureWithMeta safely records an HTTP request/response concurrently, with the actual start time and metadata.
// See HarBuilder.AddEntryFromHTTPWithMeta.
func (s *SafeRecorder) CaptureWithMeta(req *http.Request, resp *http.Response, startedAt time.Time, duration time.Duration, meta EntryMeta) *SafeRecorder {
	s.mu.Lock()
	defer s.mu.Unlock()
	builder := s.recorder.ensureBuilder()
	if builder != nil {
		builder.AddEntryFromHTTPWithMeta(req, resp, startedAt, duration, meta)
	}
	return s
}

// CaptureEntry safely records a prebuilt HAR entry without touching any body.
func (s *SafeRecorder) CaptureEntry(entry Entries) *SafeRecorder {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recorder.CaptureEntry(entry)
	return s
}

// EntryCount returns the number of recorded entries.
func (s *SafeRecorder) EntryCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.recorder.EntryCount()
}

// ToHarCopy returns a deep copy of the internal HAR, so subsequent Capture calls
// do not change the copy held by the caller. Use it to periodically export snapshots
// during concurrent archiving.
func (s *SafeRecorder) ToHarCopy() *Har {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	h := s.recorder.ToHar()
	if h == nil {
		return nil
	}
	return h.Clone()
}

// ToHar returns a pointer to the internal HAR (retrieved under lock, but its underlying
// memory may be modified by subsequent Capture calls). Use ToHarCopy for a stable snapshot.
func (s *SafeRecorder) ToHar() *Har {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.recorder.ToHar()
}

// SaveToFile safely saves the recording to a file as indented JSON.
func (s *SafeRecorder) SaveToFile(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.recorder.SaveToFile(path)
}

// SaveToFileWithOptions saves the recording with optional indentation and gzip compression.
func (s *SafeRecorder) SaveToFileWithOptions(path string, indent, gzip bool) error {
	s.mu.Lock()
	h := s.recorder.ToHar()
	s.mu.Unlock()
	if h == nil {
		return NewInvalidFormatError("Recorder is nil")
	}
	if gzip {
		return SaveToFileGzipped(h, path, indent)
	}
	return h.SaveToFile(path, indent)
}
