package mock

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	rtTokenService "github.com/jsok/vault-plugin-secrets-artifactory/pkg/token"
)

const (
	superUsername = "admin"
	superPassword = "password"
)

func Artifactory() *FakeArtifactory {
	return &FakeArtifactory{
		Tokens: make(map[string]rtTokenService.ArtifactoryToken),
	}
}

type FakeArtifactory struct {
	Tokens map[string]rtTokenService.ArtifactoryToken
}

func (f *FakeArtifactory) Username() string {
	return superUsername
}

func (f *FakeArtifactory) Password() string {
	return superPassword
}

func (f *FakeArtifactory) HandleRequests(w http.ResponseWriter, r *http.Request) {
	// See if the username and password given match any expected.
	reqUsername, reqPassword, _ := r.BasicAuth()
	if !(reqUsername == superUsername && reqPassword == superPassword) {
		w.WriteHeader(401)
		return
	}

	if err := r.ParseForm(); err != nil {
		w.WriteHeader(400)
		w.Write([]byte(fmt.Sprintf("unable to read request body due to %s", err.Error())))
		return
	}
	switch r.Method {
	case http.MethodGet:
		tokens := make([]rtTokenService.ArtifactoryToken, len(f.Tokens))
		for _, v := range f.Tokens {
			tokens = append(tokens, v)
		}
		body, err := json.Marshal(&rtTokenService.GetTokensResponse{Tokens: tokens})
		if err != nil {
			w.WriteHeader(404)
			w.Write([]byte(fmt.Sprintf("Encoding mock HTTP response failed", r.Method, r.URL.Path)))
			return
		}
		w.Write(body)
		return
	case http.MethodPost:
		scope := r.Form.Get("scope")
		username := r.Form.Get("username")
		ttl, err := time.ParseDuration(r.Form.Get("expires_in") + "s")
		tokenId := "token-id-" + username
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte(fmt.Sprintf("Parsing expiry failed", r.Method, r.URL.Path)))
			return
		}
		f.Tokens[tokenId] = rtTokenService.ArtifactoryToken{
			TokenID:     tokenId,
			Issuer:      "mock",
			Subject:     "artifactory/user/" + username,
			Expiry:      time.Now().Add(ttl).Unix(),
			Refreshable: false,
			IssuedAt:    time.Now().Unix(),
		}

		body, err := json.Marshal(&rtTokenService.CreateTokenResponse{
			AccessToken: "token-" + username,
			ExpiresIn:   3600,
			Scope:       scope,
			TokenType:   "Bearer",
		})
		if err != nil {
			w.WriteHeader(404)
			w.Write([]byte(fmt.Sprintf("Encoding mock HTTP response failed", r.Method, r.URL.Path)))
			return
		}
		w.Write(body)
		return
	case http.MethodDelete:
		tokenId := r.Form.Get("token_id")
		if _, found := f.Tokens[tokenId]; found {
			delete(f.Tokens, tokenId)
			return
		}

		w.WriteHeader(404)
		w.Write([]byte(fmt.Sprintf("No token found for id: %s", tokenId)))
		return
	}
	// We received an unexpected request.
	w.WriteHeader(404)
	w.Write([]byte(fmt.Sprintf("%s to %s is unsupported", r.Method, r.URL.Path)))
}
