package har

import (
	"encoding/json"
	"net"
	"net/url"
	"regexp"
	"strings"
)

// RedactURLRule defines a rule for redacting URL path segments.
type RedactURLRule struct {
	Pattern     string // regex pattern to match URL path segments
	Replacement string // replacement for matched segments
}

// RedactValuePattern defines a rule for redacting by *value content* rather than
// by field name. The regex is matched against the full string value of any
// string field (header/cookie/query/post-data/JSON string value); a match
// triggers redaction regardless of which key it lives under.
//
// This catches secrets that hide in fields whose names are not on the
// name-based redaction list — e.g. a custom header "X-Trace-Id" whose value
// happens to be a Bearer token, or a JSON field "notes" containing a pasted
// API key.
type RedactValuePattern struct {
	// Name is a human label for the pattern, used in logs/debugging. Optional.
	Name string
	// Pattern is a regex matched (unanchored) against string field values.
	Pattern string
	// Replacement overrides the global Replacement for this pattern. If empty,
	// the global Replacement is used.
	Replacement string
}

// RedactOptions configures how sensitive data is redacted from a HAR file.
type RedactOptions struct {
	Headers        []string                                                 // header names to redact (case-insensitive)
	Cookies        []string                                                 // cookie names to redact (case-insensitive)
	QueryParams    []string                                                 // query parameter names to redact (case-insensitive)
	PostDataFields []string                                                 // POST form field names to redact (case-insensitive)
	Replacement    string                                                   // replacement text (default: "[REDACTED]")
	RedactIPs      bool                                                     // whether to anonymize IP addresses
	RedactURLs     []RedactURLRule                                          // URL path segment redaction rules
	CustomRedactor func(fieldType string, name string, value string) string // custom redaction function
	// ValuePatterns redacts by *value content* regardless of field name.
	// Applied to every string field after name-based redaction. A value that
	// matches any pattern is replaced. See RedactValuePattern for details.
	ValuePatterns []RedactValuePattern
}

// DefaultRedactOptions returns a RedactOptions with sensible defaults for
// common sensitive fields found in HTTP traffic.
func DefaultRedactOptions() RedactOptions {
	return RedactOptions{
		Headers: []string{
			"Authorization",
			"Proxy-Authorization",
			"WWW-Authenticate",
			"Cookie",
			"Set-Cookie",
			"X-Api-Key",
			"X-Auth-Token",
			"X-CSRF-Token",
		},
		Cookies: []string{
			"session",
			"token",
			"auth",
			"password",
			"secret",
			"api_key",
			"access_token",
			"refresh_token",
		},
		QueryParams: []string{
			"password",
			"token",
			"api_key",
			"secret",
			"access_token",
			"refresh_token",
			"private_key",
			"client_secret",
		},
		PostDataFields: []string{
			"password",
			"token",
			"api_key",
			"secret",
			"access_token",
			"refresh_token",
			"private_key",
			"client_secret",
		},
		Replacement: "[REDACTED]",
		RedactIPs:   false,
		// By default, redact values matching common secret formats in any field,
		// without relying on field names (including custom headers and arbitrary JSON keys).
		ValuePatterns: DefaultRedactValuePatterns(),
	}
}

// DefaultRedactValuePatterns returns a set of common secret-shaped patterns
// for use as RedactOptions.ValuePatterns. Each matches a recognizable secret
// format (Bearer/JWT tokens, long hex/base64 secrets, AWS-style keys, etc.)
// so that secrets hiding under arbitrary field names still get redacted.
//
// Patterns are intentionally conservative: minimum lengths are set high
// enough to avoid flagging ordinary short strings.
func DefaultRedactValuePatterns() []RedactValuePattern {
	return []RedactValuePattern{
		{Name: "bearer-token", Pattern: `(?i)\bBearer\s+[A-Za-z0-9\-\._~+/]+=*`},
		{Name: "jwt", Pattern: `\beyJ[A-Za-z0-9_\-=]+\.[A-Za-z0-9_\-=]+\.?[A-Za-z0-9_\-=]*`},
		{Name: "aws-access-key", Pattern: `\bAKIA[0-9A-Z]{16}\b`},
		{Name: "aws-secret", Pattern: `\b[A-Za-z0-9/+=]{40}\b`},
		{Name: "github-pat", Pattern: `\bgh[pousr]_[A-Za-z0-9]{36,}\b`},
		{Name: "slack-token", Pattern: `\bxox[bp]-[A-Za-z0-9-]+\b`},
		{Name: "google-api-key", Pattern: `\bAIza[0-9A-Za-z\-_]{35}\b`},
		{Name: "hex-secret", Pattern: `\b[0-9a-fA-F]{64}\b`},
	}
}

