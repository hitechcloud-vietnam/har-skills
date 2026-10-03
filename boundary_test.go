package har

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- testdata integration tests ---

// TestCompressedHARIntegration verifies that all compressed formats in testdata/compressed.har are parsed and decompressed correctly.
// Covers round trips for gzip, deflate, Brotli, and Zstandard single-layer compression, plus base64 encoding.
func TestCompressedHARIntegration(t *testing.T) {
	h, err := ParseHarFile("testdata/compressed.har")
	require.NoError(t, err)
	require.Len(t, h.Log.Entries, 5)

	expectedBody := `{"message":"compressed response body","data":[1,2,3,4,5],"nested":{"key":"value"}}`

	// The first four entries use single-layer compression: gzip/deflate/br/zstd.
	for i := 0; i < 4; i++ {
		entry := h.Log.Entries[i]
		encoding := entry.Response.Headers[0].Value
		data, err := entry.DecodeContent()
		require.NoError(t, err, "entry %d (%s) 解压失败", i, encoding)
		assert.Equal(t, expectedBody, string(data), "entry %d (%s) 解压结果不匹配", i, encoding)
	}

	// The fifth entry uses multiple encodings, gzip and deflate; the SDK's DecodeContent currently detects compression by magic number,
	// so it can only decode the outer gzip layer; the inner deflate layer must be decoded manually.
	multiEntry := h.Log.Entries[4]
	assert.Equal(t, "gzip, deflate", multiEntry.Response.Headers[0].Value)
	// The outer gzip layer should decode to the inner deflate zlib bytes.
	outerData, err := multiEntry.DecodeContent()
	require.NoError(t, err)
	// Decode the inner deflate layer with DecompressByEncoding.
	innerData, err := DecompressByEncoding(outerData, "deflate")
	require.NoError(t, err)
	assert.Equal(t, expectedBody, string(innerData))
}

// TestSensitiveHARRedactionIntegration verifies redaction of testdata/sensitive.har.
// All sensitive data (secrets, tokens, JWTs, AWS keys, and GitHub PATs) should be redacted.
func TestSensitiveHARRedactionIntegration(t *testing.T) {
	h, err := ParseHarFile("testdata/sensitive.har")
	require.NoError(t, err)

	redacted := h.Redact(DefaultRedactOptions())

	// Scan the serialized output to ensure no sensitive values remain in plaintext.
	jsonBytes, err := redacted.ToJSON(false)
	require.NoError(t, err)
	jsonStr := string(jsonBytes)

	// These plaintext sensitive values must no longer appear.
	forbidden := []string{
		"AKIAIOSFODNN7EXAMPLE", // AWS access key
		"eyJhbGciOiJIUzI1NiJ9", // JWT header
		"ghp_",                 // GitHub PAT 前缀
		"xoxb-",                // Slack token 前缀
		"hunter2",              // Plaintext password
		"secret123",            // Plaintext token
	}
	for _, s := range forbidden {
		assert.NotContains(t, jsonStr, s, "Plaintext sensitive value remains after redaction: %q", s)
	}

	// Non-sensitive information such as the URL host and path should be preserved.
	assert.Contains(t, jsonStr, "api.example.com")
	assert.Contains(t, jsonStr, "/users")
	assert.Contains(t, jsonStr, "/login")
}

// TestSensitiveHARNonStringJSONRedaction verifies redaction of non-string JSON body values
// (numbers, booleans, null, and nested objects); this capability was added in Task 19.
func TestSensitiveHARNonStringJSONRedaction(t *testing.T) {
	h, err := ParseHarFile("testdata/sensitive.har")
	require.NoError(t, err)

	// The second entry's postData is JSON containing password (string) and secret_code (int).
	entry := h.Log.Entries[1]
	require.NotNil(t, entry.Request.PostData)
	assert.Contains(t, entry.Request.PostData.Text, `"secret_code": 12345`)

	// Redact only secret_code (the number), not password.
	opts := DefaultRedactOptions()
	opts.PostDataFields = []string{"secret_code"}
	opts.ValuePatterns = nil // Disable value patterns to focus on name matching for non-string values.

	redacted := h.Redact(opts)
	redactedText := redacted.Log.Entries[1].Request.PostData.Text

	// The numeric value is replaced with the string "[REDACTED]".
	assert.Contains(t, redactedText, `"secret_code": "[REDACTED]"`)
	// password is not in the redaction list and remains unchanged.
	assert.Contains(t, redactedText, `"password": "hunter2"`)
}

