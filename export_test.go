package har

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createTestHar creates a Har object for testing.
func createTestHar() *Har {
	h := NewHar()
	h.SetCreator("test-creator", "1.0")

	entry := h.AddEntry("GET", "https://example.com/api/v1/users?limit=10", "HTTP/1.1", "")
	entry.AddRequestHeader("Content-Type", "application/json")
	entry.AddRequestHeader("Accept-Encoding", "gzip, deflate")
	entry.AddRequestHeader("Authorization", "Bearer token123")
	entry.SetResponseStatus(200, "OK")
	entry.SetResponseContent(1024, "application/json")
	entry.AddResponseHeader("Content-Type", "application/json")

	entry2 := h.AddEntry("POST", "https://api.example.com/login", "HTTP/1.1", "")
	entry2.AddRequestHeader("Content-Type", "application/json")
	entry2.SetPostData("application/json", `{"username":"admin","password":"test's pass"}`)
	entry2.SetResponseStatus(200, "OK")
	entry2.SetResponseContent(256, "application/json")
	entry2.AddResponseHeader("Content-Type", "application/json")

	entry3 := h.AddEntry("DELETE", "https://api.example.com/items/42", "HTTP/1.1", "")
	entry3.AddRequestHeader("Authorization", "Bearer token123")
	entry3.SetResponseStatus(204, "No Content")

	return h
}

// ---------------------------------------------------------------------------
// ToCurl tests.
// ---------------------------------------------------------------------------

func TestExportToCurlOnHar(t *testing.T) {
	h := createTestHar()
	result := h.ToCurl()

	if result == "" {
		t.Fatal("ToCurl() should not return an empty string")
	}

	// Should contain multiple cURL commands separated by double newlines.
	parts := strings.Split(result, "\n\n")
	if len(parts) != 3 {
		t.Fatalf("Expected 3 cURL commands, got %d.", len(parts))
	}
}

func TestExportToCurlOnHarNil(t *testing.T) {
	var h *Har
	result := h.ToCurl()
	if result != "" {
		t.Fatalf("nil HAR should return an empty string; got: %s", result)
	}
}

func TestExportToCurlOnHarEmpty(t *testing.T) {
	h := &Har{}
	result := h.ToCurl()
	if result != "" {
		t.Fatalf("empty HAR should return an empty string; got: %s", result)
	}
}

func TestExportToCurlOnEntries(t *testing.T) {
	h := createTestHar()
	entry := &h.Log.Entries[0]
	result := entry.ToCurl()

	if result == "" {
		t.Fatal("Entries.ToCurl() should not return an empty string")
	}
	if !strings.HasPrefix(result, "curl") {
		t.Errorf("cURL command should start with 'curl'; got: %s", result[:10])
	}
}

func TestExportToCurlOnEntriesNil(t *testing.T) {
	var e *Entries
	result := e.ToCurl()
	if result != "" {
		t.Fatalf("nil Entries should return an empty string; got: %s", result)
	}
}

func TestExportCurlMethod(t *testing.T) {
	h := NewHar()
	entry := h.AddEntry("POST", "https://example.com/api", "HTTP/1.1", "")
	entry.AddRequestHeader("Content-Type", "application/json")
	entry.SetPostData("application/json", `{"key":"value"}`)
	entry.SetResponseStatus(200, "OK")

	result := entry.ToCurl()

	if !strings.Contains(result, "-X POST") {
		t.Errorf("POST request should include -X POST; got: %s", result)
	}
	if !strings.Contains(result, "--data") {
		t.Errorf("Request with POST data should include --data; got: %s", result)
	}
}

func TestExportCurlGETNoMethod(t *testing.T) {
	h := NewHar()
	entry := h.AddEntry("GET", "https://example.com/api", "HTTP/1.1", "")
	entry.SetResponseStatus(200, "OK")

	result := entry.ToCurl()

	if strings.Contains(result, "-X GET") {
		t.Errorf("GET request should not include -X GET; got: %s", result)
	}
}

func TestExportCurlHeaders(t *testing.T) {
	h := NewHar()
	entry := h.AddEntry("GET", "https://example.com/api", "HTTP/1.1", "")
	entry.AddRequestHeader("X-Custom", "value123")
	entry.AddRequestHeader("Host", "example.com") // should be skipped
	entry.SetResponseStatus(200, "OK")

	result := entry.ToCurl()

	if !strings.Contains(result, "-H 'X-Custom: value123'") {
		t.Errorf("Should include the custom request header; got: %s", result)
	}
	if strings.Contains(result, "Host:") {
		t.Errorf("Should not include the Host header; got: %s", result)
	}
}

func TestExportCurlCompressed(t *testing.T) {
	h := NewHar()
	entry := h.AddEntry("GET", "https://example.com/api", "HTTP/1.1", "")
	entry.AddRequestHeader("Accept-Encoding", "gzip, deflate")
	entry.SetResponseStatus(200, "OK")

	result := entry.ToCurl()

	if !strings.Contains(result, "--compressed") {
		t.Errorf("Should add --compressed when Accept-Encoding contains gzip/deflate; got: %s", result)
	}
}

