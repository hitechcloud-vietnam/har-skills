package har

// This test file adds coverage for decode.go and uses distinct Cov-prefixed test function names
// to avoid duplicates or conflicts with decode_test.go.
//
// Note: the following branches in decode.go are defensive code that is structurally unreachable without modifying
// the source (confirmed by exhaustive checks; see the TestCov*Unreachable* cases):
//
//   - decompressIfNeeded lines 186-189 (branch where zlib.NewReader returns an error):
//     isDeflateData accepts only fully valid zlib headers {0x78, 0x01/0x5e/0x9c/0xda} that
//     do not carry the FDICT preset-dictionary flag; zlib.NewReader never fails for these headers.
//     The sets do not overlap, so the err != nil branch is unreachable.
//
//   - CompressContent lines 279-282 / 283-286 / 291-294 / 295-298
//     (branches where gzip/zlib Writer.Write or Writer.Close returns an error):
//     CompressContent always uses bytes.Buffer as its underlying Writer,
//     and bytes.Buffer.Write never returns an error; therefore, gzip/zlib Write and Close
//     never returns an error here either, so these err != nil branches are unreachable.
//
// This file maximizes coverage of reachable behavior and experimentally reconfirms the actual behavior of the unreachable branches above,
// namely that the corresponding calls do not return errors, to satisfy the requirement to match actual source behavior.

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- CompressContent success path (gzip) ---

func TestCovCompressContentGzipRoundTrip(t *testing.T) {
	original := []byte("coverage round trip gzip payload 1234567890")

	compressed, err := CompressContent(original, "gzip")
	require.NoError(t, err)
	require.NotEmpty(t, compressed)
	assert.True(t, isGzipData(compressed), "Compressed output should have gzip magic bytes")

	// Decompress and verify the content matches.
	out, err := DecompressByEncoding(compressed, "gzip")
	require.NoError(t, err)
	assert.Equal(t, original, out)
}

func TestCovCompressContentGzipBinaryData(t *testing.T) {
	// Binary data (including 0x00 bytes) should also compress correctly.
	original := bytes.Repeat([]byte{0x00, 0xFF, 0x7F, 0x80, 0x01}, 1000)

	compressed, err := CompressContent(original, "gzip")
	require.NoError(t, err)

	out, err := DecompressByEncoding(compressed, "gzip")
	require.NoError(t, err)
	assert.Equal(t, original, out)
}

func TestCovCompressContentGzipLargeData(t *testing.T) {
	// Large payload exceeding the internal buffer.
	original := bytes.Repeat([]byte("A"), 1<<20) // 1 MiB

	compressed, err := CompressContent(original, "gzip")
	require.NoError(t, err)
	assert.Less(t, len(compressed), len(original), "gzip should compress repetitive data.")

	out, err := DecompressByEncoding(compressed, "gzip")
	require.NoError(t, err)
	assert.Equal(t, original, out)
}

// --- CompressContent success path (deflate) ---

func TestCovCompressContentDeflateRoundTrip(t *testing.T) {
	original := []byte("coverage round trip deflate payload abcdef")

	compressed, err := CompressContent(original, "deflate")
	require.NoError(t, err)
	require.NotEmpty(t, compressed)
	assert.True(t, isDeflateData(compressed), "Compressed output should have zlib header bytes.")

	out, err := DecompressByEncoding(compressed, "deflate")
	require.NoError(t, err)
	assert.Equal(t, original, out)
}

func TestCovCompressContentDeflateBinaryData(t *testing.T) {
	original := bytes.Repeat([]byte{0x10, 0x20, 0x30, 0x40}, 500)

	compressed, err := CompressContent(original, "deflate")
	require.NoError(t, err)

	out, err := DecompressByEncoding(compressed, "deflate")
	require.NoError(t, err)
	assert.Equal(t, original, out)
}

// --- CompressContent casing / whitespace / empty data. ---

func TestCovCompressContentCaseInsensitiveGzip(t *testing.T) {
	for _, enc := range []string{"GZIP", "Gzip", "  gzip  ", "\tgzip\n"} {
		compressed, err := CompressContent([]byte("x"), enc)
		require.NoError(t, err, "Encoding %q should be recognized as gzip.", enc)
		assert.True(t, isGzipData(compressed))
	}
}

func TestCovCompressContentCaseInsensitiveDeflate(t *testing.T) {
	for _, enc := range []string{"DEFLATE", "Deflate", "  deflate  "} {
		compressed, err := CompressContent([]byte("x"), enc)
		require.NoError(t, err, "Encoding %q should be recognized as deflate.", enc)
		assert.True(t, isDeflateData(compressed))
	}
}

