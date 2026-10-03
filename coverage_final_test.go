package har

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// This file contains final tests targeting 100% coverage for remaining branches in builder.go, decode.go,
// http_convert.go, and redact.go. All test functions use the TestCov prefix
// to avoid conflicts with existing tests.

// --- builder.go ---

// Cover AddEntryFromHTTPWithMeta nil-HarBuilder branch (builder.go:163-165):
// A nil *HarBuilder makes ensureHar return nil, so har == nil and the method returns nil.
func TestCovAddEntryFromHTTPWithMeta_NilBuilder(t *testing.T) {
	body := bytes.NewBufferString(`{"k":"v"}`)
	req := httptest.NewRequest(http.MethodPost, "https://api.example.com", body)
	resp := &http.Response{
		StatusCode: 200,
		Body:       nopBodyReadCloser(bytes.NewBufferString(`{"id":1}`)),
	}
	var b *HarBuilder // nil receiver causes ensureHar to return nil
	eb := b.AddEntryFromHTTPWithMeta(req, resp, time.Now(), 0, EntryMeta{})
	if eb != nil {
		t.Fatalf("expected nil EntryBuilder for nil HarBuilder, got %v", eb)
	}
}

// Cover applyEntryMeta nil-entry branch (builder.go:266-268) and
// InitiatorLine>0 branch (builder.go:286-288).
func TestCovApplyEntryMeta_Branches(t *testing.T) {
	// nil entry -> early return (line 266-268)
	applyEntryMeta(nil, EntryMeta{ServerIPAddress: "1.2.3.4"})

	// InitiatorType + InitiatorLine>0 -> sets Initiator with LineNumber (line 286-288)
	e := &Entries{}
	applyEntryMeta(e, EntryMeta{
		InitiatorType: "script",
		InitiatorURL:  "https://example.com/app.js",
		InitiatorLine: 42,
	})
	if e.Initiator.Type != "script" || e.Initiator.URL != "https://example.com/app.js" {
		t.Fatalf("Initiator not set correctly: %+v", e.Initiator)
	}
	if e.Initiator.LineNumber != 42 {
		t.Fatalf("Initiator.LineNumber = %d, want 42", e.Initiator.LineNumber)
	}
}

// Cover WriteEntryToWriter Encode-error branch (builder.go:692-694):
// Set Response.Error to func(){} (unsupported by json.Marshal) to make Encode fail.
func TestCovWriteEntryToWriter_EncodeError(t *testing.T) {
	entry := Entries{
		Request:  Request{Method: "GET", URL: "https://example.com"},
		Response: Response{Error: func() {}}, // unsupported type for json.Marshal
	}
	var buf bytes.Buffer
	err := WriteEntryToWriter(&buf, entry)
	assertHarErrorCode(t, err, ErrCodeJSONParse)
}

// Cover AppendEntryToJSONLFile empty-path branch (builder.go:703-705).
func TestCovAppendEntryToJSONLFile_EmptyPath(t *testing.T) {
	err := AppendEntryToJSONLFile("", Entries{})
	assertHarErrorCode(t, err, ErrCodeInvalidFormat)
}

// Cover SafeRecorder.ToHarCopy nil-har branch (builder.go:814-816) and
// SaveToFileWithOptions nil-har branch (builder.go:843-845).
// When SafeRecorder.recorder is nil, recorder.ToHar() calls ensureBuilder,
// which returns nil for r == nil; ToHar then returns nil and reaches the branch above.
func TestCovSafeRecorder_NilHarBranches(t *testing.T) {
	var sr *SafeRecorder // nil receiver + nil recorder

	// ToHarCopy: s == nil triggers the early return at lines 808-810; an uninitialized SafeRecorder covers recorder == nil.
	sr2 := &SafeRecorder{} // recorder == nil
	h := sr2.ToHarCopy()
	if h != nil {
		t.Fatalf("expected nil from ToHarCopy when recorder is nil, got %v", h)
	}

	// SaveToFileWithOptions: recorder == nil makes ToHar return nil and produces an error (lines 843-845).
	err := sr2.SaveToFileWithOptions("/tmp/should-not-be-created.har", false, false)
	assertHarErrorCode(t, err, ErrCodeInvalidFormat)

	// Also covers ToHarCopy on a nil SafeRecorder (lines 808-810).
	if sr.ToHarCopy() != nil {
		t.Fatalf("expected nil from nil SafeRecorder.ToHarCopy")
	}
}

