package har

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This test file adds coverage for parser.go and uses distinct Cov-prefixed test function names,
// to avoid duplicates or conflicts with parser_test.go.

// --- Helper functions ---

// covWriteFile writes a file under t.TempDir() and returns its full path.
func covWriteFile(t *testing.T, name string, content []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(p, content, 0644))
	return p
}

// validHarJSON returns the bytes of a minimal valid HAR JSON document.
func validHarJSON() []byte {
	return []byte(`{
		"log": {
			"version": "1.2",
			"creator": {"name": "test", "version": "1.0"},
			"entries": []
		}
	}`)
}

// --- ParseHarWithOptions branch coverage ---

// TestCovParseHarWithOptionsEmpty covers lines 13-15: empty-input branch.
func TestCovParseHarWithOptionsEmpty(t *testing.T) {
	h, err := ParseHarWithOptions(nil, DefaultParseOptions())
	assert.Nil(t, h)
	require.Error(t, err)
	he, ok := err.(*HarError)
	assert.True(t, ok)
	assert.Equal(t, ErrCodeInvalidFormat, he.Code)
}

// TestCovParseHarWithOptionsEmptySlice covers lines 13-15: zero-length slice.
func TestCovParseHarWithOptionsEmptySlice(t *testing.T) {
	h, err := ParseHarWithOptions([]byte{}, DefaultParseOptions())
	assert.Nil(t, h)
	require.Error(t, err)
}

// TestCovParseHarWithOptionsNonJSON covers lines 18-19: non-JSON branch.
func TestCovParseHarWithOptionsNonJSON(t *testing.T) {
	h, err := ParseHarWithOptions([]byte("not a json"), DefaultParseOptions())
	assert.Nil(t, h)
	require.Error(t, err)
	he, ok := err.(*HarError)
	assert.True(t, ok)
	assert.Equal(t, ErrCodeInvalidFormat, he.Code)
}

// TestCovParseHarWithOptionsNonJSONBrackets covers lines 18-19: JSON-like but invalid input (mismatched delimiters).
func TestCovParseHarWithOptionsNonJSONBrackets(t *testing.T) {
	// The prefix is { but the suffix is not }; isJSONContent returns false.
	h, err := ParseHarWithOptions([]byte("{ not closing brace"), DefaultParseOptions())
	assert.Nil(t, h)
	require.Error(t, err)
}

// TestCovParseHarWithOptionsStrictJSONError covers lines 25-28: json.Unmarshal failure in strict mode.
func TestCovParseHarWithOptionsStrictJSONError(t *testing.T) {
	// Valid JSON, but the structure cannot be mapped to Har (log is a string, not an object).
	h, err := ParseHarWithOptions([]byte(`{"log":"notobj"}`), DefaultParseOptions())
	assert.Nil(t, h)
	require.Error(t, err)
}

// TestCovParseHarWithOptionsStrictValidationError covers lines 31-34: parse succeeds but validation fails in strict mode.
func TestCovParseHarWithOptionsStrictValidationError(t *testing.T) {
	// Valid JSON structure, but version/creator.name is missing, so validateHar returns an error.
	bad := []byte(`{"log":{"entries":[]}}`)
	h, err := ParseHarWithOptions(bad, DefaultParseOptions())
	assert.Nil(t, h)
	require.Error(t, err)
}

// TestCovParseHarWithOptionsStrictSkipValidation covers line 31 (SkipValidation=true skips validation) and line 37.
func TestCovParseHarWithOptionsStrictSkipValidation(t *testing.T) {
	opts := DefaultParseOptions()
	opts.SkipValidation = true
	// Missing version, but parsing should succeed when validation is skipped.
	bad := []byte(`{"log":{"entries":[]}}`)
	h, err := ParseHarWithOptions(bad, opts)
	require.NoError(t, err)
	assert.NotNil(t, h)
}

// TestCovParseHarWithOptionsLenient covers line 41: lenient-mode branch.
func TestCovParseHarWithOptionsLenient(t *testing.T) {
	opts := DefaultParseOptions()
	opts.Lenient = true
	opts.CollectWarnings = true
	h, err := ParseHarWithOptions(validHarJSON(), opts)
	require.NoError(t, err)
	assert.NotNil(t, h)
}

// --- ParseHarFileWithOptions coverage ---

