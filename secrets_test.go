package artifactory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
)

func TestSecretAccessToken_Revoke(t *testing.T) {

	tests := []struct {
		expectation Expectation
		request     *logical.Request
		handler     http.HandlerFunc
	}{
		{
			ExpectedToSucceed,
			&logical.Request{
				Operation: logical.RevokeOperation,
				Secret: &logical.Secret{
					InternalData: map[string]interface{}{
						"role_name":   "test-role",
						"secret_type": accessTokenSecretType,
					},
				},
				Data: map[string]interface{}{
					"access_token": "fake-token",
				},
			},
			func(w http.ResponseWriter, r *http.Request) {
				if r.FormValue("token") != "fake-token" {
					t.Fatal("Revoke token request does not contain access token")
				}
				w.WriteHeader(http.StatusOK)
			},
		},
		{
			FailWithError,
			&logical.Request{
				Operation: logical.RevokeOperation,
				Secret: &logical.Secret{
					InternalData: map[string]interface{}{
						"role_name":   "test-role",
						"secret_type": accessTokenSecretType,
					},
				},
				Data: map[string]interface{}{
					"access_token": "fake-token",
				},
			},
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
		},
		{
			FailWithError,
			&logical.Request{
				Operation: logical.RevokeOperation,
				Secret: &logical.Secret{
					InternalData: map[string]interface{}{
						"secret_type": "Unknown secret type",
					},
				},
			},
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
		},
	}

	for _, test := range tests {
		// Mock the api/security/token/revoke endpoint
		ts := httptest.NewTLSServer(test.handler)
		defer ts.Close()

		b, storage := newBackend(t)

		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "config",
			Storage:   storage,
			Data: map[string]interface{}{
				"address":    ts.URL + "/",
				"api_key":    "abc123",
				"tls_verify": false,
			},
		})
		assertLogicalResponse(t, ExpectedToSucceed, err, resp)

		test.request.Storage = storage
		resp, err = b.HandleRequest(context.Background(), test.request)
		assertLogicalResponse(t, test.expectation, err, resp)
	}
}
