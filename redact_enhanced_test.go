package har

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Redaction of non-string JSON values (Task 19)---

func TestRedactJSONNonStringValues(t *testing.T) {
	// Sensitive JSON keys have numeric, boolean, null, or nested-object values; all should be replaced entirely.
	h := &Har{
		Log: Log{
			Entries: []Entries{
				{
					Request: Request{
						Method: "POST",
						PostData: &PostData{
							MimeType: "application/json",
							Text:     `{"secret_code": 12345, "is_admin": true, "hidden": null, "nested": {"token": "abc123"}}`,
						},
					},
				},
			},
		},
	}

	opts := RedactOptions{
		PostDataFields: []string{"secret_code", "is_admin", "hidden", "token"},
		Replacement:    "[REDACTED]",
		ValuePatterns:  nil, // Disable default value patterns; test name matching only.
	}
	result := h.Redact(opts)
	text := result.Log.Entries[0].Request.PostData.Text

	// Numeric, boolean, and null values are replaced entirely with the string "[REDACTED]".
	assert.Contains(t, text, `"secret_code": "[REDACTED]"`)
	assert.Contains(t, text, `"is_admin": "[REDACTED]"`)
	assert.Contains(t, text, `"hidden": "[REDACTED]"`)
	// Also redact token values in nested objects.
	assert.Contains(t, text, `"token": "[REDACTED]"`)
}

func TestRedactJSONArrayValues(t *testing.T) {
	// Sensitive keys have array values.
	h := &Har{
		Log: Log{
			Entries: []Entries{
				{
					Request: Request{
						PostData: &PostData{
							Text: `{"tokens": ["abc", "def"], "ids": [1, 2, 3]}`,
						},
					},
				},
			},
		},
	}

	opts := RedactOptions{
		PostDataFields: []string{"tokens", "ids"},
		Replacement:    "***",
		ValuePatterns:  nil,
	}
	result := h.Redact(opts)
	text := result.Log.Entries[0].Request.PostData.Text

	assert.Contains(t, text, `"tokens": "***"`)
	assert.Contains(t, text, `"ids": "***"`)
}

// --- Value-pattern redaction (match by content, not name).---

func TestRedactValuePatternsBearerToken(t *testing.T) {
	// Bearer token 藏在自定义 header 值里
	h := &Har{
		Log: Log{
			Entries: []Entries{
				{
					Request: Request{
						Headers: []Headers{
							{Name: "X-Custom-Auth", Value: "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.abc.def"},
						},
					},
				},
			},
		},
	}

	opts := RedactOptions{
		Headers:       []string{}, // Do not match by name.
		Replacement:   "[REDACTED]",
		ValuePatterns: DefaultRedactValuePatterns(),
	}
	result := h.Redact(opts)
	headerVal := result.Log.Entries[0].Request.Headers[0].Value

	assert.Equal(t, "[REDACTED]", headerVal)
}

func TestRedactValuePatternsInCookie(t *testing.T) {
	// A JWT is hidden in a cookie value.
	h := &Har{
		Log: Log{
			Entries: []Entries{
				{
					Request: Request{
						Cookies: []Cookie{
							{Name: "session", Value: "eyJhbGciOiJIUzI1NiJ9.payload.sig"},
						},
					},
				},
			},
		},
	}

	opts := RedactOptions{
		Cookies:       []string{}, // Do not match by name.
		Replacement:   "***",
		ValuePatterns: DefaultRedactValuePatterns(),
	}
	result := h.Redact(opts)
	cookieVal := result.Log.Entries[0].Request.Cookies[0].Value

	assert.Equal(t, "***", cookieVal)
}

func TestRedactValuePatternsInQueryParam(t *testing.T) {
	// The query parameter value is an AWS access key.
	h := &Har{
		Log: Log{
			Entries: []Entries{
				{
					Request: Request{
						URL: "https://api.example.com/data?key=AKIAIOSFODNN7EXAMPLE",
					},
				},
			},
		},
	}

	opts := RedactOptions{
		QueryParams:   []string{}, // Do not match by name.
		Replacement:   "[SCRUBBED]",
		ValuePatterns: DefaultRedactValuePatterns(),
	}
	result := h.Redact(opts)

	assert.Contains(t, result.Log.Entries[0].Request.URL, "key=[SCRUBBED]")
}

