package har

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// ParseHarWithOptions parses HAR bytes using custom parsing options.
func ParseHarWithOptions(harFileBytes []byte, options ParseOptions) (*Har, error) {
	if len(harFileBytes) == 0 {
		return nil, NewInvalidFormatError("Input is empty")
	}

	// Check whether the file is JSON.
	if !isJSONContent(harFileBytes) {
		return nil, NewInvalidFormatError("Input is not valid JSON")
	}

	// In strict mode, parse directly.
	if !options.Lenient {
		har := new(Har)
		err := json.Unmarshal(harFileBytes, har)
		if err != nil {
			return nil, WrapJSONUnmarshalError(err)
		}

		// Validate if requested.
		if !options.SkipValidation {
			if err := validateHar(har); err != nil {
				return nil, err
			}
		}

		return har, nil
	}

	// Lenient mode: attempt to parse as much content as possible.
	return parseLenient(harFileBytes, options)
}

// ParseHarFileWithOptions parses a HAR file using custom parsing options.
func ParseHarFileWithOptions(harFilePath string, options ParseOptions) (*Har, error) {
	harFileBytes, err := os.ReadFile(harFilePath)
	if err != nil {
		return nil, NewFileSystemError(fmt.Sprintf("Unable to read file '%s'", harFilePath), err)
	}

	har, err := ParseHarWithOptions(harFileBytes, options)
	if err != nil {
		harErr, ok := err.(*HarError)
		if ok {
			_ = harErr.WithMetadata("filePath", harFilePath)
		}
		return nil, err
	}

	return har, nil
}

// ParseHarEnhanced parses HAR data and provides detailed error information.
func ParseHarEnhanced(harFileBytes []byte) (*Har, *HarError) {
	har, err := ParseHarWithOptions(harFileBytes, DefaultParseOptions())
	if err != nil {
		// All error paths in ParseHarWithOptions return *HarError.
		return nil, err.(*HarError)
	}
	return har, nil
}

// ParseHarFileEnhanced parses HAR files with detailed error information.
func ParseHarFileEnhanced(harFilePath string) (*Har, *HarError) {
	har, err := ParseHarFileWithOptions(harFilePath, DefaultParseOptions())
	if err != nil {
		// All error paths in ParseHarFileWithOptions return *HarError.
		return nil, err.(*HarError)
	}
	return har, nil
}

// ParseHarLenient parses HAR file contents in lenient mode.
func ParseHarLenient(harFileBytes []byte) (*Har, error) {
	options := DefaultParseOptions()
	options.Lenient = true
	options.CollectWarnings = true
	return ParseHarWithOptions(harFileBytes, options)
}

// ParseHarFileLenient parses a HAR file in lenient mode.
func ParseHarFileLenient(harFilePath string) (*Har, error) {
	options := DefaultParseOptions()
	options.Lenient = true
	options.CollectWarnings = true
	return ParseHarFileWithOptions(harFilePath, options)
}

// isJSONContent checks whether the content is JSON.
func isJSONContent(content []byte) bool {
	trimmed := strings.TrimSpace(string(content))
	return (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]"))
}

// validateHar validates the contents of a Har object.
// This function now delegates to ValidateHarFile, implemented in validator.go.
func validateHar(har *Har) error {
	if har == nil {
		return NewInvalidFormatError("HAR object is nil")
	}

	return ValidateHarFile(har)
}

