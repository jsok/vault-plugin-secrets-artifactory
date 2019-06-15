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
			"username": {
				Type:        framework.TypeString,
				Description: "Username which will be used to create access tokens",
			},
			"password": {
				Type:        framework.TypeString,
				Description: "Password of the user which will be used to create access tokens",
			},
			"tls_verify": {
				Type:        framework.TypeBool,
				Description: "Disable TLS verification of Artifactory server",
				Default:     true,
			},
		},
		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.ReadOperation:   b.pathConfigRead,
			logical.UpdateOperation: b.pathConfigWrite,
		},
		HelpSynopsis: pathConfigRootHelpSyn,
	}
}

func (b *backend) readConfig(ctx context.Context, storage logical.Storage) (*accessConfig, error) {
	entry, err := storage.Get(ctx, "config")
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}

	conf := &accessConfig{}
	if err := entry.DecodeJSON(conf); err != nil {
		return nil, fmt.Errorf("error reading artifactory configuration: %v", err)
	}

	return conf, nil
}

func (b *backend) pathConfigRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	conf, err := b.readConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if conf == nil {
		return nil, fmt.Errorf("No artifactory configuration found")
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"address": conf.Address,
		},
	}, nil
}

func (b *backend) pathConfigWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	config := accessConfig{
		Address:   data.Get("address").(string),
		ApiKey:    data.Get("api_key").(string),
		Username:  data.Get("username").(string),
		Password:  data.Get("password").(string),
		TlsVerify: data.Get("tls_verify").(bool),
	}
	if config.Address == "" {
		return logical.ErrorResponse("address must be set"), nil
	}
	if config.ApiKey != "" && config.Username != "" {
		return logical.ErrorResponse("provide either api_key or username, not both"), nil
	}

	if config.Username != "" {
		if config.Password == "" {
			return logical.ErrorResponse("must provide password with username"), nil
		}
	} else if config.ApiKey == "" {
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
	Address   string `json:"address"`
	ApiKey    string `json:"api_key"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	TlsVerify bool   `json:"tls_verify"`
}

const pathConfigRootHelpSyn = `
Configure the address and API key to access the Artifactory server.
`
