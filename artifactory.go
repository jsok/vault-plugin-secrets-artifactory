package artifactory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/database/dbplugin"
	"github.com/hashicorp/vault/sdk/database/helper/connutil"
	"github.com/hashicorp/vault/sdk/database/helper/credsutil"
	"github.com/hashicorp/vault/sdk/database/helper/dbutil"

	rtAuth "github.com/jfrog/jfrog-client-go/artifactory/auth"
	rtHttpClient "github.com/jfrog/jfrog-client-go/artifactory/httpclient"

	"github.com/mitchellh/mapstructure"

	rtTokenService "github.com/jsok/vault-plugin-secrets-artifactory/pkg/token"
)

// Artifactory implements dbplugin's Database interface.
type Artifactory struct {
	Address     string `json:"address" structs:"address" mapstructure:"address"`
	ApiKey      string `json:"api_key" structs:"api_key" mapstructure:"api_key"`
	Username    string `json:"username" structs:"username" mapstructure:"username"`
	Password    string `json:"password" structs:"password" mapstructure:"password"`
	InsecureTls bool   `json:"insecure_tls" structs:"insecure_tls" mapstructure:"insecure_tls"`

	// The CredentialsProducer is never mutated and thus is inherently thread-safe.
	CredentialsProducer credsutil.CredentialsProducer

	// This protects the config from races while also allowing multiple threads
	// to read the config simultaneously when it's not changing.
	mux sync.RWMutex

	Initialized bool

	rawConfig    map[string]interface{}
	tokenService *rtTokenService.AccessTokenService
}

func New() (interface{}, error) {
	db := NewArtifactory()
	// Wrap the plugin with middleware to sanitize errors
	dbType := dbplugin.NewDatabaseErrorSanitizerMiddleware(db, db.SecretValues)
	return dbType, nil
}

// Run instantiates an Artifactory object, and runs the RPC server for the plugin
func Run(apiTLSConfig *api.TLSConfig) error {
	dbType, err := New()
	if err != nil {
		return err
	}

	dbplugin.Serve(dbType.(dbplugin.Database), api.VaultPluginTLSProvider(apiTLSConfig))

	return nil
}

func NewArtifactory() *Artifactory {
	return &Artifactory{
		CredentialsProducer: &credsutil.SQLCredentialsProducer{
			DisplayNameLen: 15,
			RoleNameLen:    15,
			UsernameLen:    100,
			Separator:      "-",
		},
	}
}

func (rt *Artifactory) Initialize(ctx context.Context, conf map[string]interface{}, verifyConnection bool) error {
	_, err := rt.Init(ctx, conf, verifyConnection)
	return err
}

// Init is called on `$ vault write database/config/:db-name`,
// or when you do a creds call after Vault's been restarted.
func (rt *Artifactory) Init(ctx context.Context, conf map[string]interface{}, verifyConnection bool) (map[string]interface{}, error) {
	rt.mux.RLock()
	defer rt.mux.RUnlock()

	rt.rawConfig = conf

	err := mapstructure.WeakDecode(conf, rt)
	if err != nil {
		return nil, err
	}

	if rt.Address == "" {
		return nil, fmt.Errorf("Address cannot be empty")
	}

	if rt.ApiKey != "" && rt.Username != "" {
		return nil, fmt.Errorf("provide either api_key or username, not both")
	}

	if rt.Username != "" {
		if rt.Password == "" {
			return nil, fmt.Errorf("must provide password with username")
		}
	} else if rt.ApiKey == "" {
		return nil, fmt.Errorf("api_key must be set")
	}

	// Set initialized to true at this point since all fields are set,
	// and the connection can be established at a later time.
	rt.Initialized = true

	// Test the given config to see if we can make a client.
	client, err := rt.Connection(ctx)
	if err != nil {
		return nil, errwrap.Wrapf("couldn't make client with config: {{err}}", err)
	}

	if verifyConnection {
		if _, err := client.GetTokens(); err != nil {
			return nil, errwrap.Wrapf("error verifying connection: {{err}}", err)
		}
	}

	return conf, nil
}

func (rt *Artifactory) Connection(_ context.Context) (*rtTokenService.AccessTokenService, error) {
	if !rt.Initialized {
		return nil, connutil.ErrNotInitialized
	}

	// If we already have a DB, return it
	if rt.tokenService != nil {
		return rt.tokenService, nil
	}

	rtDetails := rtAuth.NewArtifactoryDetails()
	rtDetails.SetUrl(strings.TrimSuffix(rt.Address, "/") + "/")
	rtDetails.SetApiKey(rt.ApiKey)
	rtDetails.SetUser(rt.Username)
	rtDetails.SetPassword(rt.Password)

	client, err := rtHttpClient.ArtifactoryClientBuilder().
		SetInsecureTls(rt.InsecureTls).
		SetArtDetails(&rtDetails).
		Build()
	if err != nil {
		return nil, fmt.Errorf("Failed to create Artifactory client: %v\n", err)
	}

	//  Store the session in backend for reuse
	rt.tokenService = rtTokenService.NewAccessTokenService(client)
	rt.tokenService.SetArtifactoryDetails(rtDetails)

	return rt.tokenService, nil
}