// parseLenient parses in lenient mode, attempting to parse as much content as possible.
func parseLenient(harFileBytes []byte, options ParseOptions) (*Har, error) {
	// Create an empty Har object.
	har := &Har{
		Log: Log{
			Entries: []Entries{},
			Pages:   []Pages{},
		},
	}

	// Use a map for initial parsing so other fields can still be parsed when some fields are invalid.
	var rawData map[string]json.RawMessage
	if err := json.Unmarshal(harFileBytes, &rawData); err != nil {
		return nil, WrapJSONUnmarshalError(err)
	}

	// Track all errors.
	rootError := &HarError{
		Code:    ErrCodeJSONParse,
		Message: "An error occurred while parsing the HAR file, but some content was parsed successfully",
	}

	// Parse the log field.
	if logBytes, ok := rawData["log"]; ok {
		var logData map[string]json.RawMessage
		if err := json.Unmarshal(logBytes, &logData); err != nil {
			_ = rootError.AddPartialError(
				NewJSONParseError("Unable to parse the log field", err).WithField("log"))
		} else {
			// Parse the version field.
			if versionBytes, ok := logData["version"]; ok {
				var version string
				if err := json.Unmarshal(versionBytes, &version); err == nil {
					har.Log.Version = version
				} else {
					_ = rootError.AddPartialError(
						NewJSONParseError("Unable to parse the version field", err).WithField("log.version"))
				}
			}

			// Parse the creator field.
			if creatorBytes, ok := logData["creator"]; ok {
				var creator Creator
				if err := json.Unmarshal(creatorBytes, &creator); err == nil {
					har.Log.Creator = creator
				} else {
					_ = rootError.AddPartialError(
						NewJSONParseError("Unable to parse the creator field", err).WithField("log.creator"))
				}
			}

			// Parse the pages field.
			if pagesBytes, ok := logData["pages"]; ok {
				var pages []json.RawMessage
				if err := json.Unmarshal(pagesBytes, &pages); err == nil {
					for i, pageBytes := range pages {
						var page Pages
						if err := json.Unmarshal(pageBytes, &page); err == nil {
							har.Log.Pages = append(har.Log.Pages, page)
						} else {
							_ = rootError.AddPartialError(
								NewJSONParseError(
									fmt.Sprintf("Unable to parse page %d", i+1), err).
									WithField(fmt.Sprintf("log.pages[%d]", i)))
						}
					}
				} else {
					_ = rootError.AddPartialError(
						NewJSONParseError("Unable to parse the pages field", err).WithField("log.pages"))
				}
			}

			// Parse the entries field, which is the most important part.
			if entriesBytes, ok := logData["entries"]; ok {
				var entries []json.RawMessage
				if err := json.Unmarshal(entriesBytes, &entries); err == nil {
					for i, entryBytes := range entries {
						var entry Entries
						if err := json.Unmarshal(entryBytes, &entry); err == nil {
							har.Log.Entries = append(har.Log.Entries, entry)
						} else {
							_ = rootError.AddPartialError(
								NewJSONParseError(
									fmt.Sprintf("Unable to parse entry %d", i+1), err).
									WithField(fmt.Sprintf("log.entries[%d]", i)))
						}
					}
				} else {
					_ = rootError.AddPartialError(
						NewJSONParseError("Unable to parse the entries field", err).WithField("log.entries"))
				}
			}
		}
	} else {
		_ = rootError.AddPartialError(NewMissingFieldError("log"))
	}

	// If there are errors and the options specify collecting warnings.
	if rootError.HasPartialErrors() && options.CollectWarnings {
		// If some content was parsed, return the Har object and the error.
		if har.Log.Version != "" || len(har.Log.Entries) > 0 || len(har.Log.Pages) > 0 {
			return har, rootError
		}
		// Otherwise, treat parsing as a complete failure.
		return nil, rootError
	} else if rootError.HasPartialErrors() {
		// If there are errors but warnings are not collected, return only the error.
		return nil, rootError
	}

	return har, nil
}

// Result contains the parsed Har object and any warnings.
type Result struct {
	Har      *Har
	Warnings []*HarError
}

