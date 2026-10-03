package har

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Create test HAR files
func setupTestFiles(t *testing.T) {
	// Create the test directory
	testDataDir := "testdata"
	_ = os.MkdirAll(testDataDir, 0755)

	// Valid HAR file - minimal configuration
	minimalHar := Har{
		Log: Log{
			Version: "1.2",
			Creator: Creator{
				Name:    "Go-HAR Test",
				Version: "1.0",
			},
			Entries: []Entries{},
		},
	}
	writeHarFile(t, filepath.Join(testDataDir, "minimal.har"), minimalHar)

	// Valid HAR file - full configuration
	fullHar := createFullHar()
	writeHarFile(t, filepath.Join(testDataDir, "full.har"), fullHar)

	// Valid HAR file - version 1.1
	har11 := Har{
		Log: Log{
			Version: "1.1",
			Creator: Creator{
				Name:    "Go-HAR Test",
				Version: "1.0",
			},
			Entries: []Entries{},
		},
	}
	writeHarFile(t, filepath.Join(testDataDir, "v1.1.har"), har11)

	// Invalid HAR file - missing required fields
	invalidHar := map[string]interface{}{
		"log": map[string]interface{}{
			// Missing version
			"creator": map[string]interface{}{
				// Missing name
				"version": "1.0",
			},
			"entries": []interface{}{},
		},
	}
	writeJSONFile(t, filepath.Join(testDataDir, "invalid.har"), invalidHar)

	// Invalid HAR file - not JSON format
	writeTextFile(t, filepath.Join(testDataDir, "not_json.har"), "This is not a JSON file")

	// Invalid HAR file - invalid date format
	invalidDateHar := Har{
		Log: Log{
			Version: "1.2",
			Creator: Creator{
				Name:    "Go-HAR Test",
				Version: "1.0",
			},
			Entries: []Entries{
				{
					StartedDateTime: time.Now(), // Valid date
					Request: Request{
						Method: "GET",
						URL:    "https://example.com",
					},
					Response: Response{
						Status: 200,
					},
				},
			},
		},
	}
	writeHarFile(t, filepath.Join(testDataDir, "invalid_date.har"), invalidDateHar)

	// Large HAR file - for performance testing
	largeHar := createLargeHar(1000) // 1000 entries
	writeHarFile(t, filepath.Join(testDataDir, "large.har"), largeHar)
}

// Helper: write a HAR file
func writeHarFile(t *testing.T, filename string, har Har) {
	data, err := json.MarshalIndent(har, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(filename, data, 0644)
	require.NoError(t, err)
}

// Helper: write an arbitrary JSON file
func writeJSONFile(t *testing.T, filename string, data interface{}) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(filename, jsonData, 0644)
	require.NoError(t, err)
}

// Helper: write a text file
func writeTextFile(t *testing.T, filename string, content string) {
	err := os.WriteFile(filename, []byte(content), 0644)
	require.NoError(t, err)
}

// Helper: create a complete HAR object
func createFullHar() Har {
	now := time.Now()
	return Har{
		Log: Log{
			Version: "1.2",
			Creator: Creator{
				Name:    "Go-HAR Test",
				Version: "1.0",
			},
			Pages: []Pages{
				{
					StartedDateTime: now,
					ID:              "page_1",
					Title:           "Test Page",
					PageTimings: PageTimings{
						OnContentLoad: 150.5,
						OnLoad:        250.75,
						Comment:       "Page timing comment",
					},
				},
			},
			Entries: []Entries{
				{
					Pageref:         "page_1",
					StartedDateTime: now,
					Time:            350.25,
					Request: Request{
						Method:      "GET",
						URL:         "https://example.com/test",
						HTTPVersion: "HTTP/1.1",
						Headers: []Headers{
							{Name: "Accept", Value: "application/json"},
							{Name: "User-Agent", Value: "Go-HAR Test"},
						},
						QueryString: []QueryString{
							{Name: "id", Value: "12345"},
							{Name: "format", Value: "json"},
						},
						Cookies: []Cookie{
							{
								Name:     "session",
								Value:    "abc123",
								Path:     "/",
								Domain:   "example.com",
								Expires:  now.Add(24 * time.Hour),
								HTTPOnly: true,
								Secure:   true,
							},
						},
						HeadersSize: 150,
						BodySize:    0,
					},
					Response: Response{
						Status:      200,
						StatusText:  "OK",
						HTTPVersion: "HTTP/1.1",
						Headers: []Headers{
							{Name: "Content-Type", Value: "application/json"},
							{Name: "Cache-Control", Value: "no-cache"},
						},
						Cookies: []Cookie{},
						Content: Content{
							Size:     1024,
							MimeType: "application/json",
						},
						RedirectURL:  "",
						HeadersSize:  120,
						BodySize:     1024,
						TransferSize: 1144,
					},
					Cache: Cache{},
					Timings: Timings{
						Blocked: 12.5,
						DNS:     10.0,
						Connect: 25.5,
						Send:    5.5,
						Wait:    75.25,
						Receive: 15.75,
						Ssl:     20.0,
					},
					ServerIPAddress: "192.168.1.1",
					Connection:      "close",
				},
			},
		},
	}
}

