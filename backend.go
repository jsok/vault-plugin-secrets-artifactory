package artifactory

import (
	"context"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const accessTokenSecretType = "artifactory_access_token"

func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b := Backend()
	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}
	return b, nil
}

func Backend() *backend {
	var b backend
	b.Backend = &framework.Backend{
		PathsSpecial: &logical.Paths{
			SealWrapStorage: []string{
				"config",
			},
		},

		Paths: []*framework.Path{
			pathConfig(&b),
			pathListRoles(&b),
			pathRoles(&b),
			pathToken(&b),
		},

		Secrets: []*framework.Secret{
			&framework.Secret{
				Type: accessTokenSecretType,
				Fields: map[string]*framework.FieldSchema{
					"access_token": {
						Type:        framework.TypeString,
						Description: "Artifactory Access Token",
					},
				},
			},
		},

		BackendType: logical.TypeLogical,
	}

	return &b
}

type backend struct {
	*framework.Backend
}
