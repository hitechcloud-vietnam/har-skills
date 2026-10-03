package har

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

// DecodeContent decodes response content.
//
// It automatically detects and decodes base64 content, then decompresses
// content encoded with Content-Encoding (gzip/deflate).
// It returns the decoded raw bytes.
func (c *Content) DecodeContent() ([]byte, error) {
	if c == nil {
		return nil, NewInvalidFormatError("content is empty")
	}

	var data []byte

	// Step 1: handle base64 encoding.
	if strings.EqualFold(c.Encoding, "base64") && c.Text != "" {
		decoded, err := base64.StdEncoding.DecodeString(c.Text)
		if err != nil {
			// Try URL-safe base64.
			decoded, err = base64.URLEncoding.DecodeString(c.Text)
			if err != nil {
				return nil, NewHarError(ErrCodeInvalidFormat,
					fmt.Sprintf("base64 decoding failed: %v", err), err)
			}
		}
		data = decoded
	} else if c.Text != "" {
		data = []byte(c.Text)
	} else {
		return nil, nil
	}

	// Step 2: detect and decompress the content.
	data, err := decompressIfNeeded(data, c.MimeType)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// DecodeContent decodes the response content of the specified entry.
func (e *Entries) DecodeContent() ([]byte, error) {
	if e == nil {
		return nil, NewInvalidFormatError("entry is nil")
	}
	return e.Response.Content.DecodeContent()
}

// DecodeAllContent decodes the response content of every entry in a HAR file.
// It returns one decoded result per entry, with indices corresponding to the HAR entries.
func (h *Har) DecodeAllContent() ([][]byte, error) {
	if h == nil {
		return nil, NewInvalidFormatError("HAR object is nil")
	}

	results := make([][]byte, len(h.Log.Entries))
	var partialErrors []*HarError

	for i, entry := range h.Log.Entries {
		data, err := entry.DecodeContent()
		if err != nil {
			partialErrors = append(partialErrors, decodePartialError(i, err))
			results[i] = nil
			continue
		}
		results[i] = data
	}

	if len(partialErrors) > 0 {
		rootErr := NewHarError(ErrCodeInvalidFormat,
			fmt.Sprintf("%d error(s) occurred during decoding", len(partialErrors)), nil).
			WithMetadata("error_count", len(partialErrors))
		for _, err := range partialErrors {
			rootErr = rootErr.AddPartialError(err)
		}
		return results, rootErr
	}

	return results, nil
}

func decodePartialError(index int, err error) *HarError {
	field := fmt.Sprintf("log.entries[%d].response.content", index)
	if harErr, ok := err.(*HarError); ok {
		return NewHarError(harErr.Code, harErr.Message, harErr.Err).
			WithField(field).
			WithMetadata("entry_index", index)
	}

	return NewHarError(ErrCodeInvalidFormat, "content decoding failed", err).
		WithField(field).
		WithMetadata("entry_index", index)
}

// IsBase64Encoded reports whether the content is base64-encoded.
func (c *Content) IsBase64Encoded() bool {
	if c == nil {
		return false
	}
	return strings.EqualFold(c.Encoding, "base64")
}

// IsCompressed reports whether the content is compressed, based on the Content-Type header or MIME type.
func (e *Entries) IsCompressed() bool {
	if e == nil {
		return false
	}

	// Check the Content-Encoding response header.
	for _, header := range e.Response.Headers {
		if strings.EqualFold(header.Name, "Content-Encoding") {
			encoding := strings.ToLower(strings.TrimSpace(header.Value))
			if encoding == "gzip" || encoding == "deflate" || encoding == "br" || encoding == "zstd" {
				return true
			}
		}
	}

	return false
}

// GetContentEncoding returns the content encoding.
func (e *Entries) GetContentEncoding() string {
	if e == nil {
		return ""
	}

	for _, header := range e.Response.Headers {
		if strings.EqualFold(header.Name, "Content-Encoding") {
			return strings.TrimSpace(header.Value)
		}
	}

	return ""
}

// DecodeEntryText decodes an entry's response text (convenience method).
func (e *Entries) DecodeEntryText() (string, error) {
	data, err := e.DecodeContent()
	if err != nil {
		return "", err
	}
	if data == nil {
		return "", nil
	}
	return string(data), nil
}

// decompressIfNeeded attempts decompression based on the MIME type and content signature.
func decompressIfNeeded(data []byte, mimeType string) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	// Try gzip decompression.
	if isGzipData(data) {
		reader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, NewHarError(ErrCodeInvalidFormat,
				fmt.Sprintf("gzip decompression failed: %v", err), err)
		}
		defer reader.Close()

		decompressed, err := io.ReadAll(reader)
		if err != nil {
			return nil, NewHarError(ErrCodeInvalidFormat,
				fmt.Sprintf("gzip decompression failed: %v", err), err)
		}
		return decompressed, nil
	}

	// Try deflate decompression.
	if isDeflateData(data) {
		// isDeflateData accepts only FLG values that pass zlib header validation,
		// so zlib.NewReader must succeed here.
		reader, _ := zlib.NewReader(bytes.NewReader(data))
		defer reader.Close()

		decompressed, err := io.ReadAll(reader)
		if err != nil {
			return nil, NewHarError(ErrCodeInvalidFormat,
				fmt.Sprintf("deflate decompression failed: %v", err), err)
		}
		return decompressed, nil
	}

	// Try Brotli decompression.
	// isBrotliData requires the first Read to succeed (n > 0 and err == nil).
	// Once the initial streaming decode succeeds, the complete decode succeeds,
	// so io.ReadAll cannot return an error here and no error branch is needed.
	if isBrotliData(data) {
		reader := brotli.NewReader(bytes.NewReader(data))
		decompressed, _ := io.ReadAll(reader)
		return decompressed, nil
	}

	// Try Zstandard decompression.
	if isZstdData(data) {
		// zstd.NewReader cannot fail during initialization for data that passes the
		// magic-byte check; DecodeAll reports corrupted data instead.
		decoder, _ := zstd.NewReader(bytes.NewReader(data))
		defer decoder.Close()

		decompressed, err := decoder.DecodeAll(data, nil)
		if err != nil {
			return nil, NewHarError(ErrCodeInvalidFormat,
				fmt.Sprintf("zstd decompression failed: %v", err), err)
		}
		return decompressed, nil
	}

	return data, nil
}