// --- decode.go ---

// Cover DecompressByEncoding brotli error branch (decode.go:294-297).
// Passing "br" explicitly bypasses isBrotliData detection; corrupted Brotli data makes io.ReadAll fail.
func TestCovDecompressByEncoding_BrotliError(t *testing.T) {
	// 0x21 is a common Brotli starting byte, but the following corrupted data makes decoding fail immediately.
	badBrotli := []byte{0x21, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	_, err := DecompressByEncoding(badBrotli, "br")
	assertHarErrorCode(t, err, ErrCodeInvalidFormat)
}

// Cover DecompressByEncoding zstd DecodeAll error branch (decode.go:308-311)
// and decompressIfNeeded zstd error branch (decode.go after edit).
// Valid Zstandard magic plus a corrupted frame makes DecodeAll report a reserved block type.
func TestCovDecompress_ZstdDecodeError(t *testing.T) {
	badZstd := []byte{0x28, 0xB5, 0x2F, 0xFD, 0x00, 0x00, 0xFF, 0xFF, 0x00, 0x00}

	// Pass "zstd" explicitly to DecompressByEncoding.
	_, err := DecompressByEncoding(badZstd, "zstd")
	assertHarErrorCode(t, err, ErrCodeInvalidFormat)

	// decompressIfNeeded enters the branch through isZstdData (magic check), then DecodeAll errors.
	_, err = decompressIfNeeded(badZstd, "")
	assertHarErrorCode(t, err, ErrCodeInvalidFormat)
}

// --- http_convert.go ---

// Cover parseFormParams empty-body branch (http_convert.go:108-110) and
// empty-pair continue branch (http_convert.go:113-114).
func TestCovParseFormParams_Branches(t *testing.T) {
	// Empty body returns an empty slice (lines 108-110).
	if got := parseFormParams(""); len(got) != 0 {
		t.Fatalf("expected empty slice for empty body, got %v", got)
	}
	// An empty pair (an empty segment split by "&") continues (lines 113-114).
	// A pair without "=" yields Param{Name: key} (lines 118-119).
	// A key=value pair yields Param{Name, Value} (line 121).
	// "&&" produces two empty segments; "a&" has one trailing empty segment; "noeq" has no "=".
	params := parseFormParams("&&key=&noeq&k=v&")
	if len(params) != 3 {
		t.Fatalf("expected 3 params (empty pairs skipped), got %d (%v)", len(params), params)
	}
	if params[0].Name != "key" || params[0].Value != "" {
		t.Errorf("params[0] = %+v, want Name=key Value=''", params[0])
	}
	if params[1].Name != "noeq" {
		t.Errorf("params[1].Name = %q, want noeq", params[1].Name)
	}
	if params[2].Name != "k" || params[2].Value != "v" {
		t.Errorf("params[2] = %+v, want Name=k Value=v", params[2])
	}
}

// Cover isTextContentType binary application subtype branch (http_convert.go:150-152).
func TestCovIsTextContentType_BinaryApplication(t *testing.T) {
	binary := []string{
		"application/pdf",
		"application/zip",
		"application/gzip",
		"application/octet-stream",
		"application/font-woff",
		"image/png",
		"audio/mpeg",
		"video/mp4",
	}
	for _, m := range binary {
		if isTextContentType(m) {
			t.Errorf("isTextContentType(%q) = true, want false", m)
		}
	}
}

// --- redact.go ---

// Cover redactJSONBody Unmarshal-error defensive branch (redact.go:366-369).
// Call redactJSONBody directly with input that fails to unmarshal to trigger the defensive branch.
func TestCovRedactJSONBody_UnmarshalError(t *testing.T) {
	// Invalid JSON: missing value.
	text := `{"key": }`
	opts := DefaultRedactOptions()
	out, ok := redactJSONBody(text, opts, "***", nil)
	// The defensive branch should return (text, false).
	if ok != false {
		t.Errorf("expected ok=false for invalid JSON, got %v", ok)
	}
	if out != text {
		t.Errorf("expected output==input for defensive branch, got %q", out)
	}
}