// Redact returns a new Har with sensitive data redacted.
// It deep-clones the Har first, then redacts the clone so the original is unchanged.
func (h *Har) Redact(opts RedactOptions) *Har {
	if h == nil {
		return nil
	}

	clone := h.Clone()
	if clone == nil {
		return nil
	}
	clone.RedactInPlace(opts)
	return clone
}

// RedactInPlace mutates the Har in place, redacting sensitive data without cloning.
func (h *Har) RedactInPlace(opts RedactOptions) {
	if h == nil {
		return
	}

	replacement := opts.Replacement
	if replacement == "" {
		replacement = "[REDACTED]"
	}

	// Precompile value-pattern regexes to avoid recompiling them for each field.
	valueRes := compileValuePatterns(opts.ValuePatterns)

	for i := range h.Log.Entries {
		entry := &h.Log.Entries[i]

		// Redact request headers
		for j := range entry.Request.Headers {
			header := &entry.Request.Headers[j]
			if matchesAny(header.Name, opts.Headers) {
				header.Value = redactHeaderValue(header.Name, header.Value, opts, replacement)
			} else if header.Value != "" {
				header.Value = applyValuePatterns(header.Value, valueRes, replacement)
			}
		}

		// Redact response headers
		for j := range entry.Response.Headers {
			header := &entry.Response.Headers[j]
			if matchesAny(header.Name, opts.Headers) {
				header.Value = redactHeaderValue(header.Name, header.Value, opts, replacement)
			} else if header.Value != "" {
				header.Value = applyValuePatterns(header.Value, valueRes, replacement)
			}
		}

		// Redact request cookies
		for j := range entry.Request.Cookies {
			cookie := &entry.Request.Cookies[j]
			if matchesAny(cookie.Name, opts.Cookies) {
				cookie.Value = redactCookieValue(cookie.Name, cookie.Value, opts, replacement)
			} else if cookie.Value != "" {
				cookie.Value = applyValuePatterns(cookie.Value, valueRes, replacement)
			}
		}

		// Redact response cookies
		for j := range entry.Response.Cookies {
			cookie := &entry.Response.Cookies[j]
			if matchesAny(cookie.Name, opts.Cookies) {
				cookie.Value = redactCookieValue(cookie.Name, cookie.Value, opts, replacement)
			} else if cookie.Value != "" {
				cookie.Value = applyValuePatterns(cookie.Value, valueRes, replacement)
			}
		}

		// Redact query string parameters
		for j := range entry.Request.QueryString {
			qs := &entry.Request.QueryString[j]
			if matchesAny(qs.Name, opts.QueryParams) {
				qs.Value = redactQueryParamValue(qs.Name, qs.Value, opts, replacement)
			} else if qs.Value != "" {
				qs.Value = applyValuePatterns(qs.Value, valueRes, replacement)
			}
		}

		// Redact URL (query params in URL string and path segment rules)
		if entry.Request.URL != "" {
			entry.Request.URL = redactURLString(entry.Request.URL, opts, replacement, valueRes)
		}

		// Redact POST data
		if entry.Request.PostData != nil {
			pd := entry.Request.PostData
			// Redact POST params
			for j := range pd.Params {
				param := &pd.Params[j]
				if matchesAny(param.Name, opts.PostDataFields) {
					param.Value = redactPostDataFieldValue(param.Name, param.Value, opts, replacement)
				} else if param.Value != "" {
					param.Value = applyValuePatterns(param.Value, valueRes, replacement)
				}
			}
			// Redact POST text bodies (form key=value or JSON).
			if pd.Text != "" {
				pd.Text = redactPostDataText(pd.Text, opts, replacement, valueRes)
			}
		}

		// Anonymize IP addresses
		if opts.RedactIPs && entry.ServerIPAddress != "" {
			entry.ServerIPAddress = anonymizeIP(entry.ServerIPAddress)
		}
	}
}

// matchesAny checks whether name matches any of the patterns (case-insensitive).
func matchesAny(name string, patterns []string) bool {
	nameLower := strings.ToLower(name)
	for _, p := range patterns {
		if strings.ToLower(p) == nameLower {
			return true
		}
	}
	return false
}