// TestCovParseHarFileWithOptionsReadError covers lines 46-49: file read failure.
func TestCovParseHarFileWithOptionsReadError(t *testing.T) {
	h, err := ParseHarFileWithOptions("/nonexistent/path/file.har", DefaultParseOptions())
	assert.Nil(t, h)
	require.Error(t, err)
	he, ok := err.(*HarError)
	assert.True(t, ok)
	assert.Equal(t, ErrCodeFileSystem, he.Code)
}

// TestCovParseHarFileWithOptionsParseError covers lines 51-57: parse failure returning HarError and calling WithMetadata.
func TestCovParseHarFileWithOptionsParseError(t *testing.T) {
	p := covWriteFile(t, "bad.har", []byte("not json"))
	h, err := ParseHarFileWithOptions(p, DefaultParseOptions())
	assert.Nil(t, h)
	require.Error(t, err)
	he, ok := err.(*HarError)
	assert.True(t, ok)
	// WithMetadata should be called, and metadata should contain filePath.
	require.NotNil(t, he.Metadata)
	_, hasPath := he.Metadata["filePath"]
	assert.True(t, hasPath)
}

// TestCovParseHarFileWithOptionsSuccess covers line 60: success path.
func TestCovParseHarFileWithOptionsSuccess(t *testing.T) {
	p := covWriteFile(t, "ok.har", validHarJSON())
	h, err := ParseHarFileWithOptions(p, DefaultParseOptions())
	require.NoError(t, err)
	assert.NotNil(t, h)
}

// --- ParseHarEnhanced coverage ---

// TestCovParseHarEnhancedHarError covers lines 65, 67-69: returns HarError.
func TestCovParseHarEnhancedHarError(t *testing.T) {
	h, he := ParseHarEnhanced([]byte("not json"))
	assert.Nil(t, h)
	require.NotNil(t, he)
	assert.Equal(t, ErrCodeInvalidFormat, he.Code)
}

// TestCovParseHarEnhancedUnknownError covers line 71: non-HarError wrapping branch.
// Note: This branch is unreachable in practice because all ParseHarWithOptions errors are *HarError.
// Attempt to exercise it with input that might produce a non-*HarError (though it cannot be triggered in theory).
func TestCovParseHarEnhancedUnknownError(t *testing.T) {
	// All error constructors inside ParseHarWithOptions return *HarError,
	// so the unknown-wrapping branch on line 71 cannot be triggered. This only verifies that the success path does not enter it.
	h, he := ParseHarEnhanced(validHarJSON())
	assert.NotNil(t, h)
	assert.Nil(t, he)
}

// TestCovParseHarEnhancedSuccess covers line 73: success path.
func TestCovParseHarEnhancedSuccess(t *testing.T) {
	h, he := ParseHarEnhanced(validHarJSON())
	assert.Nil(t, he)
	assert.NotNil(t, h)
	assert.Equal(t, "1.2", h.Log.Version)
}

// --- ParseHarFileEnhanced coverage ---

// TestCovParseHarFileEnhancedFileSystemError covers lines 77-82: filesystem error returns HarError.
func TestCovParseHarFileEnhancedFileSystemError(t *testing.T) {
	h, he := ParseHarFileEnhanced("/nonexistent/file.har")
	assert.Nil(t, h)
	require.NotNil(t, he)
}

// TestCovParseHarFileEnhancedParseError covers lines 80-81: HarError path.
func TestCovParseHarFileEnhancedParseError(t *testing.T) {
	p := covWriteFile(t, "bad.har", []byte("not json"))
	h, he := ParseHarFileEnhanced(p)
	assert.Nil(t, h)
	require.NotNil(t, he)
	assert.Equal(t, ErrCodeInvalidFormat, he.Code)
}

// TestCovParseHarFileEnhancedUnknownError covers line 84: non-HarError wrapping branch (unreachable; only verifies the success path).
func TestCovParseHarFileEnhancedUnknownError(t *testing.T) {
	p := covWriteFile(t, "ok.har", validHarJSON())
	h, he := ParseHarFileEnhanced(p)
	assert.Nil(t, he)
	assert.NotNil(t, h)
}

// TestCovParseHarFileEnhancedSuccess covers lines 86: success path。
func TestCovParseHarFileEnhancedSuccess(t *testing.T) {
	p := covWriteFile(t, "ok.har", validHarJSON())
	h, he := ParseHarFileEnhanced(p)
	assert.Nil(t, he)
	assert.NotNil(t, h)
}