func TestRedactValuePatternsInJSONBody(t *testing.T) {
	// The JSON body contains a GitHub PAT in a field whose name is not in PostDataFields.
	h := &Har{
		Log: Log{
			Entries: []Entries{
				{
					Request: Request{
						PostData: &PostData{
							Text: `{"note": "use this token: ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx for testing"}`,
						},
					},
				},
			},
		},
	}

	opts := RedactOptions{
		PostDataFields: []string{}, // Do not match by name.
		Replacement:    "[REDACTED]",
		ValuePatterns:  DefaultRedactValuePatterns(),
	}
	result := h.Redact(opts)
	text := result.Log.Entries[0].Request.PostData.Text

	assert.Contains(t, text, "[REDACTED]")
	assert.NotContains(t, text, "ghp_")
}

func TestRedactValuePatternsNoMatch(t *testing.T) {
	// A normal value does not trigger a value pattern.
	h := &Har{
		Log: Log{
			Entries: []Entries{
				{
					Request: Request{
						Headers: []Headers{
							{Name: "X-Request-Id", Value: "abc-123-def"},
						},
					},
				},
			},
		},
	}

	opts := RedactOptions{
		Headers:       []string{},
		Replacement:   "[REDACTED]",
		ValuePatterns: DefaultRedactValuePatterns(),
	}
	result := h.Redact(opts)

	// A normal request ID is not redacted.
	assert.Equal(t, "abc-123-def", result.Log.Entries[0].Request.Headers[0].Value)
}

func TestRedactValuePatternsCustomReplacement(t *testing.T) {
	// A single pattern has a custom replacement.
	h := &Har{
		Log: Log{
			Entries: []Entries{
				{
					Request: Request{
						Headers: []Headers{
							{Name: "X-Token", Value: "Bearer secret-token-here"},
						},
					},
				},
			},
		},
	}

	opts := RedactOptions{
		Replacement: "[GLOBAL]",
		ValuePatterns: []RedactValuePattern{
			{Name: "bearer", Pattern: `(?i)\bBearer\s+\S+`, Replacement: "[TOKEN_SCRUBBED]"},
		},
	}
	result := h.Redact(opts)

	assert.Equal(t, "[TOKEN_SCRUBBED]", result.Log.Entries[0].Request.Headers[0].Value)
}

func TestRedactValuePatternsDisabled(t *testing.T) {
	// Explicitly disable value patterns (ValuePatterns=nil or empty).
	h := &Har{
		Log: Log{
			Entries: []Entries{
				{
					Request: Request{
						Headers: []Headers{
							{Name: "X-Auth", Value: "Bearer secret"},
						},
					},
				},
			},
		},
	}

	opts := RedactOptions{
		Headers:       []string{}, // Do not match by name.
		Replacement:   "[REDACTED]",
		ValuePatterns: nil, // Disable.
	}
	result := h.Redact(opts)

	// When value patterns are disabled, do not redact (the name does not match either).
	assert.Equal(t, "Bearer secret", result.Log.Entries[0].Request.Headers[0].Value)
}

// --- Edge case: preserve compact JSON body formatting without spaces. ---

func TestRedactJSONBodyCompactFormat(t *testing.T) {
	// If the input has no spaces, the output should have none either.
	h := &Har{
		Log: Log{
			Entries: []Entries{
				{
					Request: Request{
						PostData: &PostData{
							Text: `{"password":"hunter2","username":"bob"}`,
						},
					},
				},
			},
		},
	}

	opts := RedactOptions{
		PostDataFields: []string{"password"},
		Replacement:    "[REDACTED]",
		ValuePatterns:  nil,
	}
	result := h.Redact(opts)
	text := result.Log.Entries[0].Request.PostData.Text

	assert.Contains(t, text, `"password":"[REDACTED]"`)
	assert.NotContains(t, text, `"password": "[REDACTED]"`) // Should not contain spaces.
}

