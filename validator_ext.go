package har

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidationRule defines a custom validation rule.
type ValidationRule struct {
	Name        string                            // Rule name.
	Description string                            // Rule description.
	Validate    func(har *Har) []*ValidationError // Validation function.
}

// ValidationError represents a validation error.
type ValidationError struct {
	Field   string // Field path.
	Message string // Error message.
	Rule    string // Name of the triggered rule.
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Rule != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Rule, e.Field, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// customRules stores custom validation rules.
var customRules []ValidationRule

// RegisterValidator registers a custom validation rule.
//
// Registered rules are run when ValidateWithRules is called.
// Rule names must be unique; registering an existing name replaces the previous rule.
//
// Example:
//
//	har.RegisterValidator("no-internal-ips", har.ValidationRule{
//	    Name:        "no-internal-ips",
//	    Description: "Block access to private network IP addresses.",
//	    Validate: func(h *har.Har) []*har.ValidationError {
//	        var errors []*har.ValidationError
//	        for _, entry := range h.Log.Entries {
//	            if isInternalIP(entry.ServerIPAddress) {
//	                errors = append(errors, &har.ValidationError{
//	                    Field:   "serverIPAddress",
//	                    Message: fmt.Sprintf("Private network IP address: %s", entry.ServerIPAddress),
//	                    Rule:    "no-internal-ips",
//	                })
//	            }
//	        }
//	        return errors
//	    },
//	})
func RegisterValidator(name string, rule ValidationRule) {
	// Check for an existing rule with the same name and replace it if found.
	for i, r := range customRules {
		if r.Name == name {
			customRules[i] = rule
			return
		}
	}
	customRules = append(customRules, rule)
}

// UnregisterValidator removes a custom validation rule.
func UnregisterValidator(name string) {
	for i, r := range customRules {
		if r.Name == name {
			customRules = append(customRules[:i], customRules[i+1:]...)
			return
		}
	}
}

// ListValidators lists all registered custom validation rules.
func ListValidators() []ValidationRule {
	result := make([]ValidationRule, len(customRules))
	copy(result, customRules)
	return result
}

// ValidateWithRules validates a Har object using custom rules.
//
// This method first validates against the standard HAR specification, then runs all registered custom rules.
// Returns standard validation errors and custom-rule validation errors.
func ValidateWithRules(har *Har) error {
	if har == nil {
		return NewInvalidFormatError("HAR object is nil")
	}

	// Run standard validation.
	stdErr := ValidateHarFile(har)

	// Run custom-rule validation.
	var customErrors []*ValidationError
	for _, rule := range customRules {
		if rule.Validate == nil {
			customErrors = append(customErrors, &ValidationError{
				Field:   "validator",
				Message: "Validate function is nil",
				Rule:    rule.Name,
			})
			continue
		}
		for _, err := range rule.Validate(har) {
			if err != nil {
				customErrors = append(customErrors, err)
			}
		}
	}

	if stdErr != nil && len(customErrors) > 0 {
		// Combine standard and custom errors.
		if harErr, ok := stdErr.(*HarError); ok {
			for _, ce := range customErrors {
				_ = harErr.AddPartialError(NewValidationError(
					fmt.Sprintf("[%s] %s", ce.Rule, ce.Message),
					ce.Field,
				))
			}
			return harErr
		}
	}

	if len(customErrors) > 0 {
		rootError := &HarError{
			Code:    ErrCodeValidation,
			Message: "HAR validation failed (including custom rules)",
		}
		for _, ce := range customErrors {
			_ = rootError.AddPartialError(NewValidationError(
				fmt.Sprintf("[%s] %s", ce.Rule, ce.Message),
				ce.Field,
			))
		}
		return rootError
	}

	return stdErr
}

// ValidateStrict strictly validates a Har object.
//
// More strict than standard validation; includes:
// - Validate pageref cross-references.
// - Validate page ID uniqueness.
// - Validate HTTP method values.
// - Validate status-code ranges.
// - Validate Cookie.SameSite values.
// - Validate consistency between Timings and Time.
// - Validate required Cache fields.
func ValidateStrict(har *Har) error {
	if har == nil {
		return NewInvalidFormatError("HAR object is nil")
	}

	// Run standard validation first.
	rootError := &HarError{
		Code:    ErrCodeValidation,
		Message: "Strict HAR validation failed",
	}

	// Standard validation.
	if err := ValidateHarFile(har); err != nil {
		if harErr, ok := err.(*HarError); ok {
			for _, pe := range harErr.GetPartialErrors() {
				_ = rootError.AddPartialError(pe)
			}
		}
	}

	// Strict validation: pageref cross-references.
	validatePagerefReferences(har, rootError)

	// Strict validation: page ID uniqueness.
	validatePageIDUniqueness(har, rootError)

	// Strict validation: HTTP method values.
	validateHTTPMethods(har, rootError)

	// Strict validation: status-code ranges.
	validateStatusCodeRange(har, rootError)

	// Strict validation: Cookie.SameSite values.
	validateCookieSameSite(har, rootError)

	// Strict validation: required Cache fields.
	validateCacheFields(har, rootError)

	if rootError.HasPartialErrors() {
		return rootError
	}

	return nil
}

// validatePagerefReferences checks whether pages referenced by pageref exist.
func validatePagerefReferences(har *Har, rootError *HarError) {
	// Build the set of page IDs.
	pageIDs := make(map[string]bool)
	for _, page := range har.Log.Pages {
		pageIDs[page.ID] = true
	}

	// Check each entry pageref.
	for i, entry := range har.Log.Entries {
		if entry.Pageref != "" {
			if !pageIDs[entry.Pageref] {
				_ = rootError.AddPartialError(NewValidationError(
					fmt.Sprintf("Entry references a non-existent page ID '%s' does not exist", entry.Pageref),
					fmt.Sprintf("log.entries[%d].pageref", i),
				))
			}
		}
	}
}

// validatePageIDUniqueness validates page ID uniqueness.
func validatePageIDUniqueness(har *Har, rootError *HarError) {
	seen := make(map[string]int) // ID -> first occurrence index
	for i, page := range har.Log.Pages {
		if prevIdx, exists := seen[page.ID]; exists {
			_ = rootError.AddPartialError(NewValidationError(
				fmt.Sprintf("Page ID '%s' is duplicated (first seen at index %d)", page.ID, prevIdx),
				fmt.Sprintf("log.pages[%d].id", i),
			))
		} else {
			seen[page.ID] = i
		}
	}
}

// validateHTTPMethods validates HTTP method values.
func validateHTTPMethods(har *Har, rootError *HarError) {
	validMethods := map[string]bool{
		"GET":     true,
		"POST":    true,
		"PUT":     true,
		"DELETE":  true,
		"HEAD":    true,
		"OPTIONS": true,
		"PATCH":   true,
		"CONNECT": true,
		"TRACE":   true,
	}

	for i, entry := range har.Log.Entries {
		method := strings.ToUpper(entry.Request.Method)
		if !validMethods[method] {
			_ = rootError.AddPartialError(NewValidationError(
				fmt.Sprintf("Uncommon HTTP method: %s（standard methods: GET, POST, PUT, DELETE, HEAD, OPTIONS, PATCH, CONNECT, TRACE）", entry.Request.Method),
				fmt.Sprintf("log.entries[%d].request.method", i),
			))
		}
	}
}

// validateStatusCodeRange validates the status-code range.
func validateStatusCodeRange(har *Har, rootError *HarError) {
	for i, entry := range har.Log.Entries {
		status := entry.Response.Status
		if status < 100 || status > 599 {
			_ = rootError.AddPartialError(NewValidationError(
				fmt.Sprintf("Invalid HTTP status code: %d（valid range: 100-599）", status),
				fmt.Sprintf("log.entries[%d].response.status", i),
			))
		}
	}
}

// validateCookieSameSite validates Cookie.SameSite values.
func validateCookieSameSite(har *Har, rootError *HarError) {
	validSameSite := map[string]bool{
		"Strict": true,
		"Lax":    true,
		"None":   true,
		"":       true, // Empty is valid (not set).
	}

	for i, entry := range har.Log.Entries {
		// Check request cookies.
		for j, cookie := range entry.Request.Cookies {
			if !validSameSite[cookie.SameSite] {
				_ = rootError.AddPartialError(NewValidationError(
					fmt.Sprintf("Invalid Cookie.SameSite value: '%s'(valid values: Strict, Lax, None).", cookie.SameSite),
					fmt.Sprintf("log.entries[%d].request.cookies[%d].sameSite", i, j),
				))
			}
		}

		// Check response cookies.
		for j, cookie := range entry.Response.Cookies {
			if !validSameSite[cookie.SameSite] {
				_ = rootError.AddPartialError(NewValidationError(
					fmt.Sprintf("Invalid Cookie.SameSite value: '%s'(valid values: Strict, Lax, None).", cookie.SameSite),
					fmt.Sprintf("log.entries[%d].response.cookies[%d].sameSite", i, j),
				))
			}
		}
	}
}

// validateCacheFields validates required Cache fields.
func validateCacheFields(har *Har, rootError *HarError) {
	for i, entry := range har.Log.Entries {
		entryPrefix := fmt.Sprintf("log.entries[%d]", i)

		// Validate BeforeRequest.
		if entry.Cache.BeforeRequest != nil {
			br := entry.Cache.BeforeRequest
			brPrefix := fmt.Sprintf("%s.cache.beforeRequest", entryPrefix)

			if br.LastAccess.IsZero() {
				_ = rootError.AddPartialError(NewValidationError(
					"BeforeRequest must have the lastAccess field",
					fmt.Sprintf("%s.lastAccess", brPrefix),
				))
			}
			if br.ETag == "" {
				_ = rootError.AddPartialError(NewValidationError(
					"BeforeRequest must have the eTag field",
					fmt.Sprintf("%s.eTag", brPrefix),
				))
			}
			if br.HitCount < 0 {
				_ = rootError.AddPartialError(NewValidationError(
					"BeforeRequest.hitCount cannot be negative",
					fmt.Sprintf("%s.hitCount", brPrefix),
				))
			}
		}

		// Validate AfterRequest.
		if entry.Cache.AfterRequest != nil {
			ar := entry.Cache.AfterRequest
			arPrefix := fmt.Sprintf("%s.cache.afterRequest", entryPrefix)

			if ar.LastAccess.IsZero() {
				_ = rootError.AddPartialError(NewValidationError(
					"AfterRequest must have the lastAccess field",
					fmt.Sprintf("%s.lastAccess", arPrefix),
				))
			}
			if ar.ETag == "" {
				_ = rootError.AddPartialError(NewValidationError(
					"AfterRequest must have the eTag field",
					fmt.Sprintf("%s.eTag", arPrefix),
				))
			}
			if ar.HitCount < 0 {
				_ = rootError.AddPartialError(NewValidationError(
					"AfterRequest.hitCount cannot be negative",
					fmt.Sprintf("%s.hitCount", arPrefix),
				))
			}
		}
	}
}

// ValidateURL strictly validates URL format.
//
// Check whether the URL contains a valid scheme and host.
func ValidateURL(rawURL string) *ValidationError {
	if rawURL == "" {
		return &ValidationError{
			Field:   "url",
			Message: "URL cannot be empty",
		}
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return &ValidationError{
			Field:   "url",
			Message: fmt.Sprintf("URL parsing failed: %v", err),
		}
	}

	if parsed.Scheme == "" {
		return &ValidationError{
			Field:   "url.scheme",
			Message: "URL is missing a scheme (such as http:// or https://)",
		}
	}

	if parsed.Host == "" {
		return &ValidationError{
			Field:   "url.host",
			Message: "URL is missing a host",
		}
	}

	return nil
}

// ValidateTimingsConsistency checks consistency between Timings and Entries.Time.
//
// The HAR specification requires Time to equal the sum of the timing fields, minus the overlap between SSL and Connect.
// This method checks whether the difference exceeds the specified tolerance (ms).
func ValidateTimingsConsistency(har *Har, tolerance float64) []*ValidationError {
	var errors []*ValidationError

	if har == nil {
		return []*ValidationError{
			{
				Field:   "har",
				Message: "HAR object is nil",
				Rule:    "timings-consistency",
			},
		}
	}

	for i, entry := range har.Log.Entries {
		// Calculate the timing total.
		var sum float64
		if entry.Timings.Blocked > 0 {
			sum += entry.Timings.Blocked
		}
		if entry.Timings.DNS > 0 {
			sum += entry.Timings.DNS
		}
		if entry.Timings.Connect > 0 {
			sum += entry.Timings.Connect
		}
		// SSL time is included in Connect and is not counted twice (when Connect > 0 and SSL > 0).
		if entry.Timings.Ssl > 0 && entry.Timings.Connect <= 0 {
			sum += entry.Timings.Ssl
		}
		if entry.Timings.Send > 0 {
			sum += entry.Timings.Send
		}
		if entry.Timings.Wait > 0 {
			sum += entry.Timings.Wait
		}
		if entry.Timings.Receive > 0 {
			sum += entry.Timings.Receive
		}

		// Check the difference.
		diff := entry.Time - sum
		if diff < 0 {
			diff = -diff
		}
		if diff > tolerance {
			errors = append(errors, &ValidationError{
				Field:   fmt.Sprintf("log.entries[%d].time", i),
				Message: fmt.Sprintf("Time field (%.2f ms) differs from the sum of Timings (%.2f ms) by more than the tolerance (%.2f ms)", entry.Time, sum, tolerance),
				Rule:    "timings-consistency",
			})
		}
	}

	return errors
}
