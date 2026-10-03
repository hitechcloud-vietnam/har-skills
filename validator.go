package har

import (
	"fmt"
	"net/url"
	"strings"
)

// HAR specification version constants.
const (
	// HAR specification version 1.1.
	HarSpecVersion11 = "1.1"
	// HAR specification version 1.2.
	HarSpecVersion12 = "1.2"
	// HAR specification version 1.3 (unofficial, but used by some tools).
	HarSpecVersion13 = "1.3"
)

// ValidateHarFile validates the contents of a Har object.
// Supports different versions of the HAR specification.
func ValidateHarFile(har *Har) error {
	if har == nil {
		return NewInvalidFormatError("HAR object is nil")
	}

	// Create the root error.
	rootError := &HarError{
		Code:    ErrCodeValidation,
		Message: "HAR validation failed",
	}

	// Validate the basic structure.
	if err := validateBasicStructure(har, rootError); err != nil {
		return err
	}

	// Run version-specific validation.
	switch har.Log.Version {
	case HarSpecVersion11:
		validateHarV11(har, rootError)
	case HarSpecVersion12:
		validateHarV12(har, rootError)
	case HarSpecVersion13:
		validateHarV13(har, rootError)
	default:
		_ = rootError.AddPartialError(NewValidationError(
			fmt.Sprintf("Unsupported HAR version: %s", har.Log.Version),
			"log.version",
		))
	}

	// Validate common entry fields.
	validateEntries(har.Log.Entries, rootError)

	// Validate pages.
	validatePages(har.Log.Pages, rootError)

	// Return if there are partial errors.
	if rootError.HasPartialErrors() {
		return rootError
	}

	return nil
}

// validateBasicStructure validates the basic HAR structure.
func validateBasicStructure(har *Har, rootError *HarError) error {
	// Validate the Log field.
	if har.Log.Version == "" {
		_ = rootError.AddPartialError(NewMissingFieldError("log.version"))
	}

	// Validate the Creator field.
	if har.Log.Creator.Name == "" {
		_ = rootError.AddPartialError(NewMissingFieldError("log.creator.name"))
	}

	if har.Log.Creator.Version == "" {
		_ = rootError.AddPartialError(NewMissingFieldError("log.creator.version"))
	}

	// Validate the Browser field, if present.
	if har.Log.Browser.Name != "" && har.Log.Browser.Version == "" {
		_ = rootError.AddPartialError(NewValidationError(
			"Browser name is present but version is empty",
			"log.browser.version",
		))
	}

	// Validate the Entries array.
	// A HAR file may have no entries, but the Entries array must be present.
	if har.Log.Entries == nil {
		_ = rootError.AddPartialError(NewMissingFieldError("log.entries"))
	}

	// Return if there are partial errors.
	if rootError.HasPartialErrors() {
		return rootError
	}

	return nil
}

// validateHarV11 validates requirements specific to HAR 1.1.
func validateHarV11(har *Har, rootError *HarError) {
	// HAR 1.1: every PostData.params item must have a name.
	for i, entry := range har.Log.Entries {
		if entry.Request.PostData != nil && entry.Request.PostData.Params != nil {
			for j, param := range entry.Request.PostData.Params {
				if param.Name == "" {
					_ = rootError.AddPartialError(NewValidationError(
						"PostData parameter must have a name field",
						fmt.Sprintf("log.entries[%d].request.postData.params[%d].name", i, j),
					))
				}
			}
		}
	}
}

// validateHarV12 validates requirements specific to HAR 1.2.
func validateHarV12(har *Har, rootError *HarError) {
	// HAR 1.2: verify that every QueryString item has a name.
	for i, entry := range har.Log.Entries {
		for j, qs := range entry.Request.QueryString {
			if qs.Name == "" {
				_ = rootError.AddPartialError(NewValidationError(
					"QueryString parameter must have a name field",
					fmt.Sprintf("log.entries[%d].request.queryString[%d].name", i, j),
				))
			}
		}

		// Validate PostData, if present.
		if entry.Request.PostData != nil {
			if entry.Request.PostData.MimeType == "" {
				_ = rootError.AddPartialError(NewValidationError(
					"PostData must have a mimeType field",
					fmt.Sprintf("log.entries[%d].request.postData.mimeType", i),
				))
			}
		}
	}

	// Validate Content.encoding, if present; it must be base64.
	for i, entry := range har.Log.Entries {
		if entry.Response.Content.Encoding != "" &&
			!strings.EqualFold(entry.Response.Content.Encoding, "base64") {
			_ = rootError.AddPartialError(NewValidationError(
				fmt.Sprintf("Content.encoding supports only base64; got: %s", entry.Response.Content.Encoding),
				fmt.Sprintf("log.entries[%d].response.content.encoding", i),
			))
		}
	}
}

// validateHarV13 validates requirements specific to HAR 1.3.
func validateHarV13(har *Har, rootError *HarError) {
	// HAR 1.3-specific validation.
	// This version is unofficial but used by some tools.
	// Includes all HAR 1.2 validation.
	validateHarV12(har, rootError)
}