func TestExportCurlNoCompressed(t *testing.T) {
	h := NewHar()
	entry := h.AddEntry("GET", "https://example.com/api", "HTTP/1.1", "")
	entry.AddRequestHeader("Accept-Encoding", "br")
	entry.SetResponseStatus(200, "OK")

	result := entry.ToCurl()

	if strings.Contains(result, "--compressed") {
		t.Errorf("Should not add --compressed when Accept-Encoding does not contain gzip/deflate; got: %s", result)
	}
}

func TestExportCurlSingleQuoteEscaping(t *testing.T) {
	h := NewHar()
	entry := h.AddEntry("POST", "https://example.com/api", "HTTP/1.1", "")
	entry.SetPostData("application/json", `it's a test`)
	entry.SetResponseStatus(200, "OK")

	result := entry.ToCurl()

	if !strings.Contains(result, `it'\''s a test`) {
		t.Errorf("Single quotes should be escaped correctly; got: %s", result)
	}
}

func TestExportCurlSSLVerifySkip(t *testing.T) {
	h := NewHar()
	entry := h.AddEntry("GET", "https://self-signed.example.com/api", "HTTP/1.1", "")
	entry.Response.Error = "SSL certificate problem"
	entry.SetResponseStatus(0, "")

	result := entry.ToCurl()

	if !strings.Contains(result, "-k") {
		t.Errorf("Should add -k when there is an SSL error; got: %s", result)
	}
}

func TestExportCurlURLQuoted(t *testing.T) {
	h := NewHar()
	entry := h.AddEntry("GET", "https://example.com/api?key=value", "HTTP/1.1", "")
	entry.SetResponseStatus(200, "OK")

	result := entry.ToCurl()

	if !strings.Contains(result, "'https://example.com/api?key=value'") {
		t.Errorf("URL should be enclosed in single quotes; got: %s", result)
	}
}

// ---------------------------------------------------------------------------
// ToWget tests.
// ---------------------------------------------------------------------------

func TestExportToWgetOnHar(t *testing.T) {
	h := createTestHar()
	result := h.ToWget()

	if result == "" {
		t.Fatal("ToWget() should not return an empty string")
	}

	parts := strings.Split(result, "\n\n")
	if len(parts) != 3 {
		t.Fatalf("Expected 3 wget commands, got %d.", len(parts))
	}
}

func TestExportToWgetOnHarNil(t *testing.T) {
	var h *Har
	result := h.ToWget()
	if result != "" {
		t.Fatalf("nil HAR should return an empty string; got: %s", result)
	}
}

func TestExportToWgetOnEntries(t *testing.T) {
	h := createTestHar()
	entry := &h.Log.Entries[0]
	result := entry.ToWget()

	if result == "" {
		t.Fatal("Entries.ToWget() should not return an empty string")
	}
	if !strings.HasPrefix(result, "wget") {
		t.Errorf("wget command should start with 'wget'; got: %s", result[:10])
	}
}

func TestExportToWgetOnEntriesNil(t *testing.T) {
	var e *Entries
	result := e.ToWget()
	if result != "" {
		t.Fatalf("nil Entries should return an empty string; got: %s", result)
	}
}

func TestExportWgetMethod(t *testing.T) {
	h := NewHar()
	entry := h.AddEntry("DELETE", "https://example.com/api/items/1", "HTTP/1.1", "")
	entry.SetResponseStatus(204, "No Content")

	result := entry.ToWget()

	if !strings.Contains(result, "--method=DELETE") {
		t.Errorf("DELETE request should include --method=DELETE; got: %s", result)
	}
}

func TestExportWgetGETNoMethod(t *testing.T) {
	h := NewHar()
	entry := h.AddEntry("GET", "https://example.com/api", "HTTP/1.1", "")
	entry.SetResponseStatus(200, "OK")

	result := entry.ToWget()

	if strings.Contains(result, "--method=") {
		t.Errorf("GET request should not include --method; got: %s", result)
	}
}

func TestExportWgetPostData(t *testing.T) {
	h := NewHar()
	entry := h.AddEntry("POST", "https://example.com/api", "HTTP/1.1", "")
	entry.SetPostData("application/json", `{"name":"test"}`)
	entry.SetResponseStatus(200, "OK")

	result := entry.ToWget()

	if !strings.Contains(result, "--post-data=") {
		t.Errorf("Request with POST data should include --post-data; got: %s", result)
	}
}

func TestExportWgetHTTPSNoCheckCert(t *testing.T) {
	h := NewHar()
	entry := h.AddEntry("GET", "https://example.com/api", "HTTP/1.1", "")
	entry.SetResponseStatus(200, "OK")

	result := entry.ToWget()

	if !strings.Contains(result, "--no-check-certificate") {
		t.Errorf("HTTPS request should include --no-check-certificate; got: %s", result)
	}
}

func TestExportWgetHTTPNoCertFlag(t *testing.T) {
	h := NewHar()
	entry := h.AddEntry("GET", "http://example.com/api", "HTTP/1.1", "")
	entry.SetResponseStatus(200, "OK")

	result := entry.ToWget()

	if strings.Contains(result, "--no-check-certificate") {
		t.Errorf("HTTP request should not include --no-check-certificate; got: %s", result)
	}
}