// SecretValues is used by some error-sanitizing middleware in Vault that basically
// replaces the keys in the map with the values given so they're not leaked via
// error messages.
func (rt *Artifactory) SecretValues() map[string]interface{} {
	rt.mux.RLock()
	defer rt.mux.RUnlock()

	replacements := make(map[string]interface{})

	if rt.Password != "" {
		replacements[rt.Password] = "[password]"
	}

	if rt.ApiKey != "" {
		replacements[rt.ApiKey] = "[api_key]"
	}

	return replacements
}

// Json structure to capture Artifactory specific creation info from the creation statements section
type creationStatement struct {
	ArtifactoryGroups []string `json:"artifactory_groups"`
}

func getArtifactoryGroups(statements dbplugin.Statements) ([]string, error) {
	if len(statements.Creation) == 0 {
		return nil, dbutil.ErrEmptyCreationStatement
	}

	stmt := &creationStatement{}
	if err := json.Unmarshal([]byte(statements.Creation[0]), stmt); err != nil {
		return nil, errwrap.Wrapf(fmt.Sprintf("unable to unmarshal %s: {{err}}", []byte(statements.Creation[0])), err)
	}

	if len(stmt.ArtifactoryGroups) == 0 {
		return nil, errors.New("No artifactory group was provided in the creation statement")
	}
	return stmt.ArtifactoryGroups, nil
}

func (rt *Artifactory) Type() (string, error) {
	return "artifactory", nil
}

// CreateUser is called on `$ vault read database/creds/:role-name`
// and it's the first time anything is touched from `$ vault write database/roles/:role-name`.
// This is likely to be the highest-throughput method for this plugin.
func (rt *Artifactory) CreateUser(ctx context.Context, statements dbplugin.Statements, usernameConfig dbplugin.UsernameConfig, expiration time.Time) (string, string, error) {
	username, err := rt.CredentialsProducer.GenerateUsername(usernameConfig)
	if err != nil {
		return "", "", errwrap.Wrapf(fmt.Sprintf("unable to generate username for %q: {{err}}", usernameConfig), err)
	}

	groups, err := getArtifactoryGroups(statements)
	if err != nil {
		return "", "", err
	}

	request := &rtTokenService.CreateTokenRequest{
		GrantType:   "client_credentials",
		Username:    username,
		Scope:       fmt.Sprintf(`member-of-groups:"%s"`, strings.Join(groups, ",")),
		ExpiresIn:   int64(expiration.Sub(time.Now()).Seconds()),
		Refreshable: false,
	}

	// Don't let anyone write the config while we're using it for our current client.
	rt.mux.RLock()
	defer rt.mux.RUnlock()

	service, err := rt.Connection(ctx)
	if err != nil {
		return "", "", errwrap.Wrapf("unable to get client: {{err}}", err)
	}

	token, err := service.CreateToken(request)
	if err != nil {
		return "", "", errwrap.Wrapf(fmt.Sprintf("unable to create token name %s, user %v: {{err}}", username, request), err)
	}

	return username, token.AccessToken, nil
}

// RenewUser gets called on `$ vault lease renew {{lease-id}}`. It automatically pushes out the amount of time until
// the database secrets engine calls RevokeUser, if appropriate.
func (rt *Artifactory) RenewUser(_ context.Context, _ dbplugin.Statements, _ string, _ time.Time) error {
	// Normally, this function would update a "VALID UNTIL" statement on a database user
	// but there's no similar need here.
	return nil
}

// RevokeUser is called when a lease expires.
func (rt *Artifactory) RevokeUser(ctx context.Context, statements dbplugin.Statements, username string) error {
	// Don't let anyone write the config while we're using it for our current client.
	rt.mux.RLock()
	defer rt.mux.RUnlock()

	tokenID, err := rt.tokenService.LookupTokenID(username)
	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to revoke token for %s. Couldn't lookup token id: {{err}}", username), err)
	}

	if tokenID == nil {
		return nil
	}

	service, err := rt.Connection(ctx)
	if err != nil {
		return errwrap.Wrapf("unable to get client: {{err}}", err)
	}

	service.RevokeToken(&rtTokenService.RevokeTokenRequest{TokenID: *tokenID})
	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf("failed to revoke token for %s: {{err}}", username), err)
	}
	return nil
}

// GenerateCredentials returns a generated password
func (rt *Artifactory) GenerateCredentials(ctx context.Context) (string, error) {
	password, err := rt.CredentialsProducer.GeneratePassword()
	if err != nil {
		return "", err
	}
	return password, nil
}

func (rt *Artifactory) Close() error {
	// NOOP, nothing to close.
	return nil
}

// RotateRootCredentials is useful when we try to change root credential.
// This is not currently supported by the artifactory plugin, but is needed
// to conform to the dbplugin.Database interface.
func (rt *Artifactory) RotateRootCredentials(ctx context.Context, statements []string) (map[string]interface{}, error) {
	return nil, dbutil.Unimplemented()
}

// SetCredentials is used to set the credentials for a database user to a
// specific username and password. This is not currently supported by the
// artifactory plugin, but is needed to conform to the dbplugin.Database
// interface.
func (rt *Artifactory) SetCredentials(ctx context.Context, statements dbplugin.Statements, staticConfig dbplugin.StaticUserConfig) (username string, password string, err error) {
	return "", "", dbutil.Unimplemented()
}