func TestRedactJSONBodyPrettifiedFormat(t *testing.T) {
	// Multiline indented input should remain multiline in the output.
	h := &Har{
		Log: Log{
			Entries: []Entries{
				{
					Request: Request{
						PostData: &PostData{
							Text: `{
  "password": "hunter2",
  "username": "bob"
}`,
						},
					},
				},
			},
		},
	}

	opts := RedactOptions{
		PostDataFields: []string{"password"},
		Replacement:    "***",
		ValuePatterns:  nil,
	}
	result := h.Redact(opts)
	text := result.Log.Entries[0].Request.PostData.Text

	require.Contains(t, text, `"password": "***"`)
	require.Contains(t, text, "\n") // Preserve multiple lines.
}

// --- Edge case: skip an invalid value-pattern regex without interrupting redaction. ---

func TestRedactInvalidValuePatternSkipped(t *testing.T) {
	h := &Har{
		Log: Log{
			Entries: []Entries{
				{
					Request: Request{
						Headers: []Headers{
							{Name: "X-Token", Value: "Bearer secret"},
						},
					},
				},
			},
		},
	}

	opts := RedactOptions{
		Replacement: "[REDACTED]",
		ValuePatterns: []RedactValuePattern{
			{Name: "bad", Pattern: `[`}, // Invalid regex.
			{Name: "bearer", Pattern: `(?i)Bearer\s+\S+`},
		},
	}
	result := h.Redact(opts)

	// Skip the invalid pattern and apply the valid pattern.
	assert.Equal(t, "[REDACTED]", result.Log.Entries[0].Request.Headers[0].Value)
}

// --- Edge case: name matching takes precedence over value patterns. ---

func TestRedactNameMatchOverridesValuePattern(t *testing.T) {
	// A name match replaces the entire value; value patterns are not run afterward (the value is already replaced).
	h := &Har{
		Log: Log{
			Entries: []Entries{
				{
					Request: Request{
						Headers: []Headers{
							{Name: "Authorization", Value: "Bearer secret"},
						},
					},
				},
			},
		},
	}

	opts := RedactOptions{
		Headers:       []string{"Authorization"},
		Replacement:   "***",
		ValuePatterns: DefaultRedactValuePatterns(),
	}
	result := h.Redact(opts)

	// Name match triggered; CustomRedactor is unset, so replace the entire value with ***.
	assert.Equal(t, "***", result.Log.Entries[0].Request.Headers[0].Value)
}

// --- Coverage: CustomRedactor handles non-string JSON values. ---

func TestRedactCustomRedactorNonStringValue(t *testing.T) {
	// When CustomRedactor receives a non-string value (number/boolean/null), the fmtJSON branch is used.
	h := &Har{
		Log: Log{
			Entries: []Entries{
				{
					Request: Request{
						PostData: &PostData{
							Text: `{"secret_code": 12345, "is_admin": true, "hidden": null}`,
						},
					},
				},
			},
		},
	}

	opts := RedactOptions{
		PostDataFields: []string{"secret_code", "is_admin", "hidden"},
		ValuePatterns:  nil,
		CustomRedactor: func(fieldType, name, value string) string {
			return "[C:" + name + ":" + value + "]"
		},
	}
	result := h.Redact(opts)
	text := result.Log.Entries[0].Request.PostData.Text

	// Non-string values are converted to strings by fmtJSON before being passed to CustomRedactor.
	assert.Contains(t, text, `"secret_code": "[C:secret_code:12345]"`)
	assert.Contains(t, text, `"is_admin": "[C:is_admin:true]"`)
	assert.Contains(t, text, `"hidden": "[C:hidden:null]"`)
}

// --- Coverage: fall back to regex when JSON body parsing fails. ---

func TestRedactJSONBodyFallbackToRegex(t *testing.T) {
	// Text contains = but is not valid JSON, so redactKeyValuePairs is used; token is redacted as a key.
	h := &Har{
		Log: Log{
			Entries: []Entries{
				{
					Request: Request{
						PostData: &PostData{
							Text: `token=secret_value&keep=1`,
						},
					},
				},
			},
		},
	}

	opts := RedactOptions{
		PostDataFields: []string{"token"},
		Replacement:    "[R]",
		ValuePatterns:  nil,
	}
	result := h.Redact(opts)
	text := result.Log.Entries[0].Request.PostData.Text
	// The token field is redacted.
	assert.Contains(t, text, "token=[R]")
	assert.Contains(t, text, "keep=1")
}