// ParseHarWithWarnings parses a HAR file and returns warnings.
// This function parses in lenient mode and collects all warnings instead of failing immediately.
func ParseHarWithWarnings(harFileBytes []byte) (*Result, error) {
	// Use lenient mode and collect warnings.
	options := DefaultParseOptions()
	options.Lenient = true
	options.CollectWarnings = true

	// Parse HAR data.
	har, err := ParseHarWithOptions(harFileBytes, options)

	// Initialize the result object.
	result := &Result{
		Har:      har,
		Warnings: []*HarError{},
	}

	// Handle warnings from the parsing phase.
	if err != nil {
		if harErr, ok := err.(*HarError); ok && har != nil {
			// In lenient mode, convert parsing errors into warnings.
			result.Warnings = appendWarnings(result.Warnings, harErr.GetPartialErrors())
		} else {
			// Handle the case where parsing fails completely.
			return nil, err
		}
	}

	// Validate URLs.
	urlWarnings := validateURLs(har)
	if len(urlWarnings) > 0 {
		result.Warnings = appendWarnings(result.Warnings, urlWarnings)
	}

	// If no warnings have been found, attempt full validation.
	if len(result.Warnings) == 0 {
		validationWarnings := performFullValidation(har)
		result.Warnings = appendWarnings(result.Warnings, validationWarnings)
	}

	return result, nil
}

// validateURLs validates URL fields in all entries.
func validateURLs(har *Har) []*HarError {
	if har == nil || len(har.Log.Entries) == 0 {
		return nil
	}

	var warnings []*HarError
	for i, entry := range har.Log.Entries {
		if entry.Request.URL == "" {
			continue
		}

		// Strict URL validation.
		if _, err := url.Parse(entry.Request.URL); err != nil {
			urlError := NewValidationError(
				fmt.Sprintf("Invalid URL format: %s", err.Error()),
				fmt.Sprintf("log.entries[%d].request.url", i),
			)
			warnings = append(warnings, urlError)
			continue
		}

		// Check for common URL issues as well.
		if strings.Contains(entry.Request.URL, " ") {
			urlError := NewValidationError(
				fmt.Sprintf("URL contains spaces: %s", entry.Request.URL),
				fmt.Sprintf("log.entries[%d].request.url", i),
			)
			warnings = append(warnings, urlError)
		}

		if !strings.Contains(entry.Request.URL, "://") {
			urlError := NewValidationError(
				fmt.Sprintf("URL is missing a scheme: %s", entry.Request.URL),
				fmt.Sprintf("log.entries[%d].request.url", i),
			)
			warnings = append(warnings, urlError)
		}
	}

	return warnings
}

// performFullValidation runs full HAR validation and converts errors to warnings.
func performFullValidation(har *Har) []*HarError {
	if har == nil {
		return nil
	}

	validationErr := ValidateHarFile(har)
	if validationErr == nil {
		return nil
	}

	// All error paths in ValidateHarFile return *HarError.
	return validationErr.(*HarError).GetPartialErrors()
}

// appendWarnings adds new warnings to the existing list without duplicates.
func appendWarnings(existing []*HarError, newWarnings []*HarError) []*HarError {
	if len(newWarnings) == 0 {
		return existing
	}

	if existing == nil {
		return newWarnings
	}

	// Use a map to detect duplicates.
	warningMap := make(map[string]bool)
	for _, warn := range existing {
		key := warn.Field + ":" + warn.Message
		warningMap[key] = true
	}

	// Add warnings that are not duplicates.
	for _, warn := range newWarnings {
		key := warn.Field + ":" + warn.Message
		if !warningMap[key] {
			existing = append(existing, warn)
			warningMap[key] = true
		}
	}

	return existing
}

// ParseHarFileWithWarnings parses a HAR file and returns warnings.
func ParseHarFileWithWarnings(harFilePath string) (*Result, error) {
	harFileBytes, err := os.ReadFile(harFilePath)
	if err != nil {
		return nil, NewFileSystemError(fmt.Sprintf("Unable to read file '%s'", harFilePath), err)
	}

	return ParseHarWithWarnings(harFileBytes)
}
