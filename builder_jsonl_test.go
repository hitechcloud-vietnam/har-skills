package har

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- WriteEntryToWriter coverage tests ---

func TestWriteEntryToWriterNormal(t *testing.T) {
	entry := Entries{
		StartedDateTime: time.Now(),
		Time:            42,
		Request:         Request{URL: "https://example.com"},
	}

	var buf bytes.Buffer
	err := WriteEntryToWriter(&buf, entry)
	require.NoError(t, err)

	// Output should be valid JSON (json.Encoder.Encode appends a newline, as required by JSONL).
	line := strings.TrimRight(buf.String(), "\n")
	assert.True(t, json.Valid([]byte(line)), "Output should be valid JSON")
	assert.NotContains(t, line, "\n", "JSON content should be a single line")

	// Deserialize and verify the content.
	var decoded Entries
	require.NoError(t, json.Unmarshal([]byte(line), &decoded))
	assert.Equal(t, entry.Request.URL, decoded.Request.URL)
	assert.Equal(t, entry.Time, decoded.Time)
}

func TestWriteEntryToWriterNilWriter(t *testing.T) {
	entry := Entries{Request: Request{URL: "https://example.com"}}
	err := WriteEntryToWriter(nil, entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "writer")
}

func TestWriteEntryToWriterEncodeError(t *testing.T) {
	// Entries are always JSON-encodable, so an encode error cannot be triggered.
	// Instead, test a writer error using errWriter.
	entry := Entries{Request: Request{URL: "https://example.com"}}
	err := WriteEntryToWriter(errWriter{}, entry)
	assert.Error(t, err)
}

func TestWriteEntryToWriterShortWrite(t *testing.T) {
	entry := Entries{Request: Request{URL: "https://example.com"}}
	err := WriteEntryToWriter(jsonlShortWriter{}, entry)
	assert.Error(t, err)
}

// --- AppendEntryToJSONLFile coverage tests ---

func TestAppendEntryToJSONLFileCreate(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "archive.jsonl")

	entry := Entries{
		StartedDateTime: time.Now(),
		Request:         Request{URL: "https://example.com/1"},
	}

	// The file is created automatically if it does not exist.
	err := AppendEntryToJSONLFile(path, entry)
	require.NoError(t, err)

	// Verify the file contents.
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.True(t, json.Valid(data))
}

func TestAppendEntryToJSONLFileAppend(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "archive.jsonl")

	// Write two entries first.
	for i := 0; i < 2; i++ {
		entry := Entries{
			Request: Request{URL: "https://example.com/" + string(rune('a'+i))},
		}
		require.NoError(t, AppendEntryToJSONLFile(path, entry))
	}

	// Verify that the file has two lines.
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	assert.Len(t, lines, 2)
}

func TestAppendEntryToJSONLFileInvalidPath(t *testing.T) {
	// The path points to a directory rather than a file.
	entry := Entries{Request: Request{URL: "https://example.com"}}
	err := AppendEntryToJSONLFile("/tmp/", entry)
	assert.Error(t, err)
}

// --- ForEachEntryFromReader coverage tests ---

func TestForEachEntryFromReaderNormal(t *testing.T) {
	entries := []Entries{
		{Request: Request{URL: "https://example.com/1"}},
		{Request: Request{URL: "https://example.com/2"}},
		{Request: Request{URL: "https://example.com/3"}},
	}

	var buf bytes.Buffer
	for _, e := range entries {
		require.NoError(t, WriteEntryToWriter(&buf, e))
	}

	var collected []Entries
	err := ForEachEntryFromReader(&buf, func(e Entries) error {
		collected = append(collected, e)
		return nil
	})
	require.NoError(t, err)
	assert.Len(t, collected, 3)
}

func TestForEachEntryFromReaderCallbackError(t *testing.T) {
	entries := []Entries{
		{Request: Request{URL: "https://example.com/1"}},
		{Request: Request{URL: "https://example.com/2"}},
	}

	var buf bytes.Buffer
	for _, e := range entries {
		require.NoError(t, WriteEntryToWriter(&buf, e))
	}

	count := 0
	err := ForEachEntryFromReader(&buf, func(e Entries) error {
		count++
		if count >= 2 {
			return errors.New("stop")
		}
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stop")
}

func TestForEachEntryFromReaderNilReader(t *testing.T) {
	err := ForEachEntryFromReader(nil, func(e Entries) error { return nil })
	assert.Error(t, err)
}

func TestForEachEntryFromReaderNilCallback(t *testing.T) {
	var buf bytes.Buffer
	err := ForEachEntryFromReader(&buf, nil)
	assert.Error(t, err)
}

func TestForEachEntryFromReaderEmpty(t *testing.T) {
	err := ForEachEntryFromReader(strings.NewReader(""), func(e Entries) error { return nil })
	assert.NoError(t, err)
}

func TestForEachEntryFromReaderInvalidJSON(t *testing.T) {
	err := ForEachEntryFromReader(strings.NewReader("not json\n"), func(e Entries) error { return nil })
	assert.Error(t, err)
}

func TestForEachEntryFromReaderSkipInvalidContinue(t *testing.T) {
	// ForEachEntryFromReader returns an error when it encounters an invalid line (the current implementation does not skip it).
	// This test verifies that behavior.
	valid := Entries{Request: Request{URL: "https://example.com"}}
	var buf bytes.Buffer
	require.NoError(t, WriteEntryToWriter(&buf, valid))
	buf.WriteString("not json\n")

	count := 0
	err := ForEachEntryFromReader(&buf, func(e Entries) error {
		count++
		return nil
	})
	assert.Error(t, err)
	assert.Equal(t, 1, count) // The first valid entry was processed.
}

// --- Edge case: URL-encoded body round-trip through JSONL ---

func TestJSONLEntryWithURLEncodedBody(t *testing.T) {
	entry := Entries{
		Request: Request{
			URL:      "https://example.com/form",
			Method:   "POST",
			PostData: &PostData{Text: "key=value&secret=hidden"},
		},
	}

	var buf bytes.Buffer
	require.NoError(t, WriteEntryToWriter(&buf, entry))

	var decoded Entries
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
	assert.Equal(t, "key=value&secret=hidden", decoded.Request.PostData.Text)
}

// --- Edge case: large entry performance (verify no panic; no deduplication)---

func TestJSONLLargeEntry(t *testing.T) {
	// Create an entry with a large body (simulating a large response).
	largeBody := strings.Repeat("x", 10000)
	entry := Entries{
		Response: Response{
			Content: Content{
				Text: largeBody,
				Size: len(largeBody),
			},
		},
	}

	var buf bytes.Buffer
	require.NoError(t, WriteEntryToWriter(&buf, entry))
	assert.Greater(t, buf.Len(), 10000)
}

// --- Edge case: entry containing special (non-ASCII) characters.---

func TestJSONLEntryWithNonASCII(t *testing.T) {
	entry := Entries{
		Request: Request{
			URL: "https://example.com/测试?name=张三",
		},
	}

	var buf bytes.Buffer
	require.NoError(t, WriteEntryToWriter(&buf, entry))

	var decoded Entries
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
	assert.Contains(t, decoded.Request.URL, "测试")
}

// --- Helper types. ---

type errWriter struct{}

func (errWriter) Write(p []byte) (int, error) { return 0, errors.New("write error") }

type jsonlShortWriter struct{}

func (jsonlShortWriter) Write(p []byte) (int, error) { return len(p) - 1, io.ErrShortWrite }
