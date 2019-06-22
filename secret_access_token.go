package artifactory

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	rtTokenService "github.com/jsok/vault-plugin-secrets-artifactory/pkg/token"
)

const accessTokenSecretType = "artifactory_access_token"

func secretAccessToken(b *backend) *framework.Secret {
	return &framework.Secret{
		Type: accessTokenSecretType,
		Fields: map[string]*framework.FieldSchema{
			"access_token": {
				Type:        framework.TypeString,
				Description: "Artifactory Access Token",
			},
		},
		Revoke: b.secretAccessTokenRevoke,
	}
}

func (b *backend) secretAccessTokenRevoke(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	accessToken := d.Get("access_token").(string)

	client, rtDetails, err := b.rtClient(ctx, req.Storage)
	if client == nil || rtDetails == nil {
		return nil, fmt.Errorf("Failed to create Artifactory client: %v\n", err)
	}

	tokenService := rtTokenService.NewAccessTokenService(client)
	tokenService.SetArtifactoryDetails(rtDetails)
	err = tokenService.RevokeToken(&rtTokenService.RevokeTokenRequest{Token: accessToken})
	if err != nil {
		return nil, fmt.Errorf("Failed to revoke token:\n%v\n", err)
	}

	return nil, nil
}