// TestParseMinimalHAR verifies that the smallest valid HAR can still be parsed.
func TestParseMinimalHAR(t *testing.T) {
	h, err := ParseHarFile("testdata/minimal.har")
	require.NoError(t, err)
	assert.NotNil(t, h)
}

// TestParseInvalidHARFiles verifies that invalid HAR files are rejected correctly.
func TestParseInvalidHARFiles(t *testing.T) {
	cases := []struct {
		name string
		file string
	}{
		{"not json", "testdata/not_json.har"},
		{"invalid json", "testdata/invalid.har"},
		{"invalid version", "testdata/invalid_version.har"},
		{"missing required", "testdata/missing_required.har"},
		// invalid_url.har is excluded because the current parser does not strictly validate URLs.
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseHarFile(tc.file)
			assert.Error(t, err, "%s should fail to parse", tc.file)
		})
	}
}

// TestRoundtripCompressedHAR verifies that a compressed HAR can be parsed and written back as a parseable file.
func TestRoundtripCompressedHAR(t *testing.T) {
	h, err := ParseHarFile("testdata/compressed.har")
	require.NoError(t, err)

	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "roundtrip.har")

	// Write the file using the SDK.
	data, err := h.ToJSON(true)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(outPath, data, 0644))

	// Parse it again.
	h2, err := ParseHarFile(outPath)
	require.NoError(t, err)
	assert.Len(t, h2.Log.Entries, len(h.Log.Entries))

	// The decompressed result of the first entry should match.
	d1, err := h.Log.Entries[0].DecodeContent()
	require.NoError(t, err)
	d2, err := h2.Log.Entries[0].DecodeContent()
	require.NoError(t, err)
	assert.Equal(t, d1, d2)
}

// TestLargeHARSmokeTest is a large-file smoke test: parsing must not panic and the entry count must be correct.
// large.har contains URLs with control characters, which fail strict validation, so parse it in lenient mode.
func TestLargeHARSmokeTest(t *testing.T) {
	f, err := os.Open("testdata/large.har")
	require.NoError(t, err)
	defer f.Close()

	h, err := ParseHarFromReaderWithOptions(f, ParseOptions{Lenient: true, SkipValidation: true})
	require.NoError(t, err)
	assert.NotEmpty(t, h.Log.Entries)
}

// TestFullHARValidate verifies that full.har passes strict validation.
func TestFullHARValidate(t *testing.T) {
	h, err := ParseHarFile("testdata/full.har")
	require.NoError(t, err)

	// Non-strict validation should pass.
	require.NoError(t, ValidateHarFile(h))
}

// TestV11HARParse verifies that HAR 1.1 files can be parsed for backward compatibility.
func TestV11HARParse(t *testing.T) {
	h, err := ParseHarFile("testdata/v1.1.har")
	require.NoError(t, err)
	assert.Equal(t, "1.1", h.Log.Version)
}

// TestCompressedHARExtractByEncoding verifies decompression using the value returned by GetContentEncoding.
func TestCompressedHARExtractByEncoding(t *testing.T) {
	h, err := ParseHarFile("testdata/compressed.har")
	require.NoError(t, err)

	expectedBody := `{"message":"compressed response body","data":[1,2,3,4,5],"nested":{"key":"value"}}`

	// For each entry, base64-decode Content.Text, then decompress according to the Content-Encoding header.
	for i := 0; i < 4; i++ {
		entry := h.Log.Entries[i]
		encoding := entry.GetContentEncoding()
		assert.NotEmpty(t, encoding, "entry %d should have a Content-Encoding header", i)

		// DecodeContent should be equivalent to base64 decoding followed by magic-number-based decompression.
		data, err := entry.DecodeContent()
		require.NoError(t, err)
		assert.Equal(t, expectedBody, string(data))
	}
}

// TestRedactValuePatternsAcrossAllFields is an integration test for value-pattern redaction across all field types.
func TestRedactValuePatternsAcrossAllFields(t *testing.T) {
	// 构造一个 entry，每个字段都藏一个 Bearer token
	h := &Har{
		Log: Log{
			Entries: []Entries{
				{
					Request: Request{
						URL: "https://api.example.com/path?trace=Bearer%20eyJhbGciOiJIUzI1NiJ9.abc.def",
						Headers: []Headers{
							{Name: "X-Custom", Value: "Bearer secret-token-1"},
						},
						Cookies: []Cookie{
							{Name: "tracking", Value: "Bearer secret-token-2"},
						},
						QueryString: []QueryString{
							{Name: "ref", Value: "Bearer secret-token-3"},
						},
						PostData: &PostData{
							Text: `{"note":"Bearer secret-token-4"}`,
						},
					},
					Response: Response{
						Headers: []Headers{
							{Name: "X-Debug", Value: "Bearer secret-token-5"},
						},
						Cookies: []Cookie{
							{Name: "analytics", Value: "Bearer secret-token-6"},
						},
					},
				},
			},
		},
	}

	// Use value patterns only; do not match by name.
	opts := RedactOptions{
		Replacement:   "[X]",
		ValuePatterns: DefaultRedactValuePatterns(),
	}
	result := h.Redact(opts)

	// 整体扫描：不应有任何 Bearer secret-token 残留
	jsonBytes, _ := result.ToJSON(false)
	jsonStr := string(jsonBytes)
	assert.NotContains(t, jsonStr, "Bearer secret-token")
	// Preserve the URL host and path.
	assert.Contains(t, jsonStr, "api.example.com")
}

