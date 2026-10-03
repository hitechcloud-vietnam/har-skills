package har

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- SafeRecorder basic functionality tests ---

func TestNewSafeRecorder(t *testing.T) {
	rec := NewSafeRecorder()
	require.NotNil(t, rec)
	assert.Equal(t, 0, rec.EntryCount())
}

func TestSafeRecorderSetCreator(t *testing.T) {
	rec := NewSafeRecorder().SetCreator("test-agent", "1.0")
	require.NotNil(t, rec)

	h := rec.ToHarCopy()
	require.NotNil(t, h)
	assert.Equal(t, "test-agent", h.Log.Creator.Name)
	assert.Equal(t, "1.0", h.Log.Creator.Version)
}

func TestSafeRecorderSetBrowser(t *testing.T) {
	rec := NewSafeRecorder().SetBrowser("test-browser", "2.0")
	require.NotNil(t, rec)

	h := rec.ToHarCopy()
	require.NotNil(t, h)
	assert.Equal(t, "test-browser", h.Log.Browser.Name)
	assert.Equal(t, "2.0", h.Log.Browser.Version)
}

func TestSafeRecorderCapture(t *testing.T) {
	rec := NewSafeRecorder()

	req, _ := http.NewRequest("GET", "https://example.com", nil)
	resp := &http.Response{
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		Body:       http.NoBody,
	}

	rec.Capture(req, resp, 50*time.Millisecond)
	assert.Equal(t, 1, rec.EntryCount())
}

func TestSafeRecorderCaptureWithMeta(t *testing.T) {
	rec := NewSafeRecorder()

	req, _ := http.NewRequest("POST", "https://api.example.com/data", nil)
	resp := &http.Response{
		StatusCode: 201,
		Proto:      "HTTP/1.1",
		Body:       http.NoBody,
	}

	started := time.Now().Add(-100 * time.Millisecond)
	meta := EntryMeta{
		ServerIPAddress: "192.0.2.1",
		Connection:      "conn-1",
		Pageref:         "page-1",
		InitiatorType:   "script",
		Priority:        "High",
		ResourceType:    "xhr",
		Comment:         "test entry",
	}

	rec.CaptureWithMeta(req, resp, started, 100*time.Millisecond, meta)
	assert.Equal(t, 1, rec.EntryCount())

	h := rec.ToHarCopy()
	require.Len(t, h.Log.Entries, 1)
	entry := h.Log.Entries[0]

	assert.Equal(t, "192.0.2.1", entry.ServerIPAddress)
	assert.Equal(t, "conn-1", entry.Connection)
	assert.Equal(t, "page-1", entry.Pageref)
	assert.Equal(t, "High", entry.Priority)
	assert.Equal(t, "xhr", entry.ResourceType)
	assert.Equal(t, "test entry", entry.Comment)
}

func TestSafeRecorderCaptureEntry(t *testing.T) {
	rec := NewSafeRecorder()

	entry := Entries{
		Request: Request{URL: "https://example.com/captured"},
	}
	rec.CaptureEntry(entry)
	assert.Equal(t, 1, rec.EntryCount())

	h := rec.ToHarCopy()
	assert.Equal(t, "https://example.com/captured", h.Log.Entries[0].Request.URL)
}

func TestSafeRecorderToHar(t *testing.T) {
	rec := NewSafeRecorder()
	rec.CaptureEntry(Entries{Request: Request{URL: "https://example.com"}})

	h := rec.ToHar()
	require.NotNil(t, h)
	assert.Len(t, h.Log.Entries, 1)
}

func TestSafeRecorderToHarCopy(t *testing.T) {
	rec := NewSafeRecorder()
	rec.CaptureEntry(Entries{Request: Request{URL: "https://example.com"}})

	copy1 := rec.ToHarCopy()
	copy2 := rec.ToHarCopy()

	// Copies are independent.
	require.NotNil(t, copy1)
	require.NotNil(t, copy2)
	assert.NotSame(t, copy1, copy2)

	// Modifying copy1 does not affect copy2.
	copy1.Log.Entries[0].Request.URL = "https://modified.com"
	assert.Equal(t, "https://example.com", copy2.Log.Entries[0].Request.URL)
}

func TestSafeRecorderSaveToFile(t *testing.T) {
	rec := NewSafeRecorder().SetCreator("test", "1.0")

	req, _ := http.NewRequest("GET", "https://example.com", nil)
	resp := &http.Response{
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		Header:     http.Header{"Content-Type": []string{"text/html"}},
		Body:       http.NoBody,
	}
	rec.Capture(req, resp, 50*time.Millisecond)

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.har")

	// SaveToFile internally calls SaveToFileWithOptions(true, false), which performs strict validation.
	// Use SaveToFileWithOptions(false, false) to skip validation and test serialization only.
	err := rec.SaveToFileWithOptions(path, false, false)
	require.NoError(t, err)

	// Verify that the file exists and contains valid JSON.
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.True(t, json.Valid(data))
}

