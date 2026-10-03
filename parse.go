package har

import (
	"fmt"
	"os"
)

// Parse parses HAR bytes using functional options.
//
// Parse is the main entry point for parsing HAR data and supports multiple parsing strategies and options.
// This function uses functional options to configure parsing flexibly.
//
// Example:
//
// // Standard parsing.
//	har, err := Parse(harBytes)
//
// // Use memory optimization.
//	har, err := Parse(harBytes, WithMemoryOptimized())
//
// // Combine multiple options.
//	har, err := Parse(harBytes, WithMemoryOptimized(), WithSkipValidation())
//
// Returns an object implementing HARProvider, providing a unified way to access different HAR implementations.
func Parse(harFileBytes []byte, opts ...Option) (HARProvider, error) {
	// Apply options.
	options := applyOptions(opts...)

	// Validate input.
	if err := validateInput(harFileBytes); err != nil {
		return nil, err
	}

	// Choose the appropriate parsing method based on the options.
	return parseWithStrategy(harFileBytes, options)
}

// validateInput checks whether the input data is valid.
func validateInput(harFileBytes []byte) error {
	// Check whether the input is empty.
	if len(harFileBytes) == 0 {
		return NewInvalidFormatError("Input is empty")
	}

	// Check whether the file is JSON.
	if !isJSONContent(harFileBytes) {
		return NewInvalidFormatError("Input is not valid JSON")
	}

	return nil
}

// parseWithStrategy selects an appropriate parsing strategy based on the options.
func parseWithStrategy(harFileBytes []byte, options options) (HARProvider, error) {
	// Streaming parsing requires special handling.
	if options.useStreaming {
		return nil, NewUnsupportedError("Streaming parsing cannot return a complete HAR object directly; use NewStreamingParser instead")
	}

	// Choose a parsing strategy based on the options.
	if options.useMemoryOptimized {
		// Memory-optimized parsing.
		return ParseHarOptimized(harFileBytes)
	} else if options.useLazyLoading {
		// Lazy parsing.
		return ParseHarWithLazyLoading(harFileBytes)
	} else {
		// Standard parsing.
		parseOptions := options.toParseOptions()
		return ParseHarWithOptions(harFileBytes, parseOptions)
	}
}

// ParseFile parses a HAR file using functional options.
//
// ParseFile is a convenience method for parsing HAR files and supports the same options as Parse.
// This function reads the file and passes its contents to Parse for processing.
//
// Example:
//
// // Standard parsing.
//	har, err := ParseFile("example.har")
//
// // Use a predefined option combination.
//	har, err := ParseFile("large.har", OptMemoryEfficient...)
func ParseFile(harFilePath string, opts ...Option) (HARProvider, error) {
	// Read the file.
	harFileBytes, err := os.ReadFile(harFilePath)
	if err != nil {
		return nil, NewFileSystemError(fmt.Sprintf("Unable to read file '%s'", harFilePath), err)
	}

	// Parse the file contents.
	har, err := Parse(harFileBytes, opts...)
	if err != nil {
		// Add the file path to the error context.
		if harErr, ok := err.(*HarError); ok {
			_ = harErr.WithMetadata("filePath", harFilePath)
		}
		return nil, err
	}

	return har, nil
}

// NewStreamingParser creates a new streaming parser.
//
// The streaming parser processes HAR entries one at a time, making it suitable for large HAR files and avoiding loading the entire file at once.
//
// Example:
//
//	iterator, err := NewStreamingParser(harBytes)
//	if err != nil {
//	    return err
//	}
//	for iterator.Next() {
//	    entry := iterator.Entry()
// // Process a single entry.
//	}
func NewStreamingParser(harFileBytes []byte, opts ...Option) (EntryIterator, error) {
	// Validate input.
	if err := validateInput(harFileBytes); err != nil {
		return nil, err
	}

	// Create a streaming parser.
	streamingHar, err := NewStreamingHarFromBytes(harFileBytes)
	if err != nil {
		return nil, err
	}
	return streamingHar.Entries(), nil
}

// NewStreamingParserFromFile creates a streaming parser from a file.
//
// This is a convenience method that creates a streaming parser from a file path, avoiding manual file reads.
func NewStreamingParserFromFile(harFilePath string, opts ...Option) (EntryIterator, error) {
	harFileBytes, err := os.ReadFile(harFilePath)
	if err != nil {
		return nil, NewFileSystemError(fmt.Sprintf("Unable to read file '%s'", harFilePath), err)
	}

	return NewStreamingParser(harFileBytes, opts...)
}