// Helper: create a large HAR file
func createLargeHar(entriesCount int) Har {
	now := time.Now()
	har := Har{
		Log: Log{
			Version: "1.2",
			Creator: Creator{
				Name:    "Go-HAR Test",
				Version: "1.0",
			},
			Pages:   []Pages{},
			Entries: make([]Entries, entriesCount),
		},
	}

	for i := 0; i < entriesCount; i++ {
		har.Log.Entries[i] = Entries{
			StartedDateTime: now.Add(time.Duration(i) * time.Second),
			Time:            float64(100 + i),
			Request: Request{
				Method:      "GET",
				URL:         "https://example.com/api/item/" + string(rune(i)),
				HTTPVersion: "HTTP/1.1",
				Headers: []Headers{
					{Name: "Accept", Value: "application/json"},
				},
				HeadersSize: 100,
				BodySize:    0,
			},
			Response: Response{
				Status:      200,
				StatusText:  "OK",
				HTTPVersion: "HTTP/1.1",
				Headers: []Headers{
					{Name: "Content-Type", Value: "application/json"},
				},
				Content: Content{
					Size:     500,
					MimeType: "application/json",
				},
				HeadersSize: 80,
				BodySize:    500,
			},
			Timings: Timings{
				Blocked: 10.0,
				DNS:     5.0,
				Connect: 15.0,
				Send:    5.0,
				Wait:    50.0,
				Receive: 15.0,
			},
		}
	}

	return har
}

// TestParseHarBasic Test basic parsing functionality
func TestParseHarBasic(t *testing.T) {
	setupTestFiles(t)

	// Test parsing a minimal HAR file
	t.Run("ParseMinimalHar", func(t *testing.T) {
		data, err := os.ReadFile("testdata/minimal.har")
		require.NoError(t, err)

		har, err := ParseHar(data)
		require.NoError(t, err)
		assert.Equal(t, "1.2", har.Log.Version)
		assert.Equal(t, "Go-HAR Test", har.Log.Creator.Name)
		assert.Empty(t, har.Log.Entries)
	})

	// Test parsing a complete HAR file
	t.Run("ParseFullHar", func(t *testing.T) {
		data, err := os.ReadFile("testdata/full.har")
		require.NoError(t, err)

		har, err := ParseHar(data)
		require.NoError(t, err)
		assert.Equal(t, "1.2", har.Log.Version)
		assert.Equal(t, "Go-HAR Test", har.Log.Creator.Name)
		assert.Equal(t, "Test Page", har.Log.Pages[0].Title)
		assert.Len(t, har.Log.Pages, 1)
		assert.Len(t, har.Log.Entries, 1)

		// Verify entry details
		entry := har.Log.Entries[0]
		assert.Equal(t, "page_1", entry.Pageref)
		assert.Equal(t, "GET", entry.Request.Method)
		assert.Equal(t, "https://example.com/test", entry.Request.URL)
		assert.Equal(t, 200, entry.Response.Status)
		assert.Equal(t, "application/json", entry.Response.Content.MimeType)
	})

	// Test parsing an invalid HAR file
	t.Run("ParseInvalidHar", func(t *testing.T) {
		data, err := os.ReadFile("testdata/invalid.har")
		require.NoError(t, err)

		har, err := ParseHar(data)
		assert.Error(t, err)
		assert.Nil(t, har)

		// Verify the error type
		harErr, ok := err.(*HarError)
		if assert.True(t, ok, "Expected HarError type") {
			assert.Equal(t, ErrCodeValidation, harErr.Code)
			assert.True(t, harErr.HasPartialErrors())
		}
	})

	// Test a non-JSON file
	t.Run("ParseNonJsonFile", func(t *testing.T) {
		data, err := os.ReadFile("testdata/not_json.har")
		require.NoError(t, err)

		har, err := ParseHar(data)
		assert.Error(t, err)
		assert.Nil(t, har)
	})
}

// Test different HAR specification versions
func TestHarVersions(t *testing.T) {
	t.Run("ParseHarV1.1", func(t *testing.T) {
		data, err := os.ReadFile("testdata/version_11.har")
		require.NoError(t, err)

		har, err := ParseHar(data)
		require.NoError(t, err)
		assert.Equal(t, "1.1", har.Log.Version)
	})

	t.Run("ParseHarV1.3", func(t *testing.T) {
		data, err := os.ReadFile("testdata/version_13.har")
		require.NoError(t, err)

		har, err := ParseHar(data)
		require.NoError(t, err)
		assert.Equal(t, "1.3", har.Log.Version)
	})
}

// Test memory-optimized mode
func TestMemoryOptimizedParsing(t *testing.T) {
	t.Run("ParseWithMemoryOptimized", func(t *testing.T) {
		data, err := os.ReadFile("testdata/full.har")
		require.NoError(t, err)

		har, err := Parse(data, WithMemoryOptimized())
		require.NoError(t, err)
		assert.IsType(t, &OptimizedHar{}, har)
		assert.Equal(t, "1.2", har.GetVersion())
		assert.Equal(t, "Go-HAR Test", har.GetCreator().Name)
		assert.Len(t, har.GetEntries(), 1)
	})
}

