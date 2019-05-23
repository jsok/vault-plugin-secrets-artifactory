package artifactory

import (
	"context"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
)

func TestConfig_Write(t *testing.T) {
	tests := []struct {
		expectation Expectation
		data        map[string]interface{}
	}{
		{
			ExpectedToSucceed,
			map[string]interface{}{
				"address": "https://example.com/artifactory",
				"api_key": "abc123",
			},
		},
		{FailWithLogicalError, map[string]interface{}{"address": "https://example.com/artifactory"}},
		{FailWithLogicalError, map[string]interface{}{"api_key": "abc123"}},
		{FailWithLogicalError, map[string]interface{}{}},
	}

	for _, test := range tests {
		b, storage := newBackend(t)

		req := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "config",
			Storage:   storage,
			Data:      test.data,
		}

		resp, err := b.HandleRequest(context.Background(), req)
		assertLogicalResponse(t, test.expectation, err, resp)
	}
}

func TestConfig_WriteIdempotent(t *testing.T) {
	b, storage := newBackend(t)

	data := map[string]interface{}{
		"address": "https://example.com/artifactory",
		"api_key": "abc123",
	}

	req := &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "config",
		Storage:   storage,
		Data:      data,
	}

	resp, err := b.HandleRequest(context.Background(), req)
	assertLogicalResponse(t, ExpectedToSucceed, err, resp)

	req = &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "config",
		Storage:   storage,
		Data:      nil,
	}

	resp, err = b.HandleRequest(context.Background(), req)
	assertLogicalResponse(t, ExpectedToSucceed, err, resp)

	if resp.Data["address"] != data["address"] {
		t.Fatalf("Read address did not equal expected: %v", resp.Data["address"])
	}
}