func TestExportWgetHeaderSkipHost(t *testing.T) {
	h := NewHar()
	entry := h.AddEntry("GET", "https://example.com/api", "HTTP/1.1", "")
	entry.AddRequestHeader("Host", "example.com")
	entry.AddRequestHeader("X-Custom", "test")
	entry.SetResponseStatus(200, "OK")

	result := entry.ToWget()

	if strings.Contains(result, "Host:") {
		t.Errorf("wget should not include the Host header; got: %s", result)
	}
	if !strings.Contains(result, "--header='X-Custom: test'") {
		t.Errorf("Should include the custom request header; got: %s", result)
	}
}

// ---------------------------------------------------------------------------
// ToPythonRequests tests.
// ---------------------------------------------------------------------------

func TestExportToPythonRequestsOnHar(t *testing.T) {
	h := createTestHar()
	result := h.ToPythonRequests()

	if result == "" {
		t.Fatal("ToPythonRequests() should not return an empty string")
	}
	if !strings.Contains(result, "import requests") {
		t.Error("Python code should include 'import requests'")
	}
	// Should contain 3 requests.
	count := strings.Count(result, "response = requests.")
	if count != 3 {
		t.Errorf("Expected 3 requests calls, got %d.", count)
	}
}

func TestExportToPythonRequestsOnHarNil(t *testing.T) {
	var h *Har
	result := h.ToPythonRequests()
	if result != "" {
		t.Fatalf("nil HAR should return an empty string; got: %s", result)
	}
}

func TestExportToPythonRequestsOnEntries(t *testing.T) {
	h := createTestHar()
	entry := &h.Log.Entries[0]
	result := entry.ToPythonRequests()

	if result == "" {
		t.Fatal("Entries.ToPythonRequests() should not return an empty string")
	}
	if !strings.Contains(result, "requests.get") {
		t.Errorf("GET request should use requests.get; got: %s", result)
	}
}

func TestExportToPythonRequestsOnEntriesNil(t *testing.T) {
	var e *Entries
	result := e.ToPythonRequests()
	if result != "" {
		t.Fatalf("nil Entries should return an empty string; got: %s", result)
	}
}

func TestExportPythonRequestMethod(t *testing.T) {
	tests := []struct {
		method         string
		expectedMethod string
	}{
		{"GET", "requests.get"},
		{"POST", "requests.post"},
		{"PUT", "requests.put"},
		{"DELETE", "requests.delete"},
		{"PATCH", "requests.patch"},
		{"HEAD", "requests.head"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			h := NewHar()
			entry := h.AddEntry(tt.method, "https://example.com/api", "HTTP/1.1", "")
			entry.SetResponseStatus(200, "OK")
			result := entry.ToPythonRequests()
			if !strings.Contains(result, tt.expectedMethod) {
				t.Errorf("Method %s should generate %s; got: %s", tt.method, tt.expectedMethod, result)
			}
		})
	}
}

func TestExportPythonHeaders(t *testing.T) {
	h := NewHar()
	entry := h.AddEntry("GET", "https://example.com/api", "HTTP/1.1", "")
	entry.AddRequestHeader("Authorization", "Bearer token")
	entry.SetResponseStatus(200, "OK")

	result := entry.ToPythonRequests()

	if !strings.Contains(result, "headers = {") {
		t.Errorf("Should generate a headers dictionary when request headers are present; got: %s", result)
	}
	if !strings.Contains(result, "'Authorization': 'Bearer token'") {
		t.Errorf("Should include the Authorization header; got: %s", result)
	}
}

func TestExportPythonPostData(t *testing.T) {
	h := NewHar()
	entry := h.AddEntry("POST", "https://example.com/api", "HTTP/1.1", "")
	entry.SetPostData("application/json", `{"key":"value"}`)
	entry.SetResponseStatus(200, "OK")

	result := entry.ToPythonRequests()

	if !strings.Contains(result, "data=") {
		t.Errorf("POST request should include the data parameter; got: %s", result)
	}
}

func TestExportPythonPrint(t *testing.T) {
	h := NewHar()
	entry := h.AddEntry("GET", "https://example.com/api", "HTTP/1.1", "")
	entry.SetResponseStatus(200, "OK")

	result := entry.ToPythonRequests()

	if !strings.Contains(result, "print(response.status_code)") {
		t.Errorf("Should print status_code; got: %s", result)
	}
	if !strings.Contains(result, "print(response.text)") {
		t.Errorf("Should print response.text; got: %s", result)
	}
}

func TestExportPythonStringEscaping(t *testing.T) {
	h := NewHar()
	entry := h.AddEntry("POST", "https://example.com/api", "HTTP/1.1", "")
	entry.SetPostData("application/json", "it's a test\nwith newline")
	entry.SetResponseStatus(200, "OK")

	result := entry.ToPythonRequests()

	if !strings.Contains(result, `it\'s a test`) {
		t.Errorf("Single quotes should be escaped; got: %s", result)
	}
	if !strings.Contains(result, `\n`) {
		t.Errorf("Newlines should be escaped; got: %s", result)
	}
}

// ---------------------------------------------------------------------------
// ToPostmanCollection tests.
// ---------------------------------------------------------------------------

