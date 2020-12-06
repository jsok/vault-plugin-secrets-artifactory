package artifactory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/database/dbplugin"
	"github.com/jsok/vault-plugin-secrets-artifactory/mock"
)

func TestArtifactory(t *testing.T) {
	rtAPI := mock.Artifactory()
	ts := httptest.NewServer(http.HandlerFunc(rtAPI.HandleRequests))
	defer ts.Close()

	env := &UnitTestEnv{
		Username:    rtAPI.Username(),
		Password:    rtAPI.Password(),
		URL:         ts.URL,
		Artifactory: NewArtifactory(),
		TestUsers:   make(map[string]dbplugin.Statements),
	}

	t.Run("test type", env.TestArtifactory_Type)
	t.Run("test init", env.TestArtifactory_Init)
	t.Run("test initialize", env.TestArtifactory_Initialize)
	t.Run("test create user", env.TestArtifactory_CreateUser)
	t.Run("test revoke user", env.TestArtifactory_RevokeUser)
}

type UnitTestEnv struct {
	Username, Password, URL string
	Artifactory             *Artifactory

	TestUsers map[string]dbplugin.Statements
}

func (e *UnitTestEnv) TestArtifactory_Type(t *testing.T) {
	if tp, err := e.Artifactory.Type(); err != nil {
		t.Fatal(err)
	} else if tp != "artifactory" {
		t.Fatalf("expected 'artifactory' but received %s", tp)
	}
}

func (e *UnitTestEnv) TestArtifactory_Init(t *testing.T) {
	config := map[string]interface{}{
		"username": e.Username,
		"password": e.Password,
		"address":  e.URL,
	}
	configToStore, err := e.Artifactory.Init(context.Background(), config, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(configToStore) != len(config) {
		t.Fatalf("expected %s, received %s", config, configToStore)
	}
	for k, v := range config {
		if configToStore[k] != v {
			t.Fatalf("for %s, expected %s but received %s", k, v, configToStore[k])
		}
	}
}

func (e *UnitTestEnv) TestArtifactory_Initialize(t *testing.T) {
	config := map[string]interface{}{
		"username": e.Username,
		"password": e.Password,
		"address":  e.URL,
	}
	if err := e.Artifactory.Initialize(context.Background(), config, true); err != nil {
		t.Fatal(err)
	}
}

func (e *UnitTestEnv) TestArtifactory_CreateUser(t *testing.T) {
	statements := dbplugin.Statements{
		Creation: []string{`{"artifactory_groups": ["readers"]}`},
	}
	usernameConfig := dbplugin.UsernameConfig{
		DisplayName: "display-name",
		RoleName:    "role-name",
	}
	username, token, err := e.Artifactory.CreateUser(context.Background(), statements, usernameConfig, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if username == "" {
		t.Fatal("expected username")
	}
	if token == "" {
		t.Fatal("expected token")
	}
	e.TestUsers[username] = statements
}

func (e *UnitTestEnv) TestArtifactory_RevokeUser(t *testing.T) {
	for username, statements := range e.TestUsers {
		if err := e.Artifactory.RevokeUser(context.Background(), statements, username); err != nil {
			t.Fatal(err)
		}
	}
}