// --- ParseHarLenient coverage ---

// TestCovParseHarLenient coverage行 90-95: entire function。
func TestCovParseHarLenient(t *testing.T) {
	h, err := ParseHarLenient(validHarJSON())
	require.NoError(t, err)
	assert.NotNil(t, h)
}

// TestCovParseHarLenientPartial covers lines 90-95: lenient mode with partial errors。
func TestCovParseHarLenientPartial(t *testing.T) {
	// 有 entries 但 version 错误类型
	bad := []byte(`{"log":{"version":123,"entries":[{"request":{"method":"GET","url":"http://x"}}]}}`)
	h, err := ParseHarLenient(bad)
	// entries exist, so Har is returned even if there are errors (CollectWarnings=true).
	require.NotNil(t, h)
	// An error may or may not occur, depending on parsing.
	_ = err
}

// --- ParseHarFileLenient coverage ---

// TestCovParseHarFileLenient coverage行 98-103: entire function。
func TestCovParseHarFileLenient(t *testing.T) {
	p := covWriteFile(t, "ok.har", validHarJSON())
	h, err := ParseHarFileLenient(p)
	require.NoError(t, err)
	assert.NotNil(t, h)
}

// TestCovParseHarFileLenientReadError covers lines 98-103: file does not exist。
func TestCovParseHarFileLenientReadError(t *testing.T) {
	h, err := ParseHarFileLenient("/nonexistent/file.har")
	assert.Nil(t, h)
	require.Error(t, err)
}

// --- validateHar nil branch ---

// TestCovValidateHarNil covers lines 115-117: nil 分支。
func TestCovValidateHarNil(t *testing.T) {
	err := validateHar(nil)
	require.Error(t, err)
	he, ok := err.(*HarError)
	assert.True(t, ok)
	assert.Equal(t, ErrCodeInvalidFormat, he.Code)
}

// TestCovValidateHarValid covers lines 119: valid HAR。
func TestCovValidateHarValid(t *testing.T) {
	h := &Har{Log: Log{Version: "1.2", Creator: Creator{Name: "x", Version: "1"}, Entries: []Entries{}}}
	err := validateHar(h)
	assert.NoError(t, err)
}

// --- parseLenient branch coverage ---

// TestCovParseLenientNoLog covers lines 215-217: missing log field。
func TestCovParseLenientNoLog(t *testing.T) {
	opts := DefaultParseOptions()
	opts.Lenient = true
	opts.CollectWarnings = true
	// 没有任何可解析内容（无 version/entries/pages），返回 (nil, err)
	h, err := ParseHarWithOptions([]byte(`{"foo":"bar"}`), opts)
	assert.Nil(t, h)
	require.Error(t, err)
}

// TestCovParseLenientNoLogNoCollect covers lines 227-230: error without warning collection。
func TestCovParseLenientNoLogNoCollect(t *testing.T) {
	opts := DefaultParseOptions()
	opts.Lenient = true
	opts.CollectWarnings = false
	h, err := ParseHarWithOptions([]byte(`{"foo":"bar"}`), opts)
	assert.Nil(t, h)
	require.Error(t, err)
}

// TestCovParseLenientLogNotObject covers lines 147-150: log field is not an object (cannot parse as a map)。
func TestCovParseLenientLogNotObject(t *testing.T) {
	opts := DefaultParseOptions()
	opts.Lenient = true
	opts.CollectWarnings = true
	// log 是字符串，无法 unmarshal 成 map[string]json.RawMessage
	h, err := ParseHarWithOptions([]byte(`{"log":"notobj"}`), opts)
	assert.Nil(t, h)
	require.Error(t, err)
}

// TestCovParseLenientVersionBadType covers lines 156-159: version field has the wrong type。
func TestCovParseLenientVersionBadType(t *testing.T) {
	opts := DefaultParseOptions()
	opts.Lenient = true
	opts.CollectWarnings = true
	// version 是数字，无法解析为 string；但有 entries，所以返回 har
	bad := []byte(`{"log":{"version":123,"entries":[{"request":{"method":"GET","url":"http://x"}}]}}`)
	h, err := ParseHarWithOptions(bad, opts)
	require.NotNil(t, h)
	assert.Len(t, h.Log.Entries, 1)
	// 有错误（版本解析失败），CollectWarnings=true 且有内容，返回 (har, err)
	require.Error(t, err)
}