// compiledValuePattern is a precompiled RedactValuePattern regex.
type compiledValuePattern struct {
	re          *regexp.Regexp
	replacement string // per-pattern override, empty means use global
}

// compileValuePatterns precompiles value-pattern regexes to avoid recompiling them for each field.
// If a Pattern is invalid, skip it without interrupting redaction.
func compileValuePatterns(patterns []RedactValuePattern) []compiledValuePattern {
	if len(patterns) == 0 {
		return nil
	}
	out := make([]compiledValuePattern, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p.Pattern)
		if err != nil {
			continue // Skip the invalid regex.
		}
		out = append(out, compiledValuePattern{re: re, replacement: p.Replacement})
	}
	return out
}

// applyValuePatterns applies all value-pattern regexes to a string.
// The first matching pattern is applied (subsequent patterns are skipped). Returns the replaced string;
// returns the original value if nothing matches.
func applyValuePatterns(value string, patterns []compiledValuePattern, globalReplacement string) string {
	for _, p := range patterns {
		if p.re.MatchString(value) {
			repl := p.replacement
			if repl == "" {
				repl = globalReplacement
			}
			return p.re.ReplaceAllString(value, repl)
		}
	}
	return value
}

// redactHeaderValue redacts a header value.
func redactHeaderValue(name string, value string, opts RedactOptions, replacement string) string {
	if opts.CustomRedactor != nil {
		return opts.CustomRedactor("header", name, value)
	}
	return replacement
}

// redactCookieValue redacts a cookie value.
func redactCookieValue(name string, value string, opts RedactOptions, replacement string) string {
	if opts.CustomRedactor != nil {
		return opts.CustomRedactor("cookie", name, value)
	}
	return replacement
}

// redactQueryParamValue redacts a query parameter value.
func redactQueryParamValue(name string, value string, opts RedactOptions, replacement string) string {
	if opts.CustomRedactor != nil {
		return opts.CustomRedactor("queryparam", name, value)
	}
	return replacement
}

// redactPostDataFieldValue redacts a POST form field value.
func redactPostDataFieldValue(name string, value string, opts RedactOptions, replacement string) string {
	if opts.CustomRedactor != nil {
		return opts.CustomRedactor("postdatafield", name, value)
	}
	return replacement
}

// redactPostDataText redacts sensitive data in POST body text.
// It auto-detects JSON vs URL-encoded form bodies and dispatches to the
// appropriate redactor. JSON bodies are parsed and walked recursively so that
// non-string values (numbers, booleans, null, nested objects/arrays) are
// handled correctly — a sensitive key like "secret": 12345 is redacted even
// though its value is not a string.
func redactPostDataText(text string, opts RedactOptions, replacement string, valueRes []compiledValuePattern) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return text
	}

	// JSON body: only attempt JSON parsing when it starts with { or [ (to avoid treating form data as JSON).
	if (trimmed[0] == '{' || trimmed[0] == '[') && looksLikeJSON(trimmed) {
		if out, ok := redactJSONBody(text, opts, replacement, valueRes); ok {
			return out
		}
		// If JSON parsing fails, fall back to regex-based redaction.
	}

	// URL-encoded form bodies (key=value&key=value)
	if strings.Contains(text, "=") {
		return redactKeyValuePairs(text, opts, replacement, valueRes)
	}

	// For unrecognized formats, still apply value patterns to catch plaintext secrets.
	if len(valueRes) > 0 {
		return applyValuePatterns(text, valueRes, replacement)
	}
	return text
}

// looksLikeJSON performs a rough check for valid JSON by attempting to unmarshal into interface{}.
func looksLikeJSON(s string) bool {
	var v interface{}
	return json.Unmarshal([]byte(s), &v) == nil
}

