package har

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// Test AddEntryFromHTTPWithMeta: metadata population and actual start time.
func TestAddEntryFromHTTPWithMeta(t *testing.T) {
	body := bytes.NewBufferString(`{"k":"v"}`)
	req := httptest.NewRequest(http.MethodPost, "https://api.example.com/users?token=abc&page=1", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")

	resp := &http.Response{
		StatusCode: 201,
		Status:     "201 Created",
		Proto:      "HTTP/2.0",
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: nopBodyReadCloser(bytes.NewBufferString(`{"id":1}`)),
	}

	started := time.Date(2024, 7, 4, 10, 0, 0, 0, time.UTC)
	eb := NewHarBuilder().AddEntryFromHTTPWithMeta(req, resp, started, 250*time.Millisecond, EntryMeta{
		ServerIPAddress: "10.0.0.1",
		Connection:      "conn-7",
		Pageref:         "page_1",
		Priority:        "High",
		ResourceType:    "xhr",
		Comment:         "captured by mapper",
	})
	if eb == nil {
		t.Fatal("EntryBuilder should not be nil")
	}
	// Post-capture customization.
	eb.AddRequestHeader("X-Trace", "trace-1").EndEntry()

	h := eb.EndEntry().Build()
	if len(h.Log.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(h.Log.Entries))
	}
	e := h.Log.Entries[0]

	// Start time should use the supplied value, not time.Now().
	if !e.StartedDateTime.Equal(started) {
		t.Errorf("StartedDateTime = %v, want %v", e.StartedDateTime, started)
	}
	if e.ServerIPAddress != "10.0.0.1" {
		t.Errorf("ServerIPAddress = %q, want 10.0.0.1", e.ServerIPAddress)
	}
	if e.Connection != "conn-7" {
		t.Errorf("Connection = %q, want conn-7", e.Connection)
	}
	if e.Pageref != "page_1" {
		t.Errorf("Pageref = %q, want page_1", e.Pageref)
	}
	if e.Priority != "High" {
		t.Errorf("Priority = %q, want High", e.Priority)
	}
	if e.ResourceType != "xhr" {
		t.Errorf("ResourceType = %q, want xhr", e.ResourceType)
	}
	if e.Comment != "captured by mapper" {
		t.Errorf("Comment = %q, want 'captured by mapper'", e.Comment)
	}
	// Time should equal the duration in milliseconds.
	if e.Time != 250 {
		t.Errorf("Time = %v, want 250", e.Time)
	}
	// Request headers should include Authorization and the appended X-Trace.
	if e.Request.GetHeader("Authorization") != "Bearer secret" {
		t.Errorf("missing Authorization header")
	}
	if e.Request.GetHeader("X-Trace") != "trace-1" {
		t.Errorf("missing X-Trace header added via EntryBuilder")
	}
	// HeadersSize should be estimated automatically (not -1).
	if e.Request.HeadersSize <= 0 {
		t.Errorf("HeadersSize = %d, want > 0", e.Request.HeadersSize)
	}
	// Request body should be recorded.
	if e.Request.PostData == nil || e.Request.PostData.Text != `{"k":"v"}` {
		t.Errorf("PostData.Text = %v, want raw body", e.Request.PostData)
	}
	// Response status.
	if e.Response.Status != 201 {
		t.Errorf("Response.Status = %d, want 201", e.Response.Status)
	}
}

// Test automatic base64 encoding of binary response bodies.
func TestAddEntryFromHTTPWithMetaBinaryBody(t *testing.T) {
	binaryData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A} // PNG header.
	req := httptest.NewRequest(http.MethodGet, "https://cdn.example.com/logo.png", nil)
	resp := &http.Response{
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		Header:     http.Header{"Content-Type": []string{"image/png"}},
		Body:       nopBodyReadCloser(bytes.NewBuffer(binaryData)),
	}

	h := NewHarBuilder().AddEntryFromHTTPWithMeta(req, resp, time.Now(), 50*time.Millisecond, EntryMeta{}).EndEntry().Build()
	e := h.Log.Entries[0]

	if e.Response.Content.Encoding != "base64" {
		t.Errorf("Encoding = %q, want base64", e.Response.Content.Encoding)
	}
	decoded, err := base64.StdEncoding.DecodeString(e.Response.Content.Text)
	if err != nil {
		t.Fatalf("Text is not valid base64: %v", err)
	}
	if !bytes.Equal(decoded, binaryData) {
		t.Errorf("decoded body mismatch")
	}
	if e.Response.Content.Size != len(binaryData) {
		t.Errorf("Content.Size = %d, want %d", e.Response.Content.Size, len(binaryData))
	}
}