// TestCovParseLenientCreatorBadType covers lines 167-170: creator field has the wrong type。
func TestCovParseLenientCreatorBadType(t *testing.T) {
	opts := DefaultParseOptions()
	opts.Lenient = true
	opts.CollectWarnings = true
	bad := []byte(`{"log":{"version":"1.2","creator":"notobj","entries":[{"request":{"method":"GET","url":"http://x"}}]}}`)
	h, err := ParseHarWithOptions(bad, opts)
	require.NotNil(t, h)
	assert.Equal(t, "1.2", h.Log.Version)
	require.Error(t, err)
}

// TestCovParseLenientPagesBadType covers lines 188-191: pages field is not an array。
func TestCovParseLenientPagesBadType(t *testing.T) {
	opts := DefaultParseOptions()
	opts.Lenient = true
	opts.CollectWarnings = true
	bad := []byte(`{"log":{"version":"1.2","pages":"notarray","entries":[{"request":{"method":"GET","url":"http://x"}}]}}`)
	h, err := ParseHarWithOptions(bad, opts)
	require.NotNil(t, h)
	require.Error(t, err)
}

// TestCovParseLenientPageItemBad covers lines 177-186: pages is an array but one page fails to parse。
func TestCovParseLenientPageItemBad(t *testing.T) {
	opts := DefaultParseOptions()
	opts.Lenient = true
	opts.CollectWarnings = true
	// The first page is a string (cannot parse as Pages); the second is valid.
	bad := []byte(`{"log":{"version":"1.2","pages":["notobj",{"id":"p1","title":"T","startedDateTime":"2024-01-01T00:00:00Z","pageTimings":{"onContentLoad":0,"onLoad":0}}],"entries":[{"request":{"method":"GET","url":"http://x"}}]}}`)
	h, err := ParseHarWithOptions(bad, opts)
	require.NotNil(t, h)
	require.Error(t, err)
	// The second page should be parsed successfully.
	require.Len(t, h.Log.Pages, 1)
}

// TestCovParseLenientEntriesBadType covers lines 209-212: entries field is not an array。
func TestCovParseLenientEntriesBadType(t *testing.T) {
	opts := DefaultParseOptions()
	opts.Lenient = true
	opts.CollectWarnings = true
	bad := []byte(`{"log":{"version":"1.2","entries":"notarray"}}`)
	h, err := ParseHarWithOptions(bad, opts)
	// version exists, so Har is returned.
	require.NotNil(t, h)
	assert.Equal(t, "1.2", h.Log.Version)
	require.Error(t, err)
}

// TestCovParseLenientEntryItemBad covers lines 202-207: entries is an array but one entry fails to parse。
func TestCovParseLenientEntryItemBad(t *testing.T) {
	opts := DefaultParseOptions()
	opts.Lenient = true
	opts.CollectWarnings = true
	// The first entry is a string; the second is valid.
	bad := []byte(`{"log":{"version":"1.2","entries":["badentry",{"startedDateTime":"2024-01-01T00:00:00Z","time":10,"request":{"method":"GET","url":"http://x"},"response":{"status":200}}]}}`)
	h, err := ParseHarWithOptions(bad, opts)
	require.NotNil(t, h)
	require.Error(t, err)
	require.Len(t, h.Log.Entries, 1)
}

// TestCovParseLenientPartialNoCollectWithContent covers lines 227-230: errors without warning collection, but with content。
// 此时返回 (nil, err) 因为 CollectWarnings=false。
func TestCovParseLenientPartialNoCollectWithContent(t *testing.T) {
	opts := DefaultParseOptions()
	opts.Lenient = true
	opts.CollectWarnings = false
	bad := []byte(`{"log":{"version":123,"entries":[{"request":{"method":"GET","url":"http://x"}}]}}`)
	h, err := ParseHarWithOptions(bad, opts)
	// CollectWarnings=false with an error -> return (nil, err).
	assert.Nil(t, h)
	require.Error(t, err)
}

