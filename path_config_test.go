package artifactory

import (
	"context"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
)

func TestConfig_Write(t *testing.T) {
	tests := []struct {
		expectedToSucceed bool
		data              map[string]interface{}
	}{
		{
			true,
			map[string]interface{}{
				"address": "https://example.com/artifactory",
				"api_key": "abc123",
			},
		},
		{false, map[string]interface{}{"address": "https://example.com/artifactory"}},
		{false, map[string]interface{}{"api_key": "abc123"}},
		{false, map[string]interface{}{}},
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
		if err != nil || (resp != nil && resp.IsError()) {
			if test.expectedToSucceed {
				t.Fatalf("Expected test case to succeed, err:%s resp:%#v\n", err, resp)
			}
		} else {
			if !test.expectedToSucceed {
				t.Fatalf("Expected test case to fail")
			}
		}
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
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%s resp:%#v\n", err, resp)
	}

	req = &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "config",
		Storage:   storage,
		Data:      nil,
	}

	resp, err = b.HandleRequest(context.Background(), req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%s resp:%#v\n", err, resp)
	}

	if resp.Data["address"] != data["address"] {
		t.Fatalf("Read address did not equal expected: %v", resp.Data["address"])
	}
}
