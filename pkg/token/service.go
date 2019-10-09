package token

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/jfrog/jfrog-client-go/artifactory/auth"
	rtHttpClient "github.com/jfrog/jfrog-client-go/artifactory/httpclient"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	clientutils "github.com/jfrog/jfrog-client-go/utils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

func init() {
	log.SetLogger(log.NewLogger(log.WARN, os.Stderr))
}

type AccessTokenService struct {
	client     *rtHttpClient.ArtifactoryHttpClient
	ArtDetails auth.ArtifactoryDetails
}

type CreateTokenRequest struct {
	GrantType   string
	Username    string
	Scope       string
	ExpiresIn   int64
	Refreshable bool
}

type CreateTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
}

type RevokeTokenRequest struct {
	Token   string
	TokenID string
}

type GetTokensResponse struct {
	Tokens []ArtifactoryToken `json:"tokens"`
}

type ArtifactoryToken struct {
	TokenID     string `json:"token_id"`
	Issuer      string `json:"issuer"`
	Subject     string `json:"subject"`
	Expiry      int64  `json:"expiry"`
	Refreshable bool   `json:"refreshable"`
	IssuedAt    int64  `json:"issued_at"`
}

const tokenApiPath = "api/security/token"
const tokenRevokeApiPath = tokenApiPath + "/revoke"

func NewAccessTokenService(client *rtHttpClient.ArtifactoryHttpClient) *AccessTokenService {
	return &AccessTokenService{client: client}
}

func (s *AccessTokenService) GetArtifactoryDetails() auth.ArtifactoryDetails {
	return s.ArtDetails
}

func (s *AccessTokenService) SetArtifactoryDetails(rt auth.ArtifactoryDetails) {
	s.ArtDetails = rt
}

func (s *AccessTokenService) CreateToken(req *CreateTokenRequest) (*CreateTokenResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("Empty request")
	}

	rtDetails := s.GetArtifactoryDetails()
	reqUrl, err := utils.BuildArtifactoryUrl(rtDetails.GetUrl(), tokenApiPath, nil)
	if err != nil {
		return nil, err
	}

	data := url.Values{}
	if req.Username != "" {
		data.Set("username", req.Username)
	}
	if req.Scope != "" {
		data.Set("scope", req.Scope)
	}
	data.Set("expires_in", fmt.Sprintf("%v", req.ExpiresIn))
	data.Set("refreshable", fmt.Sprintf("%v", req.Refreshable))
	log.Debug("Sending HTTP POST Form data: ", data.Encode())

	httpClientDetails := rtDetails.CreateHttpClientDetails()
	resp, body, err := s.client.SendPostForm(reqUrl, data, &httpClientDetails)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errorutils.CheckError(errors.New("Artifactory response: " + resp.Status + "\n" + clientutils.IndentJson(body)))
	}

	tokenResp := &CreateTokenResponse{}
	if err := json.Unmarshal(body, tokenResp); err != nil {
		return nil, err
	}

	return tokenResp, nil
}

func (s *AccessTokenService) RevokeToken(req *RevokeTokenRequest) error {
	if req.Token == "" && req.TokenID == "" {
		return fmt.Errorf("Empty request")
	}

	rtDetails := s.GetArtifactoryDetails()
	reqUrl, err := utils.BuildArtifactoryUrl(rtDetails.GetUrl(), tokenRevokeApiPath, nil)
	if err != nil {
		return err
	}

	data := url.Values{}
	data.Set("token", req.Token)
	data.Set("token_id", req.TokenID)
	log.Debug("Sending HTTP POST Form data: ", data.Encode())

	httpClientDetails := rtDetails.CreateHttpClientDetails()
	resp, body, err := s.client.SendPostForm(reqUrl, data, &httpClientDetails)
	if err != nil {
		return err
	}

	// This usually means that the token is not revocable
	if resp.StatusCode == http.StatusInternalServerError {
		log.Info("Revoke Token failed, token may not be recovable")
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		return errorutils.CheckError(errors.New("Artifactory response: " + resp.Status + "\n" + clientutils.IndentJson(body)))
	}

	return nil
}

func (s *AccessTokenService) GetTokens() ([]ArtifactoryToken, error) {
	rtDetails := s.GetArtifactoryDetails()
	reqUrl, err := utils.BuildArtifactoryUrl(rtDetails.GetUrl(), tokenApiPath, nil)
	if err != nil {
		return nil, err
	}

	httpClientDetails := rtDetails.CreateHttpClientDetails()
	resp, body, _, err := s.client.SendGet(reqUrl, true, &httpClientDetails)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errorutils.CheckError(errors.New("Artifactory response: " + resp.Status + "\n" + clientutils.IndentJson(body)))
	}

	tokenResp := &GetTokensResponse{}
	if err := json.Unmarshal(body, tokenResp); err != nil {
		return nil, err
	}

	return tokenResp.Tokens, nil
}

func (s *AccessTokenService) LookupTokenID(username string) (*string, error) {
	tokens, err := s.GetTokens()
	if err != nil {
		return nil, err
	}

	for _, token := range tokens {
		if strings.HasSuffix(token.Subject, "/"+username) {
			return &token.TokenID, nil
		}
	}

	return nil, nil
}