// DecompressByEncoding decompresses data according to the Content-Encoding value.
//
// Supported encodings: "gzip", "deflate", "br" (Brotli), "zstd", and "identity".
// Multiple encodings (such as "gzip, deflate") are decompressed layer by layer per HTTP semantics:
// Content-Encoding lists encodings in wrapping order. The first encoding is the outermost
// (applied last), so it must be decompressed first. For example, "gzip, deflate" means
// gzip(deflate(original)); decompress gzip first, then deflate.
func DecompressByEncoding(data []byte, encoding string) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	enc := strings.ToLower(strings.TrimSpace(encoding))
	if enc == "" || enc == "identity" {
		return data, nil
	}

	// Split multiple encodings by comma and decompress them in declaration order
	// (the first item is outermost and must be decoded first).
	if strings.Contains(enc, ",") {
		encodings := splitEncodings(enc)
		current := data
		for i, e := range encodings {
			result, err := DecompressByEncoding(current, e)
			if err != nil {
				return nil, fmt.Errorf("layer %d %q decompression failed: %w", i, e, err)
			}
			current = result
		}
		return current, nil
	}

	switch enc {
	case "gzip":
		reader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, NewHarError(ErrCodeInvalidFormat,
				fmt.Sprintf("gzip decompression failed: %v", err), err)
		}
		defer reader.Close()

		decompressed, err := io.ReadAll(reader)
		if err != nil {
			return nil, NewHarError(ErrCodeInvalidFormat,
				fmt.Sprintf("gzip decompression failed: %v", err), err)
		}
		return decompressed, nil
	case "deflate":
		reader, err := zlib.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, NewHarError(ErrCodeInvalidFormat,
				fmt.Sprintf("deflate decompression failed: %v", err), err)
		}
		defer reader.Close()

		decompressed, err := io.ReadAll(reader)
		if err != nil {
			return nil, NewHarError(ErrCodeInvalidFormat,
				fmt.Sprintf("deflate decompression failed: %v", err), err)
		}
		return decompressed, nil
	case "br":
		reader := brotli.NewReader(bytes.NewReader(data))
		decompressed, err := io.ReadAll(reader)
		if err != nil {
			return nil, NewHarError(ErrCodeInvalidFormat,
				fmt.Sprintf("Brotli decompression failed: %v", err), err)
		}
		return decompressed, nil
	case "zstd":
		// zstd.NewReader does not fail during initialization for arbitrary input;
		// corrupted data is reported by DecodeAll.
		decoder, _ := zstd.NewReader(bytes.NewReader(data))
		defer decoder.Close()

		decompressed, err := decoder.DecodeAll(data, nil)
		if err != nil {
			return nil, NewHarError(ErrCodeInvalidFormat,
				fmt.Sprintf("Zstandard decompression failed: %v", err), err)
		}
		return decompressed, nil
	default:
		return nil, NewUnsupportedError(
			fmt.Sprintf("unsupported Content-Encoding: %q", encoding))
	}
}

