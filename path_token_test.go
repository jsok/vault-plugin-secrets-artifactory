package artifactory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"

	rtTokenService "github.com/jsok/vault-plugin-secrets-artifactory/pkg/token"
)

func TestToken_Read(t *testing.T) {
	tests := []struct {
		expectation  Expectation
		createConfig bool
		createRole   bool
		name         string
		handler      http.HandlerFunc
	}{
		{
			FailWithLogicalError, // Role does not exist
			true,
			false,
			"nonexistent-role",
			nil,
		},
		{
			FailWithLogicalError, // Backend has not been configured
			false,
			true,
			"test",
			nil,
		},
		{
			ExpectedToSucceed,
			true,
			true,
			"test",
			func(w http.ResponseWriter, r *http.Request) {
				body, err := json.Marshal(
					&rtTokenService.CreateTokenResponse{
						AccessToken: "abc123",
						ExpiresIn:   3600,
						Scope:       "api:* member-of-groups:readers",
						TokenType:   "Bearer",
					})
				if err != nil {
					t.Fatal("Encoding mock HTTP response failed!")
				}
				w.Write(body)
			},
		},
		{
			FailWithLogicalError, // HTTP 403 response from Artifactory
			true,
			true,
			"test",
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
			},
		},
	}

	for _, test := range tests {
		b, storage := newBackend(t)

		// Mock the api/security/token endpoint
		serverURL := "http://example.com/"
		if test.handler != nil {
			ts := httptest.NewTLSServer(test.handler)
			defer ts.Close()
			serverURL = ts.URL + "/"
		}

		if test.createConfig {
			createConfigReq := &logical.Request{
				Operation: logical.UpdateOperation,
				Path:      "config",
				Storage:   storage,
				Data: map[string]interface{}{
					"address":    serverURL,
					"api_key":    "abc123",
					"tls_verify": false,
				},
			}
			resp, err := b.HandleRequest(context.Background(), createConfigReq)
			assertLogicalResponse(t, ExpectedToSucceed, err, resp)
		}
		if test.createRole {
			createRoleReq := &logical.Request{
				Operation: logical.CreateOperation,
				Path:      "roles/test",
				Storage:   storage,
				Data: map[string]interface{}{
					"username":         "user",
					"member_of_groups": "group",
				},
			}
			resp, err := b.HandleRequest(context.Background(), createRoleReq)
			assertLogicalResponse(t, ExpectedToSucceed, err, resp)
		}

		if test.handler != nil {
			req := &logical.Request{
				Operation: logical.ReadOperation,
				Path:      "token/" + test.name,
				Storage:   storage,
			}
			resp, err := b.HandleRequest(context.Background(), req)
			assertLogicalResponse(t, test.expectation, err, resp)
		}
	}
}
