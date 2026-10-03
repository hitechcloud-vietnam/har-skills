package har

import (
	"net/url"
	"regexp"
	"strings"
)

// TransformType defines the type of transformation rule.
type TransformType int

const (
	TransformURLRewrite          TransformType = iota // URL rewrite.
	TransformHostReplace                              // Host replacement.
	TransformSchemeChange                             // Scheme change.
	TransformHeaderAdd                                // Add a request header.
	TransformHeaderRemove                             // Remove request headers.
	TransformHeaderReplace                            // Replace request headers.
	TransformQueryParamRemove                         // Remove query parameters.
	TransformQueryParamAdd                            // Add a query parameter.
	TransformCookieDomainRewrite                      // Rewrite cookie domain.
	TransformBodyReplace                              // Replace the request body.
)

// TransformRule defines a transformation rule.
type TransformRule struct {
	Type        TransformType // Transformation type.
	Pattern     string        // Regular-expression or string-matching pattern.
	Replacement string        // Replacement string.
	HeaderName  string        // Header name used for request-header transformations.
	HeaderValue string        // Header value used to add or replace a request header.
}

// Transform applies transformation rules to a Har object and returns a new transformed clone.
func (h *Har) Transform(rules []TransformRule) *Har {
	if h == nil {
		return nil
	}

	cloned := h.Clone()
	if cloned == nil {
		return nil
	}
	cloned.TransformInPlace(rules)
	return cloned
}

// TransformInPlace applies transformation rules in place.
func (h *Har) TransformInPlace(rules []TransformRule) {
	if h == nil || len(rules) == 0 {
		return
	}

	for i := range h.Log.Entries {
		applyRules(&h.Log.Entries[i], rules)
	}
}

// RewriteURL is a convenience method that replaces a URL prefix and returns a new Har object.
// Example: RewriteURL("http://localhost:8080", "https://prod.example.com").
func (h *Har) RewriteURL(from, to string) *Har {
	return h.Transform([]TransformRule{
		{
			Type:        TransformURLRewrite,
			Pattern:     from,
			Replacement: to,
		},
	})
}

// RemoveHeaders removes the specified headers from all requests and responses and returns a new Har object.
func (h *Har) RemoveHeaders(names []string) *Har {
	rules := make([]TransformRule, len(names))
	for i, name := range names {
		rules[i] = TransformRule{
			Type:       TransformHeaderRemove,
			HeaderName: name,
		}
	}
	return h.Transform(rules)
}

// AddHeaders adds headers to all requests and/or responses and returns a new Har object.
// target is "request", "response", or "both".
func (h *Har) AddHeaders(headers map[string]string, target string) *Har {
	if h == nil {
		return nil
	}

	var rules []TransformRule
	for name, value := range headers {
		rules = append(rules, TransformRule{
			Type:        TransformHeaderAdd,
			HeaderName:  name,
			HeaderValue: value,
		})
	}

	cloned := h.Clone()
	if cloned == nil {
		return nil
	}

	for i := range cloned.Log.Entries {
		entry := &cloned.Log.Entries[i]
		for _, rule := range rules {
			if target == "request" || target == "both" {
				entry.Request.Headers = append(entry.Request.Headers, Headers{
					Name:  rule.HeaderName,
					Value: rule.HeaderValue,
				})
			}
			if target == "response" || target == "both" {
				entry.Response.Headers = append(entry.Response.Headers, Headers{
					Name:  rule.HeaderName,
					Value: rule.HeaderValue,
				})
			}
		}
	}

	return cloned
}

// applyRules applies all transformation rules to a single entry.
func applyRules(entry *Entries, rules []TransformRule) {
	if entry == nil || len(rules) == 0 {
		return
	}

	for _, rule := range rules {
		switch rule.Type {
		case TransformURLRewrite:
			applyURLRewrite(entry, rule)
		case TransformHostReplace:
			applyHostReplace(entry, rule)
		case TransformSchemeChange:
			applySchemeChange(entry, rule)
		case TransformHeaderAdd:
			applyHeaderAdd(entry, rule)
		case TransformHeaderRemove:
			applyHeaderRemove(entry, rule)
		case TransformHeaderReplace:
			applyHeaderReplace(entry, rule)
		case TransformQueryParamRemove:
			applyQueryParamRemove(entry, rule)
		case TransformQueryParamAdd:
			applyQueryParamAdd(entry, rule)
		case TransformCookieDomainRewrite:
			applyCookieDomainRewrite(entry, rule)
		case TransformBodyReplace:
			applyBodyReplace(entry, rule)
		}
	}
}

// applyURLRewrite replaces a URL prefix.
func applyURLRewrite(entry *Entries, rule TransformRule) {
	if entry == nil {
		return
	}
	if strings.HasPrefix(entry.Request.URL, rule.Pattern) {
		newURL := rule.Replacement + entry.Request.URL[len(rule.Pattern):]
		entry.Request.URL = newURL

		// Update QueryString if URL parsing changes it.
		entry.Request.QueryString = BuildQueryStringFromURL(newURL)

		// Update the Host request header.
		if u, err := url.Parse(newURL); err == nil {
			updateHostHeader(entry, u.Host)
		}
	}
}

// applyHostReplace replaces the hostname.
func applyHostReplace(entry *Entries, rule TransformRule) {
	if entry == nil {
		return
	}
	if u, err := url.Parse(entry.Request.URL); err == nil {
		if u.Host == rule.Pattern {
			u.Host = rule.Replacement
			newURL := u.String()
			entry.Request.URL = newURL
			entry.Request.QueryString = BuildQueryStringFromURL(newURL)
			updateHostHeader(entry, rule.Replacement)
		}
	}
}