// TestCovParseLenientFullFailureWithCollect covers lines 222-226: errors with warning collection but no parseable content.
func TestCovParseLenientFullFailureWithCollect(t *testing.T) {
	opts := DefaultParseOptions()
	opts.Lenient = true
	opts.CollectWarnings = true
	// log 存在但全是坏数据，无 version/entries/pages 成功
	bad := []byte(`{"log":{"version":123,"creator":"x","pages":"x","entries":"x"}}`)
	h, err := ParseHarWithOptions(bad, opts)
	assert.Nil(t, h)
	require.Error(t, err)
}

// TestCovParseLenientNoErrors covers lines 232: returns (har, nil) without errors.
func TestCovParseLenientNoErrors(t *testing.T) {
	opts := DefaultParseOptions()
	opts.Lenient = true
	opts.CollectWarnings = true
	h, err := ParseHarWithOptions(validHarJSON(), opts)
	require.NoError(t, err)
	assert.NotNil(t, h)
}

// TestCovParseLenientRootUnmarshalError covers lines 134-136: top-level JSON cannot be parsed as a map.
// isJSONContent 接受数组形式 [...]，但无法 unmarshal 成 map[string]json.RawMessage。
func TestCovParseLenientRootUnmarshalError(t *testing.T) {
	opts := DefaultParseOptions()
	opts.Lenient = true
	opts.CollectWarnings = true
	// 合法的 JSON 数组，isJSONContent 返回 true，但 unmarshal 到 map 失败
	h, err := ParseHarWithOptions([]byte(`[1,2,3]`), opts)
	assert.Nil(t, h)
	require.Error(t, err)
}

// --- ParseHarWithWarnings coverage ---

// TestCovParseHarWithWarningsFullFailure covers lines 259-266: complete parse failure (nil Har).
func TestCovParseHarWithWarningsFullFailure(t *testing.T) {
	// 空输入 -> ParseHarWithOptions 返回 (nil, err)，且 har==nil -> 完全失败分支
	result, err := ParseHarWithWarnings([]byte{})
	assert.Nil(t, result)
	require.Error(t, err)
}

// TestCovParseHarWithWarningsEmpty covers lines 259-266: non-empty input that cannot be parsed at all.
func TestCovParseHarWithWarningsEmpty(t *testing.T) {
	result, err := ParseHarWithWarnings(nil)
	assert.Nil(t, result)
	require.Error(t, err)
}

// TestCovParseHarWithWarningsValidNoWarnings covers lines 276-279: success with no warnings -> performFullValidation branch.
func TestCovParseHarWithWarningsValidNoWarnings(t *testing.T) {
	// 完全valid的 HAR，宽松模式解析无错误，validateURLs 也无警告
	// -> 进入 performFullValidation 分支
	result, err := ParseHarWithWarnings(validHarJSON())
	require.NoError(t, err)
	require.NotNil(t, result)
	// performFullValidation 会发现缺少 creator.name/version 等，产生警告
	// 但这里 HAR 是valid的，所以可能无警告
	_ = result.Warnings
}

// TestCovParseHarWithWarningsPartialWithHar covers lines 260-262: Har is returned with a parse error -> convert the error to a warning.
func TestCovParseHarWithWarningsPartialWithHar(t *testing.T) {
	// version 类型错误但有 entries -> (har, err) 且 har != nil
	bad := []byte(`{"log":{"version":123,"entries":[{"request":{"method":"GET","url":"http://x"}}]}}`)
	result, err := ParseHarWithWarnings(bad)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotNil(t, result.Har)
	// 应该有解析警告或 URL 验证相关
	// URL http://x 包含 :// 无空格，validateURLs 不会产生警告
	// 但解析错误会产生警告
	assert.NotEmpty(t, result.Warnings)
}

// TestCovParseHarWithWarningsURLValidation covers lines 270-273: validateURLs 产生警告。
func TestCovParseHarWithWarningsURLValidation(t *testing.T) {
	// The URL contains a space, triggering the validateURLs space branch.
	bad := []byte(`{"log":{"version":"1.2","entries":[{"request":{"method":"GET","url":"http://example.com/with space"}}]}}`)
	result, err := ParseHarWithWarnings(bad)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Warnings)
}

// --- validateURLs coverage ---

// TestCovValidateURLsNil covers lines 286-288: nil Har.
func TestCovValidateURLsNil(t *testing.T) {
	warnings := validateURLs(nil)
	assert.Nil(t, warnings)
}