// validateEntries validates HAR entries.
func validateEntries(entries []Entries, rootError *HarError) {
	for i, entry := range entries {
		entryPrefix := fmt.Sprintf("log.entries[%d]", i)

		// Validate required timing fields.
		if entry.StartedDateTime.IsZero() {
			_ = rootError.AddPartialError(NewValidationError(
				"Entry must have a start time",
				fmt.Sprintf("%s.startedDateTime", entryPrefix),
			))
		}

		// Validate timing values.
		if entry.Time < 0 {
			_ = rootError.AddPartialError(NewValidationError(
				"Entry time cannot be negative",
				fmt.Sprintf("%s.time", entryPrefix),
			))
		}

		// Validate the request.
		validateRequest(entry.Request, fmt.Sprintf("%s.request", entryPrefix), rootError)

		// Validate the response.
		validateResponse(entry.Response, fmt.Sprintf("%s.response", entryPrefix), rootError)

		// Validate timing fields.
		validateTimings(entry.Timings, fmt.Sprintf("%s.timings", entryPrefix), rootError)

	}
}

// validateRequest validates an HTTP request.
func validateRequest(req Request, fieldPath string, rootError *HarError) {
	// Validate the method.
	if req.Method == "" {
		_ = rootError.AddPartialError(NewValidationError(
			"HTTP request must specify a method",
			fmt.Sprintf("%s.method", fieldPath),
		))
	}

	// Validate the URL.
	if req.URL == "" {
		_ = rootError.AddPartialError(NewValidationError(
			"HTTP request must specify a URL",
			fmt.Sprintf("%s.url", fieldPath),
		))
	} else {
		// Validate URL format.
		_, err := url.Parse(req.URL)
		if err != nil {
			_ = rootError.AddPartialError(NewValidationError(
				fmt.Sprintf("无效的URL格式: %s", err.Error()),
				fmt.Sprintf("%s.url", fieldPath),
			))
		}
	}

	// Validate the HTTP version.
	if req.HTTPVersion == "" {
		_ = rootError.AddPartialError(NewValidationError(
			"HTTP request must specify a version",
			fmt.Sprintf("%s.httpVersion", fieldPath),
		))
	}

	// Validate headers.
	validateHeaders(req.Headers, fmt.Sprintf("%s.headers", fieldPath), rootError)

	// Validate cookies.
	validateCookies(req.Cookies, fmt.Sprintf("%s.cookies", fieldPath), rootError)

	// Validate QueryString.
	validateQueryString(req.QueryString, fmt.Sprintf("%s.queryString", fieldPath), rootError)

	// Validate PostData, if present.
	if req.PostData != nil {
		validatePostData(req.PostData, fmt.Sprintf("%s.postData", fieldPath), rootError)
	}
}

// validateResponse validates an HTTP response.
func validateResponse(resp Response, fieldPath string, rootError *HarError) {
	// Validate the status code.
	if resp.Status <= 0 {
		_ = rootError.AddPartialError(NewValidationError(
			"HTTP response must have a valid status code",
			fmt.Sprintf("%s.status", fieldPath),
		))
	}

	// Validate the HTTP version.
	if resp.HTTPVersion == "" {
		_ = rootError.AddPartialError(NewValidationError(
			"HTTP response must specify a version",
			fmt.Sprintf("%s.httpVersion", fieldPath),
		))
	}

	// Validate content.
	validateContent(resp.Content, fmt.Sprintf("%s.content", fieldPath), rootError)

	// Validate headers.
	validateHeaders(resp.Headers, fmt.Sprintf("%s.headers", fieldPath), rootError)

	// Validate cookies.
	validateCookies(resp.Cookies, fmt.Sprintf("%s.cookies", fieldPath), rootError)
}

// validateContent validates content.
func validateContent(content Content, fieldPath string, rootError *HarError) {
	// Validate the MIME type.
	if content.MimeType == "" {
		_ = rootError.AddPartialError(NewValidationError(
			"Content must have a MIME type",
			fmt.Sprintf("%s.mimeType", fieldPath),
		))
	}

	// Validate size.
	if content.Size < 0 {
		_ = rootError.AddPartialError(NewValidationError(
			"Content size cannot be negative",
			fmt.Sprintf("%s.size", fieldPath),
		))
	}

	// Validate encoding, if present; it must be a known value.
	if content.Encoding != "" &&
		!strings.EqualFold(content.Encoding, "base64") {
		_ = rootError.AddPartialError(NewValidationError(
			fmt.Sprintf("Unsupported Content.encoding: %s (only base64 is supported)", content.Encoding),
			fmt.Sprintf("%s.encoding", fieldPath),
		))
	}
}

// validateHeaders validates HTTP headers.
func validateHeaders(headers []Headers, fieldPath string, rootError *HarError) {
	for i, header := range headers {
		headerPath := fmt.Sprintf("%s[%d]", fieldPath, i)

		if header.Name == "" {
			_ = rootError.AddPartialError(NewValidationError(
				"HTTP header must have a name",
				fmt.Sprintf("%s.name", headerPath),
			))
		}
	}
}