// --- Coverage: detect multiline JSON indentation. ---

func TestRedactJSONBodyFourSpaceIndent(t *testing.T) {
	// Multiline JSON with four-space indentation.
	h := &Har{
		Log: Log{
			Entries: []Entries{
				{
					Request: Request{
						PostData: &PostData{
							Text: "{\n    \"password\": \"hunter2\"\n}",
						},
					},
				},
			},
		},
	}

	opts := RedactOptions{
		PostDataFields: []string{"password"},
		Replacement:    "***",
		ValuePatterns:  nil,
	}
	result := h.Redact(opts)
	text := result.Log.Entries[0].Request.PostData.Text

	assert.Contains(t, text, `"password": "***"`)
	assert.Contains(t, text, "\n") // Preserve multiple lines.
}

// --- Coverage: valueRes branch in redactQueryStringSimple. ---

func TestRedactQueryStringSimpleValuePatternFallback(t *testing.T) {
	// URLs accepted by url.Parse do not use the simple fallback; call the simple function directly here.
	out := redactQueryStringSimple(
		"https://example.com/?trace=Bearer%20secret",
		[]string{}, // Do not match by name.
		"[X]",
		RedactOptions{},
		compileValuePatterns(DefaultRedactValuePatterns()),
	)
	// The value pattern matches and replaces the Bearer token.
	assert.Contains(t, out, "trace=")
}

// --- Coverage: value-pattern branch in redactKeyValuePairs. ---

func TestRedactKeyValuePairsValuePattern(t *testing.T) {
	// 表单里某字段值藏 Bearer token（明文，非 URL 编码），但字段名不在 PostDataFields
	text := `note=Bearer eyJhbGciOiJIUzI1NiJ9.abc.def&keep=1`
	opts := RedactOptions{
		PostDataFields: []string{},
		Replacement:    "[R]",
		ValuePatterns:  DefaultRedactValuePatterns(),
	}
	result := redactKeyValuePairs(text, opts, "[R]", compileValuePatterns(opts.ValuePatterns))
	// The note value is redacted by the value pattern.
	assert.Contains(t, result, "note=")
	assert.NotContains(t, result, "Bearer eyJ")
	assert.Contains(t, result, "keep=1")
}

// --- Coverage: apply value patterns to string values in objects inside JSON arrays. ---

func TestRedactJSONValuePatternInArray(t *testing.T) {
	// Apply value patterns to each string value in each object in the JSON array.
	h := &Har{
		Log: Log{
			Entries: []Entries{
				{
					Request: Request{
						PostData: &PostData{
							Text: `{"items":[{"note":"Bearer token1"},{"note":"Bearer token2"}]}`,
						},
					},
				},
			},
		},
	}

	opts := RedactOptions{
		PostDataFields: []string{}, // Do not match by name.
		Replacement:    "[R]",
		ValuePatterns:  DefaultRedactValuePatterns(),
	}
	result := h.Redact(opts)
	text := result.Log.Entries[0].Request.PostData.Text

	assert.NotContains(t, text, "Bearer token1")
	assert.NotContains(t, text, "Bearer token2")
	assert.Contains(t, text, "[R]")
}

// --- Edge case: CustomRedactor and value patterns are used together. ---

func TestRedactCustomRedactorWithValuePatterns(t *testing.T) {
	// CustomRedactor handles name matches; value patterns still apply to strings whose names do not match.
	h := &Har{
		Log: Log{
			Entries: []Entries{
				{
					Request: Request{
						Headers: []Headers{
							{Name: "Authorization", Value: "Bearer named-match"},
							{Name: "X-Custom", Value: "Bearer value-pattern-match"},
						},
					},
				},
			},
		},
	}

	opts := RedactOptions{
		Headers:       []string{"Authorization"},
		Replacement:   "[REDACTED]",
		ValuePatterns: DefaultRedactValuePatterns(),
		CustomRedactor: func(fieldType, name, value string) string {
			return "[CUSTOM:" + name + "]"
		},
	}
	result := h.Redact(opts)

	// Name matches use CustomRedactor.
	assert.Equal(t, "[CUSTOM:Authorization]", result.Log.Entries[0].Request.Headers[0].Value)
	// Name does not match but a value pattern does, so use the value pattern (with global replacement).
	assert.Equal(t, "[REDACTED]", result.Log.Entries[0].Request.Headers[1].Value)
}

