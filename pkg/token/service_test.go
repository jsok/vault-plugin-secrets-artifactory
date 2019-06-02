package token

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jfrog/jfrog-client-go/artifactory/auth"
	"github.com/jfrog/jfrog-client-go/artifactory/httpclient"
)

func TestCreateToken(t *testing.T) {
	tests := []struct {
		shouldSucceed bool
		request       *CreateTokenRequest
		handler       http.HandlerFunc
	}{
		{
			true,
			&CreateTokenRequest{
				Username:  "username",
				Scope:     "member-of-groups:reader,PowerUser",
				ExpiresIn: 3600,
			},
			func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf("Expected POST but got request with method: %s\n", r.Method)
				}
				if r.URL.Path != "/"+tokenApiPath {
					t.Fatalf("Expected request path to be %s, got %s\n", tokenApiPath, r.URL.Path)
				}
				if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
					t.Fatalf("Unexpected content-type header: %s\n", r.Header.Get("Content-Type"))
				}
				if err := r.ParseForm(); err != nil {
					t.Fatalf("Unable to parse form data from request: %v\n", err)
				}
				body, err := json.Marshal(&CreateTokenResponse{
					AccessToken: "fake-access-token",
					ExpiresIn:   3600,
					Scope:       "api:* member-of-groups:readers,PowerUser",
					TokenType:   "Bearer",
				})
				if err != nil {
					t.Fatal("Encoding mock HTTP response failed!")
				}
				w.Write(body)
			},
		},
		{
			false,
			nil,
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
		},
		{
			false,
			&CreateTokenRequest{},
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
			},
		},
		{
			false,
			&CreateTokenRequest{},
			func(w http.ResponseWriter, r *http.Request) {
				body, err := json.Marshal("not a valid CreateTokenResponse")
				if err != nil {
					t.Fatal("Encoding mock HTTP response failed!")
				}
				w.Write(body)
			},
		},
	}

	for _, test := range tests {
		ts := httptest.NewTLSServer(test.handler)
		defer ts.Close()

		rtDetails := auth.NewArtifactoryDetails()
		rtDetails.SetUrl(ts.URL + "/")
		rtDetails.SetApiKey("fake-api-key")

		client, err := httpclient.ArtifactoryClientBuilder().
			SetInsecureTls(true).
			SetArtDetails(&rtDetails).
			Build()
		if err != nil {
			t.Fatalf("Failed to create Artifactory client: %v\n", err)
		}

		tokenService := NewAccessTokenService(client)
		tokenService.SetArtifactoryDetails(rtDetails)
		_, err = tokenService.CreateToken(test.request)
		if test.shouldSucceed && err != nil {
			t.Fatalf("Expected test to succeed but got error: %v\n", err)
		}
		if !test.shouldSucceed && err == nil {
			t.Fatal("Expected test to fail but succeeded!")
		}
	}
}