func TestExportToPostmanCollection(t *testing.T) {
	h := createTestHar()
	data, err := h.ToPostmanCollection()
	if err != nil {
		t.Fatalf("ToPostmanCollection() returned an error: %v", err)
	}

	// Verify that the JSON is valid.
	var collection PostmanCollection
	if err := json.Unmarshal(data, &collection); err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}

	// Verify info.
	if collection.Info.Name != "HAR Export" {
		t.Errorf("Name should be 'HAR Export'; got: %s", collection.Info.Name)
	}
	expectedSchema := "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
	if collection.Info.Schema != expectedSchema {
		t.Errorf("Schema is incorrect; got: %s", collection.Info.Schema)
	}

	// Verify the item count.
	if len(collection.Item) != 3 {
		t.Errorf("Expected 3 items, got: %d", len(collection.Item))
	}
}

func TestExportToPostmanCollectionNil(t *testing.T) {
	var h *Har
	_, err := h.ToPostmanCollection()
	assertHarErrorCode(t, err, ErrCodeInvalidFormat)
}

func TestExportPostmanCollectionMethod(t *testing.T) {
	h := NewHar()
	entry := h.AddEntry("POST", "https://example.com/api/v1/users", "HTTP/1.1", "")
	entry.SetPostData("application/json", `{"name":"test"}`)
	entry.AddRequestHeader("Content-Type", "application/json")
	entry.SetResponseStatus(200, "OK")

	data, err := h.ToPostmanCollection()
	if err != nil {
		t.Fatalf("ToPostmanCollection() returned an error: %v", err)
	}

	var collection PostmanCollection
	if err := json.Unmarshal(data, &collection); err != nil {
		t.Fatalf("JSON parsing failed: %v", err)
	}

	item := collection.Item[0]
	if item.Request.Method != "POST" {
		t.Errorf("Method should be POST; got: %s", item.Request.Method)
	}

	// Verify the URL structure.
	if item.Request.URL.Protocol != "https" {
		t.Errorf("Protocol should be https; got: %s", item.Request.URL.Protocol)
	}

	// Verify request headers.
	if len(item.Request.Header) == 0 {
		t.Error("Should have request headers.")
	}

	// Verify the request body.
	if item.Request.Body == nil {
		t.Fatal("Should have a request body.")
	}
	if item.Request.Body.Mode != "raw" {
		t.Errorf("Body mode should be raw; got: %s", item.Request.Body.Mode)
	}
}

func TestExportPostmanCollectionURLParsing(t *testing.T) {
	h := NewHar()
	entry := h.AddEntry("GET", "https://api.example.com/v1/users?limit=10&offset=0", "HTTP/1.1", "")
	entry.SetResponseStatus(200, "OK")

	data, err := h.ToPostmanCollection()
	if err != nil {
		t.Fatalf("ToPostmanCollection() returned an error: %v", err)
	}

	var collection PostmanCollection
	if err := json.Unmarshal(data, &collection); err != nil {
		t.Fatalf("JSON parsing failed: %v", err)
	}

	item := collection.Item[0]
	pmURL := item.Request.URL

	// Verify the Host.
	if len(pmURL.Host) == 0 {
		t.Error("Host should not be empty.")
	}

	// Verify the Path.
	if len(pmURL.Path) < 2 {
		t.Errorf("Path should have at least 2 segments; got: %v", pmURL.Path)
	}

	// Verify the Query.
	if len(pmURL.Query) < 2 {
		t.Errorf("Query should have at least 2 parameters; got: %v", pmURL.Query)
	}
}

// ---------------------------------------------------------------------------
// SaveAsPostmanCollection tests.
// ---------------------------------------------------------------------------

func TestExportSaveAsPostmanCollection(t *testing.T) {
	h := createTestHar()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "collection.json")

	if err := h.SaveAsPostmanCollection(filePath); err != nil {
		t.Fatalf("SaveAsPostmanCollection()  returned error: %v", err)
	}

	// Verify that the file exists.
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("File was not created.")
	}

	// Verify that the file contains valid Postman Collection JSON.
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read the file: %v", err)
	}

	var collection PostmanCollection
	if err := json.Unmarshal(data, &collection); err != nil {
		t.Fatalf("File content is not valid JSON: %v", err)
	}

	if collection.Info.Name != "HAR Export" {
		t.Errorf("Name should be 'HAR Export'; got: %s", collection.Info.Name)
	}
}

func TestExportSaveAsPostmanCollectionNil(t *testing.T) {
	var h *Har
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "collection.json")

	err := h.SaveAsPostmanCollection(filePath)
	assertHarErrorCode(t, err, ErrCodeInvalidFormat)
}

// ---------------------------------------------------------------------------
// ToXML tests.
// ---------------------------------------------------------------------------

func TestExportToXML(t *testing.T) {
	h := createTestHar()
	result, err := h.ToXML()
	if err != nil {
		t.Fatalf("ToXML() returned an error: %v", err)
	}

	if result == "" {
		t.Fatal("ToXML() should not return an empty string")
	}

	// Verify that the XML declaration is present.
	if !strings.Contains(result, `<?xml`) {
		t.Error("XML should contain an XML declaration.")
	}

	// Verify the root element.
	if !strings.Contains(result, "<har>") {
		t.Error("XML should contain the <har> root element.")
	}
	if !strings.Contains(result, "</har>") {
		t.Error("XML should contain the </har> closing tag.")
	}

	// Verify that it can be parsed correctly.
	var harXML HARXML
	if err := xml.Unmarshal([]byte(result), &harXML); err != nil {
		t.Fatalf("XML parsing failed: %v", err)
	}

	if harXML.Log.Version != "1.2" {
		t.Errorf("Version should be 1.2; got: %s", harXML.Log.Version)
	}

	if len(harXML.Log.Entries) != 3 {
		t.Errorf("Expected 3 entries, got: %d", len(harXML.Log.Entries))
	}
}

