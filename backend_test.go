package artifactory

import (
	"context"
	"testing"
	"time"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/helper/logging"
	"github.com/hashicorp/vault/sdk/logical"
)

func newBackend(t *testing.T) (logical.Backend, logical.Storage) {
	defaultLeaseTTLVal := time.Hour * 12
	maxLeaseTTLVal := time.Hour * 24

	config := &logical.BackendConfig{
		Logger: logging.NewVaultLogger(log.Trace),

		System: &logical.StaticSystemView{
			DefaultLeaseTTLVal: defaultLeaseTTLVal,
			MaxLeaseTTLVal:     maxLeaseTTLVal,
		},
		StorageView: &logical.InmemStorage{},
	}
	b, err := Factory(context.Background(), config)
	if err != nil {
		t.Fatalf("unable to create backend: %v", err)
	}

	return b, config.StorageView
}

// Assertion helpers to handle the ternary logic of making a logical.Request

type Expectation int

const (
	ExpectedToSucceed Expectation = iota
	FailWithError
	FailWithLogicalError
)

func assertLogicalResponse(t *testing.T, expectation Expectation, err error, resp *logical.Response) {
	if err != nil || (resp != nil && resp.IsError()) {
		if expectation == ExpectedToSucceed {
			t.Fatalf("Expected test case to succeed, got err:%s resp:%#v\n", err, resp)
		}
	}
	if expectation == FailWithError && err == nil {
		t.Fatalf("Expected test case to fail with error but succeeded: resp:%#v\n", resp)
	}
	if expectation == FailWithLogicalError && resp != nil && !resp.IsError() {
		t.Fatalf("Expected test case to fail with logical error but succeeded: resp:%#v\n", resp)
	}
}