// TestCovValidateURLsNoEntries covers lines 286-288: no entries.
func TestCovValidateURLsNoEntries(t *testing.T) {
	h := &Har{Log: Log{Entries: []Entries{}}}
	warnings := validateURLs(h)
	assert.Nil(t, warnings)
}

// TestCovValidateURLsEmptyURL covers lines 292-293: empty URL is skipped.
func TestCovValidateURLsEmptyURL(t *testing.T) {
	h := &Har{Log: Log{Entries: []Entries{
		{Request: Request{URL: ""}},
	}}}
	warnings := validateURLs(h)
	assert.Empty(t, warnings)
}

// TestCovValidateURLsNoScheme covers lines 315-316: URL is missing ://.
func TestCovValidateURLsNoScheme(t *testing.T) {
	h := &Har{Log: Log{Entries: []Entries{
		{Request: Request{URL: "example.com/path"}},
	}}}
	warnings := validateURLs(h)
	assert.NotEmpty(t, warnings)
	found := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "URL is missing a scheme") {
			found = true
		}
	}
	assert.True(t, found)
}

// TestCovValidateURLsSpace covers lines 307-308: URL contains spaces.
func TestCovValidateURLsSpace(t *testing.T) {
	h := &Har{Log: Log{Entries: []Entries{
		{Request: Request{URL: "http://example.com/with space"}},
	}}}
	warnings := validateURLs(h)
	assert.NotEmpty(t, warnings)
	found := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "URL contains spaces") {
			found = true
		}
	}
	assert.True(t, found)
}

// TestCovValidateURLsValid covers lines 297: valid URL (url.Parse has no error), contains ://, and has no spaces.
func TestCovValidateURLsValid(t *testing.T) {
	h := &Har{Log: Log{Entries: []Entries{
		{Request: Request{URL: "http://example.com/path"}},
	}}}
	warnings := validateURLs(h)
	assert.Empty(t, warnings)
}

// TestCovValidateURLsParseError covers lines 297-303: url.Parse returns an error.
// "http://[::1" 缺少 ] 会导致 url.Parse 报错，且 URL 包含 :// 但无空格，
// 因此进入 url.Parse 错误分支后 continue，不进入空格/协议检查。
func TestCovValidateURLsParseError(t *testing.T) {
	h := &Har{Log: Log{Entries: []Entries{
		{Request: Request{URL: "http://[::1"}},
	}}}
	warnings := validateURLs(h)
	assert.NotEmpty(t, warnings)
	found := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "Invalid URL format") {
			found = true
		}
	}
	assert.True(t, found)
}

// TestCovValidateURLsMultiple covers lines 291 multiple entries with both spaces and a missing scheme.
func TestCovValidateURLsMultiple(t *testing.T) {
	h := &Har{Log: Log{Entries: []Entries{
		{Request: Request{URL: "http://example.com/path"}}, // valid
		{Request: Request{URL: ""}},                        // empty; skipped
		{Request: Request{URL: "noscheme path"}},           // space + missing scheme
	}}}
	warnings := validateURLs(h)
	assert.NotEmpty(t, warnings)
	// Should have both space and missing-scheme warnings.
	assert.GreaterOrEqual(t, len(warnings), 2)
}

// --- performFullValidation coverage ---

// TestCovPerformFullValidationNil covers lines 329-331: nil Har.
func TestCovPerformFullValidationNil(t *testing.T) {
	warnings := performFullValidation(nil)
	assert.Nil(t, warnings)
}

// TestCovPerformFullValidationValid covers lines 333-336: valid HAR; validation succeeds.
func TestCovPerformFullValidationValid(t *testing.T) {
	h := &Har{Log: Log{Version: "1.2", Creator: Creator{Name: "x", Version: "1"}, Entries: []Entries{}}}
	warnings := performFullValidation(h)
	assert.Nil(t, warnings)
}

// TestCovPerformFullValidationInvalid covers lines 338-340: invalid HAR returns HarError -> GetPartialErrors.
func TestCovPerformFullValidationInvalid(t *testing.T) {
	// 缺少 version -> validateBasicStructure 会添加 partial error 并返回 rootError (*HarError)
	h := &Har{Log: Log{Creator: Creator{Name: "x", Version: "1"}, Entries: []Entries{}}}
	warnings := performFullValidation(h)
	assert.NotEmpty(t, warnings)
}

