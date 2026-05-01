package provider

import (
	"bytes"
	"io"
	"testing"
)

func TestGzipEncodeRoundTrip(t *testing.T) {
	t.Parallel()

	input := bytes.Repeat([]byte(`{"resource":"value","nested":{"enabled":true}}`), 1024)
	encoded, err := gzipEncode(bytes.NewReader(input))
	if err != nil {
		t.Fatalf("gzipEncode returned error: %v", err)
	}
	encodedBytes, err := io.ReadAll(encoded)
	if err != nil {
		t.Fatalf("reading encoded data failed: %v", err)
	}
	if len(encodedBytes) >= len(input) {
		t.Fatalf("expected gzip output to be smaller than input, got raw=%d encoded=%d", len(input), len(encodedBytes))
	}

	decoded, err := gzipDecode(bytes.NewReader(encodedBytes))
	if err != nil {
		t.Fatalf("gzipDecode returned error: %v", err)
	}
	got, err := io.ReadAll(decoded)
	if err != nil {
		t.Fatalf("reading decoded data failed: %v", err)
	}
	if !bytes.Equal(got, input) {
		t.Fatal("round-tripped payload did not match original input")
	}
}

func TestGzipEncodeProducesGzipMagic(t *testing.T) {
	t.Parallel()

	encoded, err := gzipEncode(bytes.NewReader([]byte("hello")))
	if err != nil {
		t.Fatalf("gzipEncode returned error: %v", err)
	}
	encodedBytes, err := io.ReadAll(encoded)
	if err != nil {
		t.Fatalf("reading encoded data failed: %v", err)
	}
	if len(encodedBytes) < 2 || encodedBytes[0] != 0x1f || encodedBytes[1] != 0x8b {
		t.Fatalf("expected output to start with gzip magic bytes, got % x", encodedBytes)
	}
}

func TestGzipEncodeEmpty(t *testing.T) {
	t.Parallel()

	encoded, err := gzipEncode(bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("gzipEncode returned error: %v", err)
	}
	encodedBytes, err := io.ReadAll(encoded)
	if err != nil {
		t.Fatalf("reading encoded data failed: %v", err)
	}

	decoded, err := gzipDecode(bytes.NewReader(encodedBytes))
	if err != nil {
		t.Fatalf("gzipDecode returned error: %v", err)
	}
	got, err := io.ReadAll(decoded)
	if err != nil {
		t.Fatalf("reading decoded data failed: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty payload, got %d bytes", len(got))
	}
}

func TestGzipDecodePlainPassthrough(t *testing.T) {
	t.Parallel()

	plain := []byte(`{"legacy":"state","compressed":false}`)
	decoded, err := gzipDecode(bytes.NewReader(plain))
	if err != nil {
		t.Fatalf("gzipDecode returned error: %v", err)
	}
	got, err := io.ReadAll(decoded)
	if err != nil {
		t.Fatalf("reading decoded data failed: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatal("plain payload did not pass through unchanged")
	}
}

func TestGzipDecodeInvalidGzip(t *testing.T) {
	t.Parallel()

	// Starts with the gzip magic bytes but is not a valid gzip stream.
	invalid := []byte{0x1f, 0x8b, 0x00, 0x00}
	_, err := gzipDecode(bytes.NewReader(invalid))
	if err == nil {
		t.Fatal("expected invalid gzip payload to return an error")
	}
}