// redactJSONBody parses JSON text and recursively visits all key-value pairs:
// - Replace the entire value of fields matching PostDataFields, regardless of type;
// - Also apply value-pattern matching to strings (to catch secrets under arbitrary keys);
// - Preserve the original structure (objects, arrays, nesting) and indentation style.
//
// looksLikeJSON ensures text is valid JSON before this call, so Unmarshal must succeed;
// data is a decoded JSON value, so Marshal/MarshalIndent must succeed.
// The second return value indicates whether parsing and rewriting succeeded (if false, callers should fall back to regex-based redaction).
func redactJSONBody(text string, opts RedactOptions, replacement string, valueRes []compiledValuePattern) (string, bool) {
	var data interface{}
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		// Filtered by looksLikeJSON and theoretically unreachable; retained as a safeguard.
		return text, false
	}
	data = redactJSONValue(data, "", opts, replacement, valueRes)

	// Preserve indentation: rebuild multiline JSON (containing newlines) with MarshalIndent.
	if strings.Contains(text, "\n") {
		indent := detectJSONIndent(text)
		out, _ := json.MarshalIndent(data, "", indent)
		return string(out), true
	}

	// Single-line JSON: compact it, then restore the original spacing style around ":" and "," when present.
	out, _ := json.Marshal(data)
	if strings.Contains(text, ": ") || strings.Contains(text, ", ") {
		return prettifySingleLineJSON(string(out)), true
	}
	return string(out), true
}

// detectJSONIndent detects the indentation unit in multiline JSON (defaults to two spaces).
func detectJSONIndent(text string) string {
	for _, line := range strings.Split(text, "\n") {
		// Skip the first line (no indentation).
		trimmed := strings.TrimLeft(line, " ")
		if trimmed == line || trimmed == "" {
			continue
		}
		return line[:len(line)-len(trimmed)]
	}
	return "  "
}

// prettifySingleLineJSON formats compact JSON {"k":"v","a":1} as {"k": "v", "a": 1}.
// Only applies to simple single-line input without nested structures; nested structures remain compact (functionally correct, but formatted differently).
func prettifySingleLineJSON(s string) string {
	s = strings.ReplaceAll(s, `":`, `": `)
	s = strings.ReplaceAll(s, `","`, `", "`)
	return s
}

// redactJSONValue recursively processes decoded JSON values (map/slice/string/number/bool/nil).
// parentKey is used as context for CustomRedactor.
func redactJSONValue(data interface{}, parentKey string, opts RedactOptions, replacement string, valueRes []compiledValuePattern) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, val := range v {
			if matchesAny(key, opts.PostDataFields) {
				// Name matches: replace the entire value, regardless of type.
				if opts.CustomRedactor != nil {
					if s, ok := val.(string); ok {
						v[key] = opts.CustomRedactor("postdatafield", key, s)
					} else {
						v[key] = opts.CustomRedactor("postdatafield", key, fmtJSON(val))
					}
				} else {
					v[key] = replacement
				}
				continue
			}
			v[key] = redactJSONValue(val, key, opts, replacement, valueRes)
		}
		return v
	case []interface{}:
		for i := range v {
			v[i] = redactJSONValue(v[i], parentKey, opts, replacement, valueRes)
		}
		return v
	case string:
		// Apply value-pattern matching to string values.
		if v != "" && len(valueRes) > 0 {
			return applyValuePatterns(v, valueRes, replacement)
		}
		return v
	default:
		// Value patterns do not apply to number/bool/nil values.
		return v
	}
}

// fmtJSON converts any value to a compact JSON string (for CustomRedactor to receive non-string values).
// v is a decoded JSON value (number/bool/nil), so Marshal must succeed.
func fmtJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// redactKeyValuePairs redacts sensitive fields in URL-encoded body text.
// Handles both key=value& and key=value at end of string.
func redactKeyValuePairs(text string, opts RedactOptions, replacement string, valueRes []compiledValuePattern) string {
	// Match key=value patterns (URL-encoded or plain)
	re := regexp.MustCompile(`([^&=]+)=([^&]*)`)
	result := re.ReplaceAllStringFunc(text, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) >= 3 {
			key := parts[1]
			value := parts[2]
			if matchesAny(key, opts.PostDataFields) {
				if opts.CustomRedactor != nil {
					return key + "=" + opts.CustomRedactor("postdatafield", key, value)
				}
				return key + "=" + replacement
			}
			// Name does not match: still apply value-pattern matching.
			if value != "" && len(valueRes) > 0 {
				return key + "=" + applyValuePatterns(value, valueRes, replacement)
			}
		}
		return match
	})
	return result
}

// (redactJSONKeys was removed; it previously redacted JSON by regex-matching "key":"value",
// and was replaced by redactJSONBody, which parses JSON and supports non-string values and nesting.)