// applySchemeChange changes the scheme (http <-> https).
func applySchemeChange(entry *Entries, rule TransformRule) {
	if entry == nil {
		return
	}
	if u, err := url.Parse(entry.Request.URL); err == nil {
		if u.Scheme == rule.Pattern {
			u.Scheme = rule.Replacement
			newURL := u.String()
			entry.Request.URL = newURL
			entry.Request.QueryString = BuildQueryStringFromURL(newURL)
		}
	}
}

// applyHeaderAdd adds a request header to requests and responses.
func applyHeaderAdd(entry *Entries, rule TransformRule) {
	if entry == nil {
		return
	}
	entry.Request.Headers = append(entry.Request.Headers, Headers{
		Name:  rule.HeaderName,
		Value: rule.HeaderValue,
	})
	entry.Response.Headers = append(entry.Response.Headers, Headers{
		Name:  rule.HeaderName,
		Value: rule.HeaderValue,
	})
}

// applyHeaderRemove removes a header from requests and responses.
func applyHeaderRemove(entry *Entries, rule TransformRule) {
	if entry == nil {
		return
	}
	entry.Request.Headers = removeHeaderByName(entry.Request.Headers, rule.HeaderName)
	entry.Response.Headers = removeHeaderByName(entry.Response.Headers, rule.HeaderName)
}

// applyHeaderReplace replaces a request header value.
func applyHeaderReplace(entry *Entries, rule TransformRule) {
	if entry == nil {
		return
	}
	replaceHeaderValue(entry.Request.Headers, rule.HeaderName, rule.HeaderValue)
	replaceHeaderValue(entry.Response.Headers, rule.HeaderName, rule.HeaderValue)
}

// applyQueryParamRemove removes the specified query parameter.
func applyQueryParamRemove(entry *Entries, rule TransformRule) {
	if entry == nil {
		return
	}
	// If QueryString is empty, parse from URL
	if len(entry.Request.QueryString) == 0 {
		entry.Request.QueryString = BuildQueryStringFromURL(entry.Request.URL)
	}

	newQS := make([]QueryString, 0, len(entry.Request.QueryString))
	for _, q := range entry.Request.QueryString {
		if q.Name != rule.Pattern {
			newQS = append(newQS, q)
		}
	}
	entry.Request.QueryString = newQS

	// Rebuild the URL.
	rebuildURLFromQueryString(entry)
}

// applyQueryParamAdd adds a query parameter.
func applyQueryParamAdd(entry *Entries, rule TransformRule) {
	if entry == nil {
		return
	}
	// If QueryString is empty, parse from URL
	if len(entry.Request.QueryString) == 0 {
		entry.Request.QueryString = BuildQueryStringFromURL(entry.Request.URL)
	}

	entry.Request.QueryString = append(entry.Request.QueryString, QueryString{
		Name:  rule.HeaderName,
		Value: rule.HeaderValue,
	})

	// Rebuild the URL.
	rebuildURLFromQueryString(entry)
}

// applyCookieDomainRewrite rewrites the cookie domain.
func applyCookieDomainRewrite(entry *Entries, rule TransformRule) {
	if entry == nil {
		return
	}
	for i := range entry.Request.Cookies {
		if entry.Request.Cookies[i].Domain == rule.Pattern {
			entry.Request.Cookies[i].Domain = rule.Replacement
		}
	}
	for i := range entry.Response.Cookies {
		if entry.Response.Cookies[i].Domain == rule.Pattern {
			entry.Response.Cookies[i].Domain = rule.Replacement
		}
	}
}

// applyBodyReplace replaces request body text.
func applyBodyReplace(entry *Entries, rule TransformRule) {
	if entry == nil {
		return
	}
	if entry.Request.PostData == nil {
		return
	}
	if rule.Pattern == "" {
		return
	}

	re, err := regexp.Compile(rule.Pattern)
	if err != nil {
		// Not a valid regular expression; replace it as a plain string.
		entry.Request.PostData.Text = strings.ReplaceAll(
			entry.Request.PostData.Text, rule.Pattern, rule.Replacement,
		)
		return
	}
	entry.Request.PostData.Text = re.ReplaceAllString(
		entry.Request.PostData.Text, rule.Replacement,
	)
}

// updateHostHeader updates the Host request header.
func updateHostHeader(entry *Entries, host string) {
	if entry == nil {
		return
	}
	for i := range entry.Request.Headers {
		if strings.EqualFold(entry.Request.Headers[i].Name, "Host") {
			entry.Request.Headers[i].Value = host
			return
		}
	}
	// Add a Host header if one is missing.
	entry.Request.Headers = append(entry.Request.Headers, Headers{
		Name:  "Host",
		Value: host,
	})
}

// removeHeaderByName removes a header by name (case-insensitive).
func removeHeaderByName(headers []Headers, name string) []Headers {
	result := make([]Headers, 0, len(headers))
	for _, h := range headers {
		if !strings.EqualFold(h.Name, name) {
			result = append(result, h)
		}
	}
	return result
}

// replaceHeaderValue replaces the value of a named header (case-insensitive).
func replaceHeaderValue(headers []Headers, name, value string) {
	for i := range headers {
		if strings.EqualFold(headers[i].Name, name) {
			headers[i].Value = value
		}
	}
}

// rebuildURLFromQueryString rebuilds the URL from the current URL and QueryString.
func rebuildURLFromQueryString(entry *Entries) {
	if entry == nil {
		return
	}
	u, err := url.Parse(entry.Request.URL)
	if err != nil {
		return
	}

	// Clear existing parameters and use the values from QueryString.
	q := url.Values{}
	for _, qs := range entry.Request.QueryString {
		q.Set(qs.Name, qs.Value)
	}
	u.RawQuery = q.Encode()
	entry.Request.URL = u.String()
}