func TestExportToXMLNil(t *testing.T) {
	var h *Har
	result, err := h.ToXML()
	assertHarErrorCode(t, err, ErrCodeInvalidFormat)
	if result != "" {
		t.Errorf("nil HAR should return an empty string; got: %s", result)
	}
}

func TestExportToXMLContent(t *testing.T) {
	h := NewHar()
	entry := h.AddEntry("POST", "https://example.com/api", "HTTP/1.1", "")
	entry.AddRequestHeader("Content-Type", "application/json")
	entry.SetPostData("application/json", `{"key":"value"}`)
	entry.SetResponseStatus(200, "OK")
	entry.SetResponseContent(42, "application/json")

	result, err := h.ToXML()
	if err != nil {
		t.Fatalf("ToXML() returned an error: %v", err)
	}

	// Verify the method.
	if !strings.Contains(result, "<method>POST</method>") {
		t.Errorf("XML should contain the POST method; got: %s", result)
	}

	// Verify the URL.
	if !strings.Contains(result, "<url>https://example.com/api</url>") {
		t.Errorf("XML should contain the URL; got: %s", result)
	}

	// Verify request headers.
	if !strings.Contains(result, "<name>Content-Type</name>") {
		t.Errorf("XML should contain the request header name; got: %s", result)
	}

	// Verify POST data.
	if !strings.Contains(result, "<postData>") {
		t.Errorf("XML should contain the postData element; got: %s", result)
	}
}

// ---------------------------------------------------------------------------
// SaveAsXML tests.
// ---------------------------------------------------------------------------

func TestExportSaveAsXML(t *testing.T) {
	h := createTestHar()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "output.xml")

	if err := h.SaveAsXML(filePath); err != nil {
		t.Fatalf("SaveAsXML()  returned error: %v", err)
	}

	// Verify that the file exists.
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("File was not created.")
	}

	// Verify the file contents.
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read the file: %v", err)
	}

	if !strings.Contains(string(data), "<?xml") {
		t.Error("File contents should include the XML declaration.")
	}
}

func TestExportSaveAsXMLNil(t *testing.T) {
	var h *Har
	filePath := filepath.Join(t.TempDir(), "output.xml")

	err := h.SaveAsXML(filePath)
	assertHarErrorCode(t, err, ErrCodeInvalidFormat)

	if _, statErr := os.Stat(filePath); !os.IsNotExist(statErr) {
		t.Fatalf("expected nil HAR not to create file, stat err: %v", statErr)
	}
}

// ---------------------------------------------------------------------------
// FormatJSON constant test.
// ---------------------------------------------------------------------------

func TestExportFormatJSON(t *testing.T) {
	if FormatJSON != "json" {
		t.Errorf("FormatJSON should be 'json'; got: %s", FormatJSON)
	}
}

// ---------------------------------------------------------------------------
// Helper function tests.
// ---------------------------------------------------------------------------