// Test backward compatibility of AddEntryFromHTTP (startedDateTime is current time; no metadata).
func TestAddEntryFromHTTPBackwardCompat(t *testing.T) {
	before := time.Now()
	req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	h := NewHarBuilder().AddEntryFromHTTP(req, nil, 100*time.Millisecond).Build()
	after := time.Now()

	e := h.Log.Entries[0]
	if e.StartedDateTime.Before(before) || e.StartedDateTime.After(after.Add(time.Second)) {
		t.Errorf("StartedDateTime %v not in expected range [%v, %v]", e.StartedDateTime, before, after)
	}
	if e.ServerIPAddress != "" {
		t.Errorf("ServerIPAddress should be empty, got %q", e.ServerIPAddress)
	}
}

// Test single-entry JSONL append and streaming-read round trip.
func TestAppendEntryToJSONLFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "archive.jsonl")

	e1 := Entries{StartedDateTime: time.Now(), Request: Request{Method: "GET", URL: "https://a.com/"}}
	e2 := Entries{StartedDateTime: time.Now(), Request: Request{Method: "POST", URL: "https://b.com/"}}

	if err := AppendEntryToJSONLFile(path, e1); err != nil {
		t.Fatalf("append e1: %v", err)
	}
	if err := AppendEntryToJSONLFile(path, e2); err != nil {
		t.Fatalf("append e2: %v", err)
	}

	// Verify the file contains two JSONL lines.
	data, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	// Stream the file.
	var got []Entries
	err := ForEachEntryFromReader(bytes.NewReader(data), func(entry Entries) error {
		got = append(got, entry)
		return nil
	})
	if err != nil {
		t.Fatalf("ForEachEntryFromReader: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries read, got %d", len(got))
	}
	if got[0].Request.URL != "https://a.com/" {
		t.Errorf("first URL = %q", got[0].Request.URL)
	}
	if got[1].Request.Method != "POST" {
		t.Errorf("second method = %q", got[1].Request.Method)
	}
}

// Test SafeRecorder concurrency safety (run with the race detector).
func TestSafeRecorderConcurrent(t *testing.T) {
	sr := NewSafeRecorder()
	var wg sync.WaitGroup
	const n = 50

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			body := bytes.NewBufferString("x")
			req := httptest.NewRequest(http.MethodPost, "https://api.example.com/item", body)
			resp := &http.Response{
				StatusCode: 200,
				Proto:      "HTTP/1.1",
				Header:     http.Header{"Content-Type": []string{"text/plain"}},
				Body:       nopBodyReadCloser(bytes.NewBufferString("ok")),
			}
			sr.CaptureWithMeta(req, resp, time.Now(), 10*time.Millisecond, EntryMeta{
				Connection:      "shared",
				ServerIPAddress: "10.0.0.1",
			})
		}(i)
	}
	wg.Wait()

	if got := sr.EntryCount(); got != n {
		t.Fatalf("EntryCount = %d, want %d", got, n)
	}

	// Exporting a snapshot should not affect subsequent operations.
	copy := sr.ToHarCopy()
	if len(copy.Log.Entries) != n {
		t.Errorf("copy has %d entries, want %d", len(copy.Log.Entries), n)
	}

	// Write to disk.
	path := filepath.Join(t.TempDir(), "out.har")
	if err := sr.SaveToFile(path); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("output file not created: %v", err)
	}
}

// Test exported conversion helpers.
func TestHTTPConvertHelpers(t *testing.T) {
	h := http.Header{
		"X-A": []string{"a1", "a2"},
		"X-B": []string{"b1"},
	}
	hdrs := HeadersFromHTTP(h)
	if len(hdrs) != 3 {
		t.Errorf("HeadersFromHTTP len = %d, want 3", len(hdrs))
	}

	cks := CookiesFromHTTP([]*http.Cookie{
		{Name: "sid", Value: "v", HttpOnly: true, Secure: true},
	})
	if len(cks) != 1 || !cks[0].HTTPOnly || !cks[0].Secure {
		t.Errorf("CookiesFromHTTP wrong: %+v", cks)
	}

	// Form parameter parsing.
	req := httptest.NewRequest(http.MethodPost, "https://x.com/", bytes.NewBufferString("a=1&b=2&c"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	pd, size := PostDataFromRequest(req)
	if pd == nil || len(pd.Params) != 3 {
		t.Fatalf("PostData params = %v", pd)
	}
	if pd.Params[0].Name != "a" || pd.Params[0].Value != "1" {
		t.Errorf("first param = %+v", pd.Params[0])
	}
	if size != 9 {
		t.Errorf("body size = %d, want 9", size)
	}
}

// nopBodyReadCloser wraps a Reader as a ReadCloser with a no-op Close,
// for constructing *http.Response.Body in tests (httptest Response.Body usually needs to be set manually).
type nopReadCloser struct {
	r *bytes.Buffer
}

func (n *nopReadCloser) Read(p []byte) (int, error) { return n.r.Read(p) }
func (n *nopReadCloser) Close() error               { return nil }

func nopBodyReadCloser(b *bytes.Buffer) *nopReadCloser { return &nopReadCloser{r: b} }