// --- Coverage: detectJSONIndent falls back to two spaces when indentation is absent. ---

func TestRedactJSONBodyMultilineNoIndent(t *testing.T) {
	// Multiline JSON with no leading spaces on any line; detectJSONIndent falls back to the default "  ".
	// (json.Unmarshal accepts this unconventional but valid multiline compact JSON.)
	h := &Har{
		Log: Log{
			Entries: []Entries{
				{
					Request: Request{
						PostData: &PostData{
							Text: "{\n\"password\":\"x\"\n}",
						},
					},
				},
			},
		},
	}

	opts := RedactOptions{
		PostDataFields: []string{"password"},
		Replacement:    "[R]",
		ValuePatterns:  nil,
	}
	result := h.Redact(opts)
	text := result.Log.Entries[0].Request.PostData.Text
	// Use MarshalIndent (with newlines) and redact password.
	assert.Contains(t, text, "[R]")
	assert.Contains(t, text, "\n")
}

// --- Coverage: value-pattern branch for non-JSON redactPostDataText input without =. ---

func TestRedactPostDataTextPlainValuePattern(t *testing.T) {
	// Input is neither JSON nor does it contain =, but a value pattern matches; use the final value-pattern branch.
	h := &Har{
		Log: Log{
			Entries: []Entries{
				{
					Request: Request{
						PostData: &PostData{
							Text: `raw text with Bearer eyJhbGciOiJIUzI1NiJ9.abc.def inside`,
						},
					},
				},
			},
		},
	}

	opts := RedactOptions{
		PostDataFields: []string{},
		Replacement:    "[R]",
		ValuePatterns:  DefaultRedactValuePatterns(),
	}
	result := h.Redact(opts)
	text := result.Log.Entries[0].Request.PostData.Text
	assert.NotContains(t, text, "Bearer eyJ")
	assert.Contains(t, text, "[R]")
}

// --- Coverage: empty text and unrecognized format branches in redactPostDataText. ---

func TestRedactPostDataTextEmpty(t *testing.T) {
	// Whitespace-only text returns immediately.
	h := &Har{
		Log: Log{
			Entries: []Entries{
				{
					Request: Request{
						PostData: &PostData{Text: "   "},
					},
				},
			},
		},
	}
	opts := RedactOptions{Replacement: "[R]", ValuePatterns: DefaultRedactValuePatterns()}
	result := h.Redact(opts)
	assert.Equal(t, "   ", result.Log.Entries[0].Request.PostData.Text)
}

func TestRedactPostDataTextUnrecognizedNoValuePattern(t *testing.T) {
	// Non-JSON, no =, and no value-pattern match: return unchanged.
	h := &Har{
		Log: Log{
			Entries: []Entries{
				{
					Request: Request{
						PostData: &PostData{Text: "just plain text no equals"},
					},
				},
			},
		},
	}
	opts := RedactOptions{Replacement: "[R]", ValuePatterns: nil}
	result := h.Redact(opts)
	assert.Equal(t, "just plain text no equals", result.Log.Entries[0].Request.PostData.Text)
}

// --- Coverage: empty-value query parameters in redactQueryStringSimple. ---

func TestRedactQueryStringSimpleEmptyValue(t *testing.T) {
	// When the parameter value is empty, the value-pattern branch should return match unchanged.
	out := redactQueryStringSimple(
		"https://example.com/?key=&keep=1",
		[]string{},
		"[R]",
		RedactOptions{},
		compileValuePatterns(DefaultRedactValuePatterns()),
	)
	// key= has an empty value and is not replaced.
	assert.Contains(t, out, "key=&")
	assert.Contains(t, out, "keep=1")
}
