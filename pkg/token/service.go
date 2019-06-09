package token

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/jfrog/jfrog-client-go/artifactory/auth"
	rtHttpClient "github.com/jfrog/jfrog-client-go/artifactory/httpclient"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	clientutils "github.com/jfrog/jfrog-client-go/utils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
)

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

	httpClientDetails := rtDetails.CreateHttpClientDetails()
	httpClientDetails.Headers["Content-Type"] = "application/x-www-form-urlencoded"

	data := url.Values{}
	if req.Username != "" {
		data.Set("username", req.Username)
	}
	if req.Scope != "" {
		data.Set("scope", req.Scope)
	}
	data.Set("expires_in", fmt.Sprintf("%v", req.ExpiresIn))
	data.Set("refreshable", fmt.Sprintf("%v", req.Refreshable))

	resp, body, err := s.client.SendPost(reqUrl, []byte(data.Encode()), &httpClientDetails)
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

	httpClientDetails := rtDetails.CreateHttpClientDetails()
	httpClientDetails.Headers["Content-Type"] = "application/x-www-form-urlencoded"

	data := url.Values{}
	data.Set("token", req.Token)
	data.Set("token_id", req.TokenID)

	resp, body, err := s.client.SendPost(reqUrl, []byte(data.Encode()), &httpClientDetails)
	if err != nil {
		return err
	}

	// This usually means that the token is not revocable
	if resp.StatusCode == http.StatusInternalServerError {
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		return errorutils.CheckError(errors.New("Artifactory response: " + resp.Status + "\n" + clientutils.IndentJson(body)))
	}

	return nil
}
