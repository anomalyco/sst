package provider

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
)

// gzipEncode reads r fully and returns a reader of the gzipped bytes.
// Buffers in memory; signature is stable so a streaming variant can replace
// it later without changing callers.
func gzipEncode(r io.Reader) (io.Reader, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := io.Copy(gw, r); err != nil {
		gw.Close()
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}
	return &buf, nil
}

// gzipDecode returns a reader that yields the original bytes from r.
// If r begins with the gzip magic bytes, it is decoded transparently;
// otherwise r is returned as-is. This keeps reads compatible with both
// compressed and legacy plain payloads.
func gzipDecode(r io.Reader) (io.Reader, error) {
	br := bufio.NewReader(r)
	head, err := br.Peek(2)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(head) == 2 && head[0] == 0x1f && head[1] == 0x8b {
		return gzip.NewReader(br)
	}
	return br, nil
}