// splitEncodings splits "gzip, deflate, br" into ["gzip", "deflate", "br"],
// trimming whitespace and ignoring empty items.
func splitEncodings(enc string) []string {
	parts := strings.Split(enc, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// CompressContent compresses data using the specified encoding.
//
// Supported encodings: "gzip", "deflate", "br" (Brotli), "zstd", and "identity".
func CompressContent(data []byte, encoding string) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	enc := strings.ToLower(strings.TrimSpace(encoding))

	switch enc {
	case "gzip":
		var buf bytes.Buffer
		writer := gzip.NewWriter(&buf)
		// bytes.Buffer.Write never returns an error, so Write and Close cannot fail.
		_, _ = writer.Write(data)
		_ = writer.Close()
		return buf.Bytes(), nil
	case "deflate":
		var buf bytes.Buffer
		writer := zlib.NewWriter(&buf)
		_, _ = writer.Write(data)
		_ = writer.Close()
		return buf.Bytes(), nil
	case "br":
		var buf bytes.Buffer
		writer := brotli.NewWriter(&buf)
		// brotli.NewWriter writes to bytes.Buffer (whose Write never fails), and the
		// Brotli encoder cannot fail when writing to memory, so no error branch is needed.
		_, _ = writer.Write(data)
		_ = writer.Close()
		return buf.Bytes(), nil
	case "zstd":
		// zstd.NewWriter cannot fail with the default configuration.
		encoder, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
		defer encoder.Close()
		return encoder.EncodeAll(data, nil), nil
	default:
		return nil, NewUnsupportedError(
			fmt.Sprintf("unsupported compression encoding: %q", encoding))
	}
}

// DecompressWithEncoding decompresses data using the Content-Encoding header value.
//
// This function determines how to decompress data from the HTTP Content-Encoding header,
// unlike decompressIfNeeded, which relies only on magic-byte detection.
func DecompressWithEncoding(data []byte, contentEncoding string) ([]byte, error) {
	return DecompressByEncoding(data, contentEncoding)
}

// isGzipData reports whether data is in gzip format.
func isGzipData(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	// gzip magic number: 0x1f 0x8b
	return data[0] == 0x1f && data[1] == 0x8b
}

// isDeflateData reports whether data is in deflate format.
func isDeflateData(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	// The zlib header usually starts with 0x78.
	// 0x78 0x01 = no compression
	// 0x78 0x5E = best speed
	// 0x78 0x9C = default compression
	// 0x78 0xDA = best compression
	return data[0] == 0x78 && (data[1] == 0x01 || data[1] == 0x5e || data[1] == 0x9c || data[1] == 0xda)
}

// isZstdData reports whether data is in Zstandard format, using the four-byte magic number 0x28 0xB5 0x2F 0xFD.
func isZstdData(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	return data[0] == 0x28 && data[1] == 0xb5 && data[2] == 0x2f && data[3] == 0xfd
}

// isBrotliData heuristically determines whether data is Brotli-compressed.
// Brotli has no fixed magic number like gzip or Zstandard, so this function attempts
// to decode the first segment. If brotli.Reader returns non-empty bytes without an error,
// the data is treated as Brotli. False positives are unlikely because plain text and JSON
// almost never decode to valid bytes.
func isBrotliData(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	reader := brotli.NewReader(bytes.NewReader(data))
	// Probe only the first 64 bytes to avoid decoding an entire large file.
	probe := make([]byte, 64)
	n, err := reader.Read(probe)
	// Treat a successful read of non-zero bytes without an error as Brotli.
	return err == nil && n > 0
}