// Test lazy-loading mode
func TestLazyLoadingParsing(t *testing.T) {
	t.Run("ParseWithLazyLoading", func(t *testing.T) {
		data, err := os.ReadFile("testdata/full.har")
		require.NoError(t, err)

		har, err := Parse(data, WithLazyLoading())
		require.NoError(t, err)
		assert.Equal(t, "1.2", har.GetVersion())

		// Verify lazy loading
		entries := har.GetEntries()
		assert.Len(t, entries, 1)

		content := entries[0].GetResponse().GetContent()
		assert.Equal(t, 1024, content.GetSize())
		assert.Equal(t, "application/json", content.GetMimeType())
	})
}

// Test skipping validation
func TestSkipValidation(t *testing.T) {
	t.Run("ParseInvalidHarWithSkipValidation", func(t *testing.T) {
		data, err := os.ReadFile("testdata/invalid.har")
		require.NoError(t, err)

		// Using the skip-validation option should allow parsing to succeed.
		har, err := Parse(data, WithSkipValidation())
		assert.NoError(t, err)
		assert.NotNil(t, har)
	})
}

// Test streaming parsing
func TestStreamingParsing(t *testing.T) {
	t.Run("EmptyEntries", func(t *testing.T) {
		data, err := os.ReadFile("testdata/minimal.har")
		require.NoError(t, err)

		parser, err := NewStreamingParser(data)
		require.NoError(t, err)

		count := 0
		for parser.Next() {
			count++
		}
		assert.NoError(t, parser.Err())
		assert.Equal(t, 0, count) // minimal.har doesn't have any entries
	})

	t.Run("WithEntries", func(t *testing.T) {
		data, err := os.ReadFile("testdata/example.har")
		require.NoError(t, err)

		parser, err := NewStreamingParser(data)
		require.NoError(t, err)

		count := 0
		for parser.Next() {
			entry := parser.Entry()
			assert.NotNil(t, entry)

			// Verify entry contents
			if count == 0 {
				assert.Equal(t, "GET", entry.Request.Method)
				assert.Equal(t, "https://example.com/test", entry.Request.URL)
				assert.Equal(t, 200, entry.Response.Status)
				assert.Equal(t, "text/plain", entry.Response.Content.MimeType)
				assert.Equal(t, 100, entry.Response.Content.Size)
			}
			count++
		}
		assert.NoError(t, parser.Err())
		assert.Equal(t, 1, count) // example.har has one entry
	})
}

// Test lenient mode
func TestLenientParsing(t *testing.T) {
	t.Run("ParseWithLenient", func(t *testing.T) {
		data, err := os.ReadFile("testdata/invalid.har")
		require.NoError(t, err)

		// Use lenient parsing options
		har, err := Parse(data, WithLenient())
		// Parsing should succeed, but with warnings.
		assert.NoError(t, err)
		assert.NotNil(t, har)
	})
}

// Test enhanced error handling
func TestEnhancedErrorHandling(t *testing.T) {
	t.Run("ParseWithWarnings", func(t *testing.T) {
		data, err := os.ReadFile("testdata/invalid_url.har")
		require.NoError(t, err)

		// Collect warnings
		result, err := ParseHarWithWarnings(data)
		assert.NoError(t, err)
		assert.NotNil(t, result.Har)

		// invalid_url.har contains an invalid URL, so a warning should be generated.
		// In lenient mode, these warnings do not cause parsing to fail.
		assert.NotEmpty(t, result.Warnings)

		// Verify that the warning contains a URL-related error.
		found := false
		for _, warning := range result.Warnings {
			if strings.Contains(warning.Field, "url") || strings.Contains(warning.Message, "URL") {
				found = true
				break
			}
		}
		assert.True(t, found, "Warning should contain a URL-related error.")
	})
}

// Test HAR conversion
func TestHarConversion(t *testing.T) {
	t.Run("StandardToOptimized", func(t *testing.T) {
		data, err := os.ReadFile("testdata/full.har")
		require.NoError(t, err)

		standard, err := ParseHar(data)
		require.NoError(t, err)

		optimized := ToOptimizedHar(standard)
		assert.Equal(t, standard.Log.Version, optimized.Log.Version)
		assert.Equal(t, standard.Log.Creator.Name, optimized.Log.Creator.Name)
		assert.Len(t, optimized.Log.Entries, len(standard.Log.Entries))
	})

	t.Run("OptimizedToStandard", func(t *testing.T) {
		data, err := os.ReadFile("testdata/full.har")
		require.NoError(t, err)

		optimized, err := Parse(data, WithMemoryOptimized())
		require.NoError(t, err)

		standard := optimized.ToStandard()
		assert.Equal(t, "1.2", standard.Log.Version)
		assert.Equal(t, "Go-HAR Test", standard.Log.Creator.Name)
		assert.Len(t, standard.Log.Entries, 1)
	})
}