// validateCookies validates cookies.
func validateCookies(cookies []Cookie, fieldPath string, rootError *HarError) {
	for i, cookie := range cookies {
		cookiePath := fmt.Sprintf("%s[%d]", fieldPath, i)

		if cookie.Name == "" {
			_ = rootError.AddPartialError(NewValidationError(
				"Cookie must have a name",
				fmt.Sprintf("%s.name", cookiePath),
			))
		}
	}
}

// validateQueryString validates query parameters.
func validateQueryString(params []QueryString, fieldPath string, rootError *HarError) {
	for i, param := range params {
		paramPath := fmt.Sprintf("%s[%d]", fieldPath, i)

		if param.Name == "" {
			_ = rootError.AddPartialError(NewValidationError(
				"Query parameter must have a name",
				fmt.Sprintf("%s.name", paramPath),
			))
		}
	}
}

// validatePostData validates POST data.
func validatePostData(postData *PostData, fieldPath string, rootError *HarError) {
	// mimeType is required.
	if postData.MimeType == "" {
		_ = rootError.AddPartialError(NewValidationError(
			"PostData must have a mimeType",
			fmt.Sprintf("%s.mimeType", fieldPath),
		))
	}

	// Validate params, if present.
	for i, param := range postData.Params {
		paramPath := fmt.Sprintf("%s.params[%d]", fieldPath, i)

		if param.Name == "" {
			_ = rootError.AddPartialError(NewValidationError(
				"PostData parameter must have a name",
				fmt.Sprintf("%s.name", paramPath),
			))
		}
	}
}

// validateTimings validates timings.
func validateTimings(timings Timings, fieldPath string, rootError *HarError) {
	// Validate required timing fields.
	if timings.Wait < 0 {
		_ = rootError.AddPartialError(NewValidationError(
			"Wait time cannot be negative",
			fmt.Sprintf("%s.wait", fieldPath),
		))
	}

	if timings.Receive < 0 {
		_ = rootError.AddPartialError(NewValidationError(
			"Receive time cannot be negative",
			fmt.Sprintf("%s.receive", fieldPath),
		))
	}

	if timings.Send < 0 {
		_ = rootError.AddPartialError(NewValidationError(
			"Send time cannot be negative",
			fmt.Sprintf("%s.send", fieldPath),
		))
	}
}

// validatePages validates pages.
func validatePages(pages []Pages, rootError *HarError) {
	for i, page := range pages {
		pagePath := fmt.Sprintf("log.pages[%d]", i)

		// Validate the ID.
		if page.ID == "" {
			_ = rootError.AddPartialError(NewValidationError(
				"Page must have an ID",
				fmt.Sprintf("%s.id", pagePath),
			))
		}

		// Validate the start time.
		if page.StartedDateTime.IsZero() {
			_ = rootError.AddPartialError(NewValidationError(
				"Page must have a start time",
				fmt.Sprintf("%s.startedDateTime", pagePath),
			))
		}

		// Validate page load timings.
		validatePageTimings(page.PageTimings, fmt.Sprintf("%s.pageTimings", pagePath), rootError)

		// Validate the page title.
		if page.Title == "" {
			_ = rootError.AddPartialError(NewValidationError(
				"Page must have a title",
				fmt.Sprintf("%s.title", pagePath),
			))
		}
	}
}

// validatePageTimings validates page load timings.
func validatePageTimings(timings PageTimings, fieldPath string, rootError *HarError) {
	// onContentLoad and onLoad may be negative to indicate unavailable values.
	// but should not be extreme values.
	if timings.OnContentLoad < -1 {
		_ = rootError.AddPartialError(NewValidationError(
			fmt.Sprintf("Invalid page content load time: %f", timings.OnContentLoad),
			fmt.Sprintf("%s.onContentLoad", fieldPath),
		))
	}

	if timings.OnLoad < -1 {
		_ = rootError.AddPartialError(NewValidationError(
			fmt.Sprintf("Invalid page load time: %f", timings.OnLoad),
			fmt.Sprintf("%s.onLoad", fieldPath),
		))
	}
}

// IsValidHarVersion checks whether the HAR version is supported.
func IsValidHarVersion(version string) bool {
	return version == HarSpecVersion11 ||
		version == HarSpecVersion12 ||
		version == HarSpecVersion13
}

// DetectHarVersion detects the HAR version.
func DetectHarVersion(har *Har) string {
	if har == nil || har.Log.Version == "" {
		return HarSpecVersion12 // Defaults to version 1.2.
	}

	version := strings.TrimSpace(har.Log.Version)
	if IsValidHarVersion(version) {
		return version
	}

	// If the version is unsupported, try to normalize it.
	if strings.HasPrefix(version, "1.1") {
		return HarSpecVersion11
	} else if strings.HasPrefix(version, "1.2") {
		return HarSpecVersion12
	} else if strings.HasPrefix(version, "1.3") {
		return HarSpecVersion13
	}

	return HarSpecVersion12 // 默认
}