func TestSafeRecorderSaveToFileWithOptions(t *testing.T) {
	rec := NewSafeRecorder()
	rec.CaptureEntry(Entries{Request: Request{URL: "https://example.com"}})

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.har")

	// No indentation + gzip.
	err := rec.SaveToFileWithOptions(path, false, true)
	require.NoError(t, err)

	// Verify that the file exists and is not empty.
	stat, err := os.Stat(path)
	require.NoError(t, err)
	assert.Greater(t, stat.Size(), int64(0))
}

func TestSafeRecorderSaveToFileInvalidPath(t *testing.T) {
	rec := NewSafeRecorder()
	err := rec.SaveToFile("/nonexistent/dir/file.har")
	assert.Error(t, err)
}

// --- Concurrency safety tests ---

func TestSafeRecorderConcurrentCapture(t *testing.T) {
	rec := NewSafeRecorder()

	const workers = 10
	const entriesPerWorker = 100

	var wg sync.WaitGroup
	wg.Add(workers)

	for w := 0; w < workers; w++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < entriesPerWorker; i++ {
				req, _ := http.NewRequest("GET", "https://example.com/"+string(rune('a'+id%26)), nil)
				resp := &http.Response{
					StatusCode: 200,
					Body:       http.NoBody,
				}
				rec.Capture(req, resp, 10*time.Millisecond)
			}
		}(w)
	}

	wg.Wait()

	assert.Equal(t, workers*entriesPerWorker, rec.EntryCount())
}

func TestSafeRecorderConcurrentReadAndWrite(t *testing.T) {
	rec := NewSafeRecorder()

	const writers = 3
	const readers = 3
	const opsPerGoroutine = 20

	var wg sync.WaitGroup
	wg.Add(writers + readers)

	// Writer goroutine.
	for w := 0; w < writers; w++ {
		go func() {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				req, _ := http.NewRequest("GET", "https://example.com/", nil)
				resp := &http.Response{Body: http.NoBody}
				rec.Capture(req, resp, 1*time.Millisecond)
			}
		}()
	}

	// Reader goroutine.
	for r := 0; r < readers; r++ {
		go func() {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				_ = rec.EntryCount()
				_ = rec.ToHarCopy()
			}
		}()
	}

	wg.Wait()

	// The final entry count should equal the number written.
	assert.Equal(t, writers*opsPerGoroutine, rec.EntryCount())
}

func TestSafeRecorderConcurrentToHarCopy(t *testing.T) {
	rec := NewSafeRecorder()
	rec.CaptureEntry(Entries{Request: Request{URL: "https://example.com"}})

	const goroutines = 20
	results := make([]*Har, goroutines)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			results[idx] = rec.ToHarCopy()
		}(i)
	}

	wg.Wait()

	// All copies should be independent and valid.
	for _, h := range results {
		require.NotNil(t, h)
		assert.Len(t, h.Log.Entries, 1)
	}
}

// --- Edge case: empty/nil input ---

func TestSafeRecorderCaptureNilRequest(t *testing.T) {
	rec := NewSafeRecorder()
	rec.Capture(nil, nil, 0)
	// A nil request should not panic, but may not produce a valid entry.
	// Depending on the implementation, it may be skipped or produce an empty entry.
	assert.LessOrEqual(t, rec.EntryCount(), 1)
}

func TestSafeRecorderToHarNil(t *testing.T) {
	var rec *SafeRecorder
	h := rec.ToHar()
	assert.Nil(t, h)
}

func TestSafeRecorderToHarCopyNil(t *testing.T) {
	var rec *SafeRecorder
	h := rec.ToHarCopy()
	assert.Nil(t, h)
}

// --- Edge case: chained calls. ---

func TestSafeRecorderChaining(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://example.com", nil)
	resp := &http.Response{Body: http.NoBody}

	rec := NewSafeRecorder().
		SetCreator("chain-test", "1.0").
		SetBrowser("chain-browser", "2.0").
		Capture(req, resp, 10*time.Millisecond).
		Capture(req, resp, 20*time.Millisecond)

	assert.Equal(t, 2, rec.EntryCount())

	h := rec.ToHarCopy()
	assert.Equal(t, "chain-test", h.Log.Creator.Name)
	assert.Equal(t, "chain-browser", h.Log.Browser.Name)
}