func TestExportEscapeSingleQuotes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"it's", "it'\\''s"},
		{"don't stop", "don'\\''t stop"},
		{"no quotes", "no quotes"},
		{"", ""},
	}

	for _, tt := range tests {
		result := escapeSingleQuotes(tt.input)
		if result != tt.expected {
			t.Errorf("escapeSingleQuotes(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestExportEscapePythonString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"it's", `it\'s`},
		{`back\slash`, `back\\slash`},
		{"line\nbreak", "line\\nbreak"},
		{"tab\there", "tab\\there"},
		{"", ""},
	}

	for _, tt := range tests {
		result := escapePythonString(tt.input)
		if result != tt.expected {
			t.Errorf("escapePythonString(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestExportHasAcceptEncoding(t *testing.T) {
	tests := []struct {
		name     string
		headers  []Headers
		expected bool
	}{
		{
			"gzip",
			[]Headers{{Name: "Accept-Encoding", Value: "gzip"}},
			true,
		},
		{
			"deflate",
			[]Headers{{Name: "Accept-Encoding", Value: "deflate"}},
			true,
		},
		{
			"mixed",
			[]Headers{{Name: "Accept-Encoding", Value: "gzip, deflate, br"}},
			true,
		},
		{
			"br only",
			[]Headers{{Name: "Accept-Encoding", Value: "br"}},
			false,
		},
		{
			"no header",
			[]Headers{},
			false,
		},
		{
			"case insensitive",
			[]Headers{{Name: "accept-encoding", Value: "GZIP"}},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &Entries{
				Request: Request{
					Headers: tt.headers,
				},
			}
			result := hasAcceptEncoding(entry)
			if result != tt.expected {
				t.Errorf("hasAcceptEncoding() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Integration tests.
// ---------------------------------------------------------------------------

func TestExportBuildPostmanURL_NilParsedURL(t *testing.T) {
	// When parsedURL is nil, buildPostmanURL should return only the Raw field
	pmURL := buildPostmanURL("https://example.com/path?q=1", nil)
	if pmURL.Raw != "https://example.com/path?q=1" {
		t.Errorf("expected Raw to be set, got %s", pmURL.Raw)
	}
	if pmURL.Protocol != "" {
		t.Errorf("expected Protocol to be empty when parsedURL is nil, got %s", pmURL.Protocol)
	}
	if len(pmURL.Host) != 0 {
		t.Errorf("expected Host to be empty when parsedURL is nil, got %v", pmURL.Host)
	}
	if len(pmURL.Path) != 0 {
		t.Errorf("expected Path to be empty when parsedURL is nil, got %v", pmURL.Path)
	}
	if len(pmURL.Query) != 0 {
		t.Errorf("expected Query to be empty when parsedURL is nil, got %v", pmURL.Query)
	}
}

func TestExportBuildPostmanURL_EmptyHost(t *testing.T) {
	// When the host has no dots, Host slice should still contain it
	pmURL := buildPostmanURL("https://localhost/path", nil)
	if pmURL.Raw != "https://localhost/path" {
		t.Errorf("expected Raw to be set, got %s", pmURL.Raw)
	}
	// With nil parsedURL, Host should be empty
	if len(pmURL.Host) != 0 {
		t.Errorf("expected Host to be empty when parsedURL is nil, got %v", pmURL.Host)
	}
}

func TestExportBuildPostmanURL_EmptyPath(t *testing.T) {
	// URL with no path
	h := NewHar()
	entry := h.AddEntry("GET", "https://example.com", "HTTP/1.1", "")
	entry.SetResponseStatus(200, "OK")

	data, err := h.ToPostmanCollection()
	if err != nil {
		t.Fatalf("ToPostmanCollection() returned error: %v", err)
	}

	var collection PostmanCollection
	if err := json.Unmarshal(data, &collection); err != nil {
		t.Fatalf("JSON parsing failed: %v", err)
	}

	pmURL := collection.Item[0].Request.URL
	if len(pmURL.Path) != 0 {
		t.Errorf("expected empty Path for URL with no path, got %v", pmURL.Path)
	}
}

func TestExportToXML_WithResponseContent(t *testing.T) {
	h := NewHar()
	entry := h.AddEntry("GET", "https://example.com/api", "HTTP/1.1", "")
	entry.AddRequestHeader("X-Test", "value")
	entry.AddResponseHeader("X-Response", "resp-value")
	entry.SetResponseStatus(200, "OK")
	entry.SetResponseContent(42, "application/json")

	result, err := h.ToXML()
	if err != nil {
		t.Fatalf("ToXML() returned error: %v", err)
	}

	if !strings.Contains(result, "<name>X-Test</name>") {
		t.Errorf("XML should contain request header, got: %s", result)
	}
	if !strings.Contains(result, "<name>X-Response</name>") {
		t.Errorf("XML should contain response header, got: %s", result)
	}
	if !strings.Contains(result, "<mimeType>application/json</mimeType>") {
		t.Errorf("XML should contain content mimeType, got: %s", result)
	}
}

func TestExportSaveAsXML_ToXMLError(t *testing.T) {
	// Test SaveAsXML when ToXML returns an error.
	// ToXML only errors on xml.MarshalIndent which is hard to trigger with normal data.
	// Instead, we can test the writeToFile error path by providing an invalid path.
	h := createTestHar()

	// Write to an invalid path (directory that doesn't exist)
	err := h.SaveAsXML(filepath.Join(t.TempDir(), "missing", "output.xml"))
	assertHarErrorCode(t, err, ErrCodeFileSystem)
}

func TestExportSaveAsXML_NilHarSave(t *testing.T) {
	var h *Har
	filePath := filepath.Join(t.TempDir(), "nil_output.xml")

	err := h.SaveAsXML(filePath)
	assertHarErrorCode(t, err, ErrCodeInvalidFormat)

	if _, statErr := os.Stat(filePath); !os.IsNotExist(statErr) {
		t.Fatalf("expected nil HAR not to create file, stat err: %v", statErr)
	}
}

func TestExportToXML_NoPostData(t *testing.T) {
	// Entry without PostData should not produce a <postData> element
	h := NewHar()
	entry := h.AddEntry("GET", "https://example.com/api", "HTTP/1.1", "")
	entry.SetResponseStatus(200, "OK")

	result, err := h.ToXML()
	if err != nil {
		t.Fatalf("ToXML() returned error: %v", err)
	}

	if strings.Contains(result, "<postData>") {
		t.Errorf("XML should not contain postData for GET request without PostData, got: %s", result)
	}
}

func TestExportToXML_WithPostDataNilText(t *testing.T) {
	// Entry with PostData that has empty Text should still produce postData element
	h := NewHar()
	entry := h.AddEntry("POST", "https://example.com/api", "HTTP/1.1", "")
	entry.Request.PostData = &PostData{MimeType: "application/json", Text: ""}
	entry.SetResponseStatus(200, "OK")

	result, err := h.ToXML()
	if err != nil {
		t.Fatalf("ToXML() returned error: %v", err)
	}

	if !strings.Contains(result, "<postData>") {
		t.Errorf("XML should contain postData element when PostData is non-nil, got: %s", result)
	}
}

func TestExportBuildPostmanURL_WithQueryParams(t *testing.T) {
	// Test that query parameters are properly extracted in Postman URL
	h := NewHar()
	entry := h.AddEntry("GET", "https://example.com/api?key1=val1&key2=val2", "HTTP/1.1", "")
	entry.SetResponseStatus(200, "OK")

	data, err := h.ToPostmanCollection()
	if err != nil {
		t.Fatalf("ToPostmanCollection() returned error: %v", err)
	}

	var collection PostmanCollection
	if err := json.Unmarshal(data, &collection); err != nil {
		t.Fatalf("JSON parsing failed: %v", err)
	}

	pmURL := collection.Item[0].Request.URL
	if len(pmURL.Query) < 2 {
		t.Errorf("expected at least 2 query parameters, got %d", len(pmURL.Query))
	}
}

func TestExportBuildPostmanURL_HostSplit(t *testing.T) {
	// Test that host is properly split by dots
	h := NewHar()
	entry := h.AddEntry("GET", "https://sub.api.example.com/path", "HTTP/1.1", "")
	entry.SetResponseStatus(200, "OK")

	data, err := h.ToPostmanCollection()
	if err != nil {
		t.Fatalf("ToPostmanCollection() returned error: %v", err)
	}

	var collection PostmanCollection
	if err := json.Unmarshal(data, &collection); err != nil {
		t.Fatalf("JSON parsing failed: %v", err)
	}

	pmURL := collection.Item[0].Request.URL
	if len(pmURL.Host) < 3 {
		t.Errorf("expected host to be split into at least 3 parts, got %v", pmURL.Host)
	}
}

func TestExportEntryToPostmanItem_UnparseableURL(t *testing.T) {
	// Test entryToPostmanItem with an unparseable URL (url.Parse fails)
	h := NewHar()
	// Use a URL with control character that url.Parse may still handle,
	// or just use a simple URL and verify the name fallback
	entry := h.AddEntry("GET", "://invalid-url", "HTTP/1.1", "")
	entry.SetResponseStatus(200, "OK")

	data, err := h.ToPostmanCollection()
	if err != nil {
		t.Fatalf("ToPostmanCollection() returned error: %v", err)
	}

	var collection PostmanCollection
	if err := json.Unmarshal(data, &collection); err != nil {
		t.Fatalf("JSON parsing failed: %v", err)
	}

	// With an unparseable URL, name should fall back to the raw URL
	if len(collection.Item) == 0 {
		t.Fatal("expected at least one item")
	}
}

func TestExportEntryToPostmanItem_NoHeaders(t *testing.T) {
	// Entry with no request headers
	h := NewHar()
	entry := h.AddEntry("GET", "https://example.com/api", "HTTP/1.1", "")
	entry.SetResponseStatus(200, "OK")

	data, err := h.ToPostmanCollection()
	if err != nil {
		t.Fatalf("ToPostmanCollection() returned error: %v", err)
	}

	var collection PostmanCollection
	if err := json.Unmarshal(data, &collection); err != nil {
		t.Fatalf("JSON parsing failed: %v", err)
	}

	if len(collection.Item[0].Request.Header) != 0 {
		t.Errorf("expected no headers, got %d", len(collection.Item[0].Request.Header))
	}
	if collection.Item[0].Request.Body != nil {
		t.Error("expected nil body for GET request")
	}
}

func TestExportToXML_EmptyHar(t *testing.T) {
	// Test ToXML with empty entries
	h := &Har{}

	result, err := h.ToXML()
	if err != nil {
		t.Fatalf("ToXML() returned error: %v", err)
	}

	if !strings.Contains(result, "<har>") {
		t.Error("XML should contain <har> root element")
	}
}

// ---------------------------------------------------------------------------
// Helper function tests (additional)
// ---------------------------------------------------------------------------

func TestExportHasAcceptEncoding_EmptyHeaders(t *testing.T) {
	entry := &Entries{
		Request: Request{
			Headers: nil,
		},
	}
	result := hasAcceptEncoding(entry)
	if result {
		t.Error("expected false for nil headers")
	}
}

func TestExportCurlHTTPNoSSLFlag(t *testing.T) {
	// HTTP request with error should not add -k flag
	h := NewHar()
	entry := h.AddEntry("GET", "http://example.com/api", "HTTP/1.1", "")
	entry.Response.Error = "connection refused"
	entry.SetResponseStatus(0, "")

	result := entry.ToCurl()

	if strings.Contains(result, "-k") {
		t.Errorf("HTTP request should not have -k flag even with error, got: %s", result)
	}
}

func TestExportCurlHTTPSNoError(t *testing.T) {
	// HTTPS request without error should not add -k flag
	h := NewHar()
	entry := h.AddEntry("GET", "https://example.com/api", "HTTP/1.1", "")
	entry.SetResponseStatus(200, "OK")

	result := entry.ToCurl()

	if strings.Contains(result, "-k") {
		t.Errorf("HTTPS request without error should not have -k flag, got: %s", result)
	}
}

func TestExportPythonNoHeaders(t *testing.T) {
	// Entry without request headers should not generate headers dict
	h := NewHar()
	entry := h.AddEntry("GET", "https://example.com/api", "HTTP/1.1", "")
	entry.SetResponseStatus(200, "OK")

	result := entry.ToPythonRequests()

	if strings.Contains(result, "headers = {") {
		t.Errorf("no headers dict should be generated when there are no request headers, got: %s", result)
	}
	if !strings.Contains(result, "requests.get(") {
		t.Errorf("should contain requests.get call, got: %s", result)
	}
}

func TestExportWgetEmptyHar(t *testing.T) {
	h := &Har{}
	result := h.ToWget()
	if result != "" {
		t.Errorf("empty HAR should return empty string, got: %s", result)
	}
}

func TestExportPythonEmptyHar(t *testing.T) {
	h := &Har{}
	result := h.ToPythonRequests()
	if result != "" {
		t.Errorf("empty HAR should return empty string, got: %s", result)
	}
}

func TestExportBuildHeadersDict(t *testing.T) {
	entry := &Entries{
		Request: Request{
			Headers: []Headers{
				{Name: "Content-Type", Value: "application/json"},
				{Name: "X-Custom", Value: "test's value"},
			},
		},
	}
	result := buildHeadersDict(entry)
	if !strings.Contains(result, "'Content-Type': 'application/json'") {
		t.Errorf("expected Content-Type header in dict, got: %s", result)
	}
	if !strings.Contains(result, `test\'s value`) {
		t.Errorf("expected escaped single quote in dict, got: %s", result)
	}
}

func TestExportBuildHeadersDictEmpty(t *testing.T) {
	entry := &Entries{
		Request: Request{
			Headers: []Headers{},
		},
	}
	result := buildHeadersDict(entry)
	if result != "" {
		t.Errorf("expected empty string for empty headers, got: %s", result)
	}
}

func TestExportSaveAsPostmanCollection_WriteError(t *testing.T) {
	h := createTestHar()

	// Write to an invalid path
	err := h.SaveAsPostmanCollection("/nonexistent_dir/subdir/collection.json")
	assertHarErrorCode(t, err, ErrCodeFileSystem)
}

func TestExportEscapePythonStringCarriageReturn(t *testing.T) {
	result := escapePythonString("line1\r\nline2")
	if !strings.Contains(result, `\r`) {
		t.Errorf("expected carriage return to be escaped, got: %s", result)
	}
}

func TestExportAllMethodsConsistency(t *testing.T) {
	h := createTestHar()

	// Ensure that no export method panics.
	curlResult := h.ToCurl()
	wgetResult := h.ToWget()
	pythonResult := h.ToPythonRequests()
	postmanResult, postmanErr := h.ToPostmanCollection()
	xmlResult, xmlErr := h.ToXML()

	if curlResult == "" {
		t.Error("ToCurl() should not return an empty string")
	}
	if wgetResult == "" {
		t.Error("ToWget() should not return an empty string")
	}
	if pythonResult == "" {
		t.Error("ToPythonRequests() should not return an empty string")
	}
	if postmanErr != nil {
		t.Errorf("ToPostmanCollection() returned an error: %v", postmanErr)
	}
	if len(postmanResult) == 0 {
		t.Error("ToPostmanCollection() should not return empty data.")
	}
	if xmlErr != nil {
		t.Errorf("ToXML() returned an error: %v", xmlErr)
	}
	if xmlResult == "" {
		t.Error("ToXML() should not return an empty string")
	}
}

func TestExportEntryMethodsConsistency(t *testing.T) {
	h := NewHar()
	entry := h.AddEntry("GET", "https://example.com/api", "HTTP/1.1", "")
	entry.AddRequestHeader("Accept", "application/json")
	entry.SetResponseStatus(200, "OK")

	curlResult := entry.ToCurl()
	wgetResult := entry.ToWget()
	pythonResult := entry.ToPythonRequests()

	if curlResult == "" {
		t.Error("Entries.ToCurl() should not return an empty string")
	}
	if wgetResult == "" {
		t.Error("Entries.ToWget() should not return an empty string")
	}
	if pythonResult == "" {
		t.Error("Entries.ToPythonRequests() should not return an empty string")
	}
}

func TestExportNilEntryHelpers(t *testing.T) {
	assertDoesNotPanic(t, func() {
		if got := entryToCurl(nil); got != "" {
			t.Fatalf("entryToCurl(nil) = %q, want empty", got)
		}
		if got := entryToWget(nil); got != "" {
			t.Fatalf("entryToWget(nil) = %q, want empty", got)
		}
		if got := entryToPythonRequests(nil); got != "" {
			t.Fatalf("entryToPythonRequests(nil) = %q, want empty", got)
		}
		if got := buildHeadersDict(nil); got != "" {
			t.Fatalf("buildHeadersDict(nil) = %q, want empty", got)
		}
		if got := hasAcceptEncoding(nil); got {
			t.Fatal("hasAcceptEncoding(nil) = true, want false")
		}
		if got := entryToPostmanItem(nil); got.Name != "" || got.Request.Method != "" || got.Request.URL.Raw != "" {
			t.Fatalf("entryToPostmanItem(nil) = %#v, want zero value", got)
		}
	})
}
