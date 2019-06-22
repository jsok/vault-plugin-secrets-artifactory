package artifactory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

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

	role, err := readRole(ctx, req.Storage, roleName)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return logical.ErrorResponse("role does not exist"), nil
	}

	client, rtDetails, err := b.rtClient(ctx, req.Storage)
	if client == nil || rtDetails == nil {
		return nil, fmt.Errorf("Failed to create Artifactory client: %v\n", err)
	}

	username := role.Username
	if username == "" {
		username = generateRoleUsername(roleName, req.ID)
	}

	tokenService := rtTokenService.NewAccessTokenService(client)
	tokenService.SetArtifactoryDetails(rtDetails)
	tokenResp, err := tokenService.CreateToken(&rtTokenService.CreateTokenRequest{
		Username:    username,
		Scope:       fmt.Sprintf("member-of-groups:%s", strings.Join(role.MemberOfGroups, ",")),
		ExpiresIn:   int64(role.TTL.Seconds()),
		Refreshable: false,
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
			"username":  username,
		},
	)
	resp.Secret.TTL = time.Duration(tokenResp.ExpiresIn) * time.Second
	resp.Secret.MaxTTL = resp.Secret.MaxTTL

	return resp, nil
}

// Generate a transient username that's highly unlikely to clash
// with an existing Artifactory username.
func generateRoleUsername(role, id string) string {
	return fmt.Sprintf("vault-%s-%s", role, id)
}

const pathTokenHelpSyn = `
Create an Artifactory access token against the specified role.
`
