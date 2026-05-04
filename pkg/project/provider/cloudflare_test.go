package provider

import (
	"bytes"
	"context"
	"testing"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

func TestCloudflarePutDataUsesBytePayload(t *testing.T) {
	var gotBody interface{}

	home := &CloudflareHome{
		provider: &CloudflareProvider{
			identifier: &cloudflare.ResourceContainer{Identifier: "account"},
		},
		bootstrap: &bootstrap{State: "sst-state"},
		request: func(api *cloudflare.API, ctx context.Context, method string, path string, body interface{}) ([]byte, error) {
			gotBody = body
			return nil, nil
		},
	}

	err := home.putData("app", "demo", "dev", bytes.NewReader([]byte("hello")))
	if err != nil {
		t.Fatal(err)
	}

	body, ok := gotBody.([]byte)
	if !ok {
		t.Fatalf("expected []byte body, got %T", gotBody)
	}
	if string(body) != "hello" {
		t.Fatalf("unexpected body: %q", string(body))
	}
}
