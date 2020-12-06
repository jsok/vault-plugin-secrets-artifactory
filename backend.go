package artifactory

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	rtAuth "github.com/jfrog/jfrog-client-go/artifactory/auth"
	rtHttpClient "github.com/jfrog/jfrog-client-go/artifactory/httpclient"
)

type backend struct {
	*framework.Backend
}

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
			secretAccessToken(&b),
		},

		BackendType: logical.TypeLogical,
	}

	return &b
}

func (b *backend) rtClient(ctx context.Context, s logical.Storage) (*rtHttpClient.ArtifactoryHttpClient, rtAuth.ArtifactoryDetails, error) {
	config, err := b.readConfig(ctx, s)
	if err != nil {
		return nil, nil, err
	}

	rtDetails := rtAuth.NewArtifactoryDetails()
	// Ensure trailing slash, rtClient assumes this when building URLs
	rtDetails.SetUrl(strings.TrimSuffix(config.Address, "/") + "/")
	rtDetails.SetApiKey(config.ApiKey)
	rtDetails.SetUser(config.Username)
	rtDetails.SetPassword(config.Password)

	client, err := rtHttpClient.ArtifactoryClientBuilder().
		SetInsecureTls(!config.TlsVerify).
		SetArtDetails(&rtDetails).
		Build()
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create Artifactory client: %v\n", err)
	}

	return client, rtDetails, nil
}
