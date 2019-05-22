package artifactory

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

func pathConfig(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: "config",
		Fields: map[string]*framework.FieldSchema{
			"address": {
				Type:        framework.TypeString,
				Description: "Artifactory server address",
			},
			"api_key": {
				Type:        framework.TypeString,
				Description: "API Key to use to create access tokens",
			},
		},
		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.ReadOperation:   b.pathConfigRead,
			logical.UpdateOperation: b.pathConfigWrite,
		},
		HelpSynopsis: pathConfigRootHelpSyn,
	}
}

func (b *backend) readConfig(ctx context.Context, storage logical.Storage) (*accessConfig, error, error) {
	entry, err := storage.Get(ctx, "config")
	if err != nil {
		return nil, nil, err
	}
	if entry == nil {
		return nil, fmt.Errorf("Backend hasn't been configured; please configure it at the '/config' endpoint"), nil
	}

	conf := &accessConfig{}
	if err := entry.DecodeJSON(conf); err != nil {
		return nil, nil, fmt.Errorf("error reading artifactory configuration: %v", err)
	}

	return conf, nil, nil
}

func (b *backend) pathConfigRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	conf, userErr, intErr := b.readConfig(ctx, req.Storage)
	if intErr != nil {
		return nil, intErr
	}
	if userErr != nil {
		return logical.ErrorResponse(userErr.Error()), nil
	}
	if conf == nil {
		return nil, fmt.Errorf("no user error reported but artifactory configuration not found")
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"address": conf.Address,
		},
	}, nil
}

func (b *backend) pathConfigWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	config := accessConfig{
		Address: data.Get("address").(string),
		ApiKey:  data.Get("api_key").(string),
	}
	if config.Address == "" {
		return logical.ErrorResponse("address must be set"), nil
	}
	if config.ApiKey == "" {
		return logical.ErrorResponse("api_key must be set"), nil
	}

	entry, err := logical.StorageEntryJSON("config", config)
	if err != nil {
		return nil, err
	}
	if err := req.Storage.Put(ctx, entry); err != nil {
		return nil, err
	}

	return nil, nil
}

type accessConfig struct {
	Address string `json:"address"`
	ApiKey  string `json:"api_key"`
}

const pathConfigRootHelpSyn = `
Configure the address and API key to access the Artifactory server.
`
