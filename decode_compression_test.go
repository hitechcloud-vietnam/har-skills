package har

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrotliRoundtrip(t *testing.T) {
	original := []byte("Hello Brotli! This is a test payload with some 非ASCII 字符 for good measure." + repeatString("X", 200))

	// Compress.
	compressed, err := CompressContent(original, "br")
	require.NoError(t, err, "Brotli compression should succeed.")
	assert.LessOrEqual(t, len(compressed), len(original), "Compressed Brotli data should be no larger than the original (short text may not compress).")

	// Decompress.
	decompressed, err := DecompressByEncoding(compressed, "br")
	require.NoError(t, err, "Brotli decompression should succeed.")
	assert.Equal(t, original, decompressed, "Round-trip should restore the original data.")

	// Verify that isBrotliData recognizes the data.
	assert.True(t, isBrotliData(compressed), "Compressed Brotli data should be recognized correctly.")
	assert.False(t, isBrotliData(original), "Plain text should not be misidentified as Brotli.")
}

func TestZstdRoundtrip(t *testing.T) {
	original := []byte("Zstandard compression test. 长文本测试：" + repeatString("A", 1000))

	compressed, err := CompressContent(original, "zstd")
	require.NoError(t, err, "Zstandard compression should succeed.")
	assert.Less(t, len(compressed), len(original), "Compressed Zstandard data should be smaller.")

	decompressed, err := DecompressByEncoding(compressed, "zstd")
	require.NoError(t, err, "Zstandard decompression should succeed.")
	assert.Equal(t, original, decompressed, "Round-trip should restore the original data.")

	// Verify isZstdData magic-number detection.
	assert.True(t, isZstdData(compressed), "Compressed Zstandard data should have the magic number.")
	assert.False(t, isZstdData(original), "Regular data should not have the Zstandard magic number.")
}

func TestMultiEncodingDecompress(t *testing.T) {
	// Scenario: gzip(deflate(original)) — two encoding layers.
	original := []byte("Multi-layer encoding test 数据")

	// Deflate first, then wrap with gzip (simulating Content-Encoding: gzip, deflate).
	deflated, err := CompressContent(original, "deflate")
	require.NoError(t, err)

	gzipWrapped, err := CompressContent(deflated, "gzip")
	require.NoError(t, err)

	// DecompressByEncoding with multiple encodings: decompress gzip, then deflate.
	decompressed, err := DecompressByEncoding(gzipWrapped, "gzip, deflate")
	require.NoError(t, err, "Multi-layer decompression should succeed.")
	assert.Equal(t, original, decompressed, "Two decompression layers should restore the original data.")

	// Three layers: br(gzip(deflate)).
	brGzipDeflate, err := CompressContent(gzipWrapped, "br")
	require.NoError(t, err)

	decompressed3, err := DecompressByEncoding(brGzipDeflate, "br, gzip, deflate")
	require.NoError(t, err)
	assert.Equal(t, original, decompressed3)
}

func TestDecompressByEncodingErrors(t *testing.T) {
	// Empty data.
	result, err := DecompressByEncoding(nil, "gzip")
	assert.NoError(t, err)
	assert.Nil(t, result)

	// Unsupported encoding.
	_, err = DecompressByEncoding([]byte("test"), "lz4")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")

	// Corrupted compressed data.
	corruptedGzip := []byte{0x1f, 0x8b, 0x00, 0x00} // gzip magic number, but truncated.
	_, err = DecompressByEncoding(corruptedGzip, "gzip")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gzip")

	corruptedZstd := []byte{0x28, 0xb5, 0x2f, 0xfd, 0x00} // Zstandard magic number, but truncated.
	_, err = DecompressByEncoding(corruptedZstd, "zstd")
	assert.Error(t, err)
}

func TestDecompressIfNeededWithBrotli(t *testing.T) {
	original := []byte("Auto-detect brotli 测试")
	compressed, err := CompressContent(original, "br")
	require.NoError(t, err)

	// decompressIfNeeded should detect compression by magic number and decompress.
	decompressed, err := decompressIfNeeded(compressed, "application/octet-stream")
	require.NoError(t, err)
	assert.Equal(t, original, decompressed)
}

func TestDecompressIfNeededWithZstd(t *testing.T) {
	original := []byte("Auto-detect zstd 测试")
	compressed, err := CompressContent(original, "zstd")
	require.NoError(t, err)

	decompressed, err := decompressIfNeeded(compressed, "")
	require.NoError(t, err)
	assert.Equal(t, original, decompressed)
}

func TestDecodeContentWithBrotliBase64(t *testing.T) {
	// Simulate Content.Encoding="base64" with a Brotli-compressed body in a HAR file.
	original := []byte("HAR body content compressed with brotli then base64 encoded")
	brCompressed, err := CompressContent(original, "br")
	require.NoError(t, err)

	// Construct a Content structure manually.
	content := &Content{
		Text:     encodeBase64(brCompressed),
		Encoding: "base64",
		MimeType: "application/json",
		Size:     len(original),
	}

	// DecodeContent should base64-decode first, then decompress Brotli.
	decoded, err := content.DecodeContent()
	require.NoError(t, err)
	assert.Equal(t, original, decoded)
}

func TestDecodeContentWithZstdBase64(t *testing.T) {
	original := []byte("zstd then base64")
	zstdCompressed, err := CompressContent(original, "zstd")
	require.NoError(t, err)

	content := &Content{
		Text:     encodeBase64(zstdCompressed),
		Encoding: "base64",
		MimeType: "",
		Size:     len(original),
	}

	decoded, err := content.DecodeContent()
	require.NoError(t, err)
	assert.Equal(t, original, decoded)
}

// Helper functions.
func repeatString(s string, n int) string {
	var b bytes.Buffer
	for i := 0; i < n; i++ {
		b.WriteString(s)
	}
	return b.String()
}

func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