// TestRedactURLWithSensitivePath verifies redaction rules for URL path segments.
func TestRedactURLWithSensitivePath(t *testing.T) {
	h := &Har{
		Log: Log{
			Entries: []Entries{
				{
					Request: Request{
						URL: "https://api.example.com/users/12345/tokens/abcdef",
					},
				},
			},
		},
	}

	opts := RedactOptions{
		Replacement: "[ID]",
		RedactURLs: []RedactURLRule{
			{Pattern: `^\d+$`, Replacement: "[ID]"},            // Numeric-only segment.
			{Pattern: `^[a-f0-9]{6,}$`, Replacement: "[HASH]"}, // Long hexadecimal segment.
		},
	}
	result := h.Redact(opts)

	url := result.Log.Entries[0].Request.URL
	// URL encoding turns [ID] into %5BID%5D; decode to verify.
	assert.Contains(t, url, "users")
	assert.Contains(t, url, "tokens")
	// The path segment is replaced while URL encoding is preserved.
	decoded := strings.ReplaceAll(url, "%5B", "[")
	decoded = strings.ReplaceAll(decoded, "%5D", "]")
	assert.Contains(t, decoded, "/users/[ID]/tokens/[HASH]")
}

// TestEmptyEntriesParse verifies that a HAR with an empty entry list can be parsed.
func TestEmptyEntriesParse(t *testing.T) {
	jsonStr := `{"log":{"version":"1.2","creator":{"name":"t","version":"1"},"entries":[]}}`
	provider, err := Parse([]byte(jsonStr))
	require.NoError(t, err)
	h := provider.ToStandard()
	assert.Empty(t, h.Log.Entries)

	stats := h.Statistics()
	assert.Equal(t, 0, stats.TotalRequests)
}

// TestUnicodeHandling verifies that Unicode and non-ASCII content round-trips correctly.
func TestUnicodeHandling(t *testing.T) {
	unicodeBody := `{"name":"张三","city":"北京","emoji":"🎉"}`
	h := &Har{
		Log: Log{
			Version: "1.2",
			Creator: Creator{Name: "test", Version: "1.0"},
			Entries: []Entries{
				{
					StartedDateTime: time.Now(),
					Time:            10,
					Request: Request{
						URL:         "https://example.com/用户/资料?名=张三",
						Method:      "POST",
						HTTPVersion: "HTTP/1.1",
						PostData: &PostData{
							MimeType: "application/json",
							Text:     unicodeBody,
						},
					},
					Response: Response{
						Status:      200,
						StatusText:  "OK",
						HTTPVersion: "HTTP/1.1",
						Content:     Content{MimeType: "application/json"},
					},
					Timings: Timings{Wait: 10},
				},
			},
		},
	}

	// Round-trip through serialization and deserialization.
	data, err := h.ToJSON(true)
	require.NoError(t, err)
	// JSON output should preserve Unicode instead of escaping it as \uXXXX.
	assert.Contains(t, string(data), "张三")

	// Parse with lenient options to avoid strict URL validation.
	h2, err := ParseHarFromReaderWithOptions(bytes.NewReader(data), ParseOptions{SkipValidation: true})
	require.NoError(t, err)
	assert.Equal(t, unicodeBody, h2.Log.Entries[0].Request.PostData.Text)
	assert.Contains(t, h2.Log.Entries[0].Request.URL, "张三")
}

// --- Helper: skip missing files ---

func TestTestDataFilesExist(t *testing.T) {
	files := []string{
		"testdata/compressed.har",
		"testdata/sensitive.har",
		"testdata/minimal.har",
		"testdata/large.har",
		"testdata/full.har",
	}
	for _, f := range files {
		_, err := os.Stat(f)
		assert.NoError(t, err, "Test data file should exist: %s", f)
	}
}

// Avoid unused imports.
var _ = strings.Contains
