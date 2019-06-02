package artifactory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	rtAuth "github.com/jfrog/jfrog-client-go/artifactory/auth"
	rtHttpClient "github.com/jfrog/jfrog-client-go/artifactory/httpclient"

	rtTokenService "github.com/jsok/vault-plugin-secrets-artifactory/pkg/token"
)

func pathToken(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: "token/" + framework.GenericNameRegex("name"),
		Fields: map[string]*framework.FieldSchema{
			"name": {
				Type:        framework.TypeString,
				Description: "The name of the role.",
			},
		},
		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.ReadOperation: b.pathTokenRead,
		},
		HelpSynopsis: pathTokenHelpSyn,
	}
}

func (b *backend) pathTokenRead(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	roleName := d.Get("name").(string)
	if roleName == "" {
		return nil, errors.New("role name is required")
	}

	role, err := b.role(ctx, req.Storage, roleName)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return logical.ErrorResponse("role does not exist"), nil
	}

	config, userErr, intErr := b.readConfig(ctx, req.Storage)
	if intErr != nil {
		return nil, intErr
	}
	if userErr != nil {
		return logical.ErrorResponse(userErr.Error()), nil
	}
	if config == nil {
		return nil, fmt.Errorf("Artifactory configuration not found")
	}

	rtDetails := rtAuth.NewArtifactoryDetails()
	rtDetails.SetUrl(config.Address)
	rtDetails.SetApiKey(config.ApiKey)

	client, clientErr := rtHttpClient.ArtifactoryClientBuilder().
		SetInsecureTls(true).
		SetArtDetails(&rtDetails).
		Build()
	if clientErr != nil {
		return nil, fmt.Errorf("Failed to create Artifactory client: %v\n", clientErr)
	}

	tokenService := rtTokenService.NewAccessTokenService(client)
	tokenService.SetArtifactoryDetails(rtDetails)
	tokenResp, err := tokenService.CreateToken(&rtTokenService.CreateTokenRequest{
		Username:  role.Username,
		Scope:     fmt.Sprintf("member-of-groups:%s", strings.Join(role.MemberOfGroups, ",")),
		ExpiresIn: int64(role.TTL.Seconds()),
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to create access token: %v\n", err)
	}

	resp := b.Secret(accessTokenSecretType).Response(
		map[string]interface{}{
			"access_token": tokenResp.AccessToken,
			"scope":        tokenResp.Scope,
			"token_type":   tokenResp.TokenType,
		},
		map[string]interface{}{
			"role_name": roleName,
		},
	)
	resp.Secret.TTL = time.Duration(tokenResp.ExpiresIn) * time.Second
	resp.Secret.MaxTTL = resp.Secret.MaxTTL

	return resp, nil
}

const pathTokenHelpSyn = `
Create an Artifactory access token against the specified role.
`
