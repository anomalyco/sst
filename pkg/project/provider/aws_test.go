package provider

import (
	"bytes"
	"io"
	"testing"
)

func TestPrepareStateUploadCompressed(t *testing.T) {
	t.Parallel()

	input := bytes.Repeat([]byte(`{"resource":"value","nested":{"enabled":true}}`), 1024)
	body, contentEncoding, err := prepareHomeUpload(bytes.NewReader(input), true)
	if err != nil {
		t.Fatalf("prepareHomeUpload returned error: %v", err)
	}
	if contentEncoding == nil || *contentEncoding != "gzip" {
		t.Fatalf("expected gzip content encoding, got %v", contentEncoding)
	}

	encoded, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("reading encoded data failed: %v", err)
	}
	if len(encoded) >= len(input) {
		t.Fatalf("expected compressed upload to be smaller than raw input, got raw=%d uploaded=%d", len(input), len(encoded))
	}

	reader, err := decodeHomeReader(io.NopCloser(bytes.NewReader(encoded)), contentEncoding)
	if err != nil {
		t.Fatalf("decodeHomeReader returned error: %v", err)
	}
	decoded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading decoded data failed: %v", err)
	}
	if !bytes.Equal(decoded, input) {
		t.Fatal("decoded payload did not match original input")
	}
}

func TestPrepareStateUploadUncompressed(t *testing.T) {
	t.Parallel()

	input := []byte(`{"resource":"value"}`)
	body, contentEncoding, err := prepareHomeUpload(bytes.NewReader(input), false)
	if err != nil {
		t.Fatalf("prepareHomeUpload returned error: %v", err)
	}
	if contentEncoding != nil {
		t.Fatalf("expected no content encoding, got %v", *contentEncoding)
	}

	encoded, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("reading encoded data failed: %v", err)
	}
	if !bytes.Equal(encoded, input) {
		t.Fatal("plain upload payload did not match original input")
	}

	reader, err := decodeHomeReader(io.NopCloser(bytes.NewReader(encoded)), contentEncoding)
	if err != nil {
		t.Fatalf("decodeHomeReader returned error: %v", err)
	}
	decoded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading decoded data failed: %v", err)
	}
	if !bytes.Equal(decoded, input) {
		t.Fatal("decoded payload did not match original input")
	}
}

func TestDecodeStateReaderRejectsInvalidGzip(t *testing.T) {
	t.Parallel()

	encoding := "gzip"
	_, err := decodeHomeReader(io.NopCloser(bytes.NewReader([]byte("not-gzip"))), &encoding)
	if err == nil {
		t.Fatal("expected invalid gzip payload to return an error")
	}
}