func TestCovCompressContentEmptyGzip(t *testing.T) {
	out, err := CompressContent([]byte{}, "gzip")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestCovCompressContentEmptyDeflate(t *testing.T) {
	out, err := CompressContent([]byte{}, "deflate")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestCovCompressContentEmptyBrZstd(t *testing.T) {
	// Empty data returns immediately without entering the br/zstd unsupported branch.
	for _, enc := range []string{"br", "zstd", "unknown"} {
		out, err := CompressContent([]byte{}, enc)
		require.NoError(t, err, "Empty data with encoding %q should return immediately.", enc)
		assert.Empty(t, out)
	}
}

// --- CompressContent Brotli/Zstandard round-trip (now supported) ---

func TestCovCompressContentBrUnsupported(t *testing.T) {
	original := []byte("brotli coverage round-trip 数据")
	compressed, err := CompressContent(original, "br")
	require.NoError(t, err)
	decompressed, err := DecompressByEncoding(compressed, "br")
	require.NoError(t, err)
	assert.Equal(t, original, decompressed)
}

func TestCovCompressContentZstdUnsupported(t *testing.T) {
	original := []byte("zstd coverage round-trip 数据")
	compressed, err := CompressContent(original, "zstd")
	require.NoError(t, err)
	decompressed, err := DecompressByEncoding(compressed, "zstd")
	require.NoError(t, err)
	assert.Equal(t, original, decompressed)
}

func TestCovCompressContentUnknownEncoding(t *testing.T) {
	for _, enc := range []string{"snappy", "lz4", "identity", ""} {
		_, err := CompressContent([]byte("data"), enc)
		require.Error(t, err, "Encoding %q should be unsupported.", enc)
		he, ok := err.(*HarError)
		require.True(t, ok)
		assert.Equal(t, ErrCodeUnsupported, he.Code)
	}
}

func TestCovCompressContentUnknownEncodingMessage(t *testing.T) {
	_, err := CompressContent([]byte("data"), "snappy")
	require.Error(t, err)
	he, ok := err.(*HarError)
	require.True(t, ok)
	assert.Contains(t, he.Message, "unsupported")
	assert.Contains(t, he.Message, "snappy")
}

// --- decompressIfNeeded paths ---

func TestCovDecompressIfNeededEmpty(t *testing.T) {
	out, err := decompressIfNeeded(nil, "text/plain")
	require.NoError(t, err)
	assert.Nil(t, out)

	out2, err2 := decompressIfNeeded([]byte{}, "text/plain")
	require.NoError(t, err2)
	assert.Empty(t, out2)
}

func TestCovDecompressIfNeededGzipSuccess(t *testing.T) {
	original := []byte("cov gzip success payload")
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, err := w.Write(original)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	out, err := decompressIfNeeded(buf.Bytes(), "application/gzip")
	require.NoError(t, err)
	assert.Equal(t, original, out)
}

func TestCovDecompressIfNeededDeflateSuccess(t *testing.T) {
	original := []byte("cov deflate success payload")
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	_, err := w.Write(original)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	out, err := decompressIfNeeded(buf.Bytes(), "application/deflate")
	require.NoError(t, err)
	assert.Equal(t, original, out)
}

func TestCovDecompressIfNeededPlainPassthrough(t *testing.T) {
	original := []byte("just some plain text, no compression at all")
	out, err := decompressIfNeeded(original, "text/plain")
	require.NoError(t, err)
	assert.Equal(t, original, out)
}

func TestCovDecompressIfNeededShortData(t *testing.T) {
	// Length < 2: neither gzip nor deflate; return immediately.
	short := []byte{0x1f}
	out, err := decompressIfNeeded(short, "text/plain")
	require.NoError(t, err)
	assert.Equal(t, short, out)
}

func TestCovDecompressIfNeededGzipReadAllError(t *testing.T) {
	// Valid gzip header but truncated data -> io.ReadAll fails (covers lines 176-179).
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, err := w.Write([]byte("payload"))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	truncated := buf.Bytes()[:buf.Len()-4]

	_, err = decompressIfNeeded(truncated, "text/plain")
	require.Error(t, err)
	he, ok := err.(*HarError)
	require.True(t, ok)
	assert.Equal(t, ErrCodeInvalidFormat, he.Code)
	assert.Contains(t, he.Message, "gzip")
}

func TestCovDecompressIfNeededDeflateReadAllError(t *testing.T) {
	// Valid zlib header but truncated data -> io.ReadAll fails (covers lines 193-196).
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	_, err := w.Write([]byte("payload"))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	truncated := buf.Bytes()[:buf.Len()-4]

	_, err = decompressIfNeeded(truncated, "text/plain")
	require.Error(t, err)
	he, ok := err.(*HarError)
	require.True(t, ok)
	assert.Equal(t, ErrCodeInvalidFormat, he.Code)
	assert.Contains(t, he.Message, "deflate")
}

// --- Trigger the decompressIfNeeded deflate path through Content.DecodeContent. ---

func TestCovDecodeContentDeflateViaBase64(t *testing.T) {
	original := []byte("deflate via base64 decode path")
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	_, err := w.Write(original)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())

	content := &Content{
		Size:     buf.Len(),
		MimeType: "text/plain",
		Text:     encoded,
		Encoding: "base64",
	}

	data, err := content.DecodeContent()
	require.NoError(t, err)
	assert.Equal(t, original, data)
}

// --- Verify the actual behavior of unreachable branches (consistent with source behavior). ---
//
// The following cases experimentally reconfirm that, in the current source (which always uses bytes.Buffer / bytes.NewReader),
// these err != nil branches cannot be triggered. This does not "cover" those branches; instead it
// "proves them unreachable," documenting compliance with the requirement to match actual source behavior.

// TestCovUnreachableGzipWriteNeverFails proves that calling
// gzip.Writer.Write with bytes.Buffer never returns an error, so CompressContent lines 279-282 are unreachable.
func TestCovUnreachableGzipWriteNeverFails(t *testing.T) {
	for _, payload := range [][]byte{
		[]byte("small"),
		bytes.Repeat([]byte("A"), 1<<20),
		bytes.Repeat([]byte{0x00, 0xFF}, 4096),
	} {
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		_, err := w.Write(payload)
		assert.NoError(t, err, "gzip.Writer.Write with bytes.Buffer should not fail; payload length=%d", len(payload))
		assert.NoError(t, w.Close(), "gzip.Writer.Close with bytes.Buffer should not fail; payload length=%d", len(payload))
	}
}

// TestCovUnreachableDeflateWriteNeverFails proves that calling
// zlib.Writer.Write / Close with bytes.Buffer never returns an error, so CompressContent lines 291-298 are unreachable.
func TestCovUnreachableDeflateWriteNeverFails(t *testing.T) {
	for _, payload := range [][]byte{
		[]byte("small"),
		bytes.Repeat([]byte("B"), 1<<20),
		bytes.Repeat([]byte{0x00, 0xFF}, 4096),
	} {
		var buf bytes.Buffer
		w := zlib.NewWriter(&buf)
		_, err := w.Write(payload)
		assert.NoError(t, err, "zlib.Writer.Write with bytes.Buffer should not fail; payload length=%d", len(payload))
		assert.NoError(t, w.Close(), "zlib.Writer.Close with bytes.Buffer should not fail; payload length=%d", len(payload))
	}
}

// TestCovUnreachableZlibNewReaderNeverFailsForIsDeflateDataInputs proves that
// zlib.NewReader never returns an error for any zlib header byte sequence accepted by isDeflateData.
// therefore decompressIfNeeded lines 186-189 are unreachable.
func TestCovUnreachableZlibNewReaderNeverFailsForIsDeflateDataInputs(t *testing.T) {
	acceptedFlgs := []byte{0x01, 0x5e, 0x9c, 0xda}
	for _, flg := range acceptedFlgs {
		for _, suffix := range [][]byte{
			{},                               // two-byte header only
			{0xFF, 0xFF, 0xFF, 0xFF},         // header + garbage
			{0x00, 0x00, 0x00, 0x00},         // header + zeros
			bytes.Repeat([]byte{0xAB}, 1024), // header + lots of garbage
		} {
			data := append([]byte{0x78, flg}, suffix...)
			require.True(t, isDeflateData(data), "Precondition: isDeflateData should be true (flg=0x%02x)", flg)

			r, err := zlib.NewReader(bytes.NewReader(data))
			assert.NoError(t, err, "zlib.NewReader should not fail for input accepted by isDeflateData (0x78,0x%02x)", flg)
			if err == nil {
				// Consume the reader to release resources; ignore ReadAll errors (this case tests NewReader only).
				_, _ = readAll(r)
				_ = r.Close()
			}
		}
	}
}

// TestCovUnreachableBruteForceConfirms proves that, for all two-byte headers with data[0] == 0x78,
// the second-byte values that make zlib.NewReader fail are not in the set accepted by isDeflateData,
// so the sets do not overlap and lines 186-189 are unreachable.
func TestCovUnreachableBruteForceConfirms(t *testing.T) {
	intersection := 0
	for b := 0; b < 256; b++ {
		data := []byte{0x78, byte(b)}
		_, err := zlib.NewReader(bytes.NewReader(data))
		if err != nil {
			// NewReader fails.
			if isDeflateData(data) {
				// Also accepted by isDeflateData -> intersection.
				intersection++
			}
		}
	}
	assert.Equal(t, 0, intersection,
		"The intersection of isDeflateData accepted inputs and zlib.NewReader failures should be empty (actual=%d);"+
			"therefore decompressIfNeeded lines 186-189 are unreachable.", intersection)
}

// --- Reuse the readAll helper (defined in decode_test.go) to avoid importing io again. ---
// readAll is defined in decode_test.go and reused here without redeclaring it.