// TestCovPerformFullValidationNonHarError covers lines 343-345: non-HarError branch.
// 注意: ValidateHarFile 总是返回 *HarError 或 nil，因此此分支在实践中不可达。
// This test documents the branch but cannot actually trigger it.
func TestCovPerformFullValidationNonHarError(t *testing.T) {
	// 此分支不可达，ValidateHarFile 不会返回非 *HarError 类型。
	// 通过验证一个有部分错误的 HAR 确保走 HarError 分支。
	h := &Har{Log: Log{Creator: Creator{Name: "x", Version: "1"}, Entries: []Entries{}}}
	warnings := performFullValidation(h)
	assert.NotEmpty(t, warnings)
}

// --- appendWarnings coverage ---

// TestCovAppendWarningsEmptyNew covers lines 350-352: new warnings are empty.
func TestCovAppendWarningsEmptyNew(t *testing.T) {
	existing := []*HarError{NewValidationError("a", "f1")}
	result := appendWarnings(existing, nil)
	assert.Len(t, result, 1)
}

// TestCovAppendWarningsNilExisting covers lines 354-356: existing is nil.
func TestCovAppendWarningsNilExisting(t *testing.T) {
	newW := []*HarError{NewValidationError("a", "f1")}
	result := appendWarnings(nil, newW)
	assert.Len(t, result, 1)
}

// TestCovAppendWarningsDuplicate covers lines 360-363 + 366-372 deduplication branch.
func TestCovAppendWarningsDuplicate(t *testing.T) {
	w1 := NewValidationError("msg", "field")
	existing := []*HarError{w1}
	// Matching field+message pairs should be deduplicated.
	dup := NewValidationError("msg", "field")
	result := appendWarnings(existing, []*HarError{dup})
	assert.Len(t, result, 1)
}

// TestCovAppendWarningsNewAdded covers lines 366-372: append new warnings.
func TestCovAppendWarningsNewAdded(t *testing.T) {
	existing := []*HarError{NewValidationError("msg1", "f1")}
	newW := []*HarError{NewValidationError("msg2", "f2")}
	result := appendWarnings(existing, newW)
	assert.Len(t, result, 2)
}

// TestCovAppendWarningsMixed deduplication and append combined.
func TestCovAppendWarningsMixed(t *testing.T) {
	existing := []*HarError{
		NewValidationError("keep", "f1"),
		NewValidationError("dup", "f2"),
	}
	newW := []*HarError{
		NewValidationError("dup", "f2"), // duplicate
		NewValidationError("new", "f3"), // new
	}
	result := appendWarnings(existing, newW)
	assert.Len(t, result, 3)
}

// --- ParseHarFileWithWarnings coverage ---

// TestCovParseHarFileWithWarningsReadError covers lines 378-382: file read failure。
func TestCovParseHarFileWithWarningsReadError(t *testing.T) {
	result, err := ParseHarFileWithWarnings("/nonexistent/file.har")
	assert.Nil(t, result)
	require.Error(t, err)
	he, ok := err.(*HarError)
	assert.True(t, ok)
	assert.Equal(t, ErrCodeFileSystem, he.Code)
}

// TestCovParseHarFileWithWarningsSuccess covers lines 384: success path。
func TestCovParseHarFileWithWarningsSuccess(t *testing.T) {
	p := covWriteFile(t, "ok.har", validHarJSON())
	result, err := ParseHarFileWithWarnings(p)
	require.NoError(t, err)
	require.NotNil(t, result)
}

// --- Additional isJSONContent coverage ---

// TestCovIsJSONContentArray 覆盖array-form JSON.
func TestCovIsJSONContentArray(t *testing.T) {
	assert.True(t, isJSONContent([]byte("  [1,2,3]  ")))
}

// TestCovIsJSONContentObject 覆盖object-form JSON.
func TestCovIsJSONContentObject(t *testing.T) {
	assert.True(t, isJSONContent([]byte("  {\"a\":1}  ")))
}

// TestCovIsJSONContentInvalid 覆盖non-JSON.
func TestCovIsJSONContentInvalid(t *testing.T) {
	assert.False(t, isJSONContent([]byte("  hello  ")))
	assert.False(t, isJSONContent([]byte("[1,2}"))) // mismatched delimiters
}