// anonymizeIP replaces the last octet of an IPv4 address with .0.
// For IPv6, it replaces the last segment with :0.
// If the string is not a valid IP, it returns the replacement text.
func anonymizeIP(ip string) string {
	// Try IPv4 first
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		// Not a valid IP, just return the original
		return ip
	}

	if parsedIP.To4() != nil {
		// IPv4: replace last octet with 0
		parts := strings.Split(ip, ".")
		if len(parts) == 4 {
			return parts[0] + "." + parts[1] + "." + parts[2] + ".0"
		}
	}

	// IPv6: replace last hextet with :0
	// net.IP.String() for IPv6 addresses always contains ":", so LastIndex must be >= 0.
	str := parsedIP.String()
	lastColon := strings.LastIndex(str, ":")
	return str[:lastColon] + ":0"
}

// redactURLString redacts sensitive query parameters in a URL string
// and applies URL path segment redaction rules.
func redactURLString(rawURL string, opts RedactOptions, replacement string, valueRes []compiledValuePattern) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		// If URL can't be parsed, try simple string-based redaction
		return redactQueryStringSimple(rawURL, opts.QueryParams, replacement, opts, valueRes)
	}

	// Redact query parameters in the URL
	if parsed.RawQuery != "" {
		parsed.RawQuery = redactURLQuery(parsed.RawQuery, opts, replacement, valueRes)
	}

	// Apply URL path segment redaction rules
	if len(opts.RedactURLs) > 0 {
		parsed.Path = redactURLPath(parsed.Path, opts.RedactURLs)
	}

	return parsed.String()
}

// redactURLQuery redacts sensitive query parameters in a URL query string.
func redactURLQuery(query string, opts RedactOptions, replacement string, valueRes []compiledValuePattern) string {
	params := strings.Split(query, "&")
	var result []string
	for _, param := range params {
		if param == "" {
			continue
		}
		parts := strings.SplitN(param, "=", 2)
		key := parts[0]
		if len(parts) == 2 {
			value := parts[1]
			if matchesAny(key, opts.QueryParams) {
				if opts.CustomRedactor != nil {
					result = append(result, key+"="+opts.CustomRedactor("queryparam", key, value))
				} else {
					result = append(result, key+"="+replacement)
				}
			} else if value != "" && len(valueRes) > 0 {
				// Name does not match: still apply value-pattern matching.
				result = append(result, key+"="+applyValuePatterns(value, valueRes, replacement))
			} else {
				result = append(result, param)
			}
		} else {
			// No value, just a key
			if matchesAny(key, opts.QueryParams) {
				if opts.CustomRedactor != nil {
					result = append(result, key+"="+opts.CustomRedactor("queryparam", key, ""))
				} else {
					result = append(result, key+"="+replacement)
				}
			} else {
				result = append(result, param)
			}
		}
	}
	return strings.Join(result, "&")
}

// redactURLPath applies URL path segment redaction rules.
func redactURLPath(path string, rules []RedactURLRule) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		for _, rule := range rules {
			re, err := regexp.Compile(rule.Pattern)
			if err != nil {
				continue
			}
			if re.MatchString(segment) {
				segments[i] = re.ReplaceAllString(segment, rule.Replacement)
			}
		}
	}
	return strings.Join(segments, "/")
}

// redactQueryStringSimple redacts query parameters in a raw URL string
// without fully parsing it. Used as a fallback when url.Parse fails.
func redactQueryStringSimple(rawURL string, paramNames []string, replacement string, opts RedactOptions, valueRes []compiledValuePattern) string {
	for _, name := range paramNames {
		// Match name=value pattern
		re := regexp.MustCompile(`(?i)(` + regexp.QuoteMeta(name) + `)=([^&]*)`)
		rawURL = re.ReplaceAllStringFunc(rawURL, func(match string) string {
			// match is produced by a regex match, so FindStringSubmatch must return three groups
			// (the full match plus two capture groups).
			parts := re.FindStringSubmatch(match)
			if opts.CustomRedactor != nil {
				return parts[1] + "=" + opts.CustomRedactor("queryparam", parts[1], parts[2])
			}
			return parts[1] + "=" + replacement
		})
	}
	// Parameters whose names do not match still use value-pattern matching.
	if len(valueRes) > 0 {
		re := regexp.MustCompile(`([^&=]+)=([^&]*)`)
		rawURL = re.ReplaceAllStringFunc(rawURL, func(match string) string {
			parts := re.FindStringSubmatch(match)
			if len(parts) >= 3 && parts[2] != "" {
				return parts[1] + "=" + applyValuePatterns(parts[2], valueRes, replacement)
			}
			return match
		})
	}
	return rawURL
}
