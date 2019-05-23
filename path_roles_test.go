package artifactory

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
)

func TestRole_Create(t *testing.T) {
	tests := []struct {
		expectation Expectation
		name        string
		data        map[string]interface{}
	}{
		{
			ExpectedToSucceed,
			"role",
			map[string]interface{}{
				"username":         "role1-user",
				"member_of_groups": "role1-group",
				"ttl":              "10h",
			},
		},
		{
			ExpectedToSucceed,
			"role-without-username",
			map[string]interface{}{
				"member_of_groups": "role1-group",
			},
		},
		{
			FailWithLogicalError,
			"role-with-invalid-ttl",
			map[string]interface{}{
				"member_of_groups": "group",
				"ttl":              "invalid",
			},
		},
		{
			FailWithLogicalError,
			"role-without-groups",
			map[string]interface{}{
				"username":         "user",
				"member_of_groups": "",
			},
		},
	}

	for _, test := range tests {
		b, storage := newBackend(t)

		req := &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "roles/" + test.name,
			Storage:   storage,
			Data:      test.data,
		}

		resp, err := b.HandleRequest(context.Background(), req)
		assertLogicalResponse(t, test.expectation, err, resp)
	}
}

func TestRole_Update(t *testing.T) {
	b, storage := newBackend(t)

	req := &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "roles/test",
		Storage:   storage,
		Data:      map[string]interface{}{"username": "user"},
	}
	resp, err := b.HandleRequest(context.Background(), req)
	// Updating a role that doesn't exist should fail
	assertLogicalResponse(t, FailWithError, err, resp)
}

func TestRole_Lifecycle(t *testing.T) {
	b, storage := newBackend(t)

	roleData := map[string]interface{}{
		"username":         "user",
		"member_of_groups": "group",
		"ttl":              "10h",
	}

	req := &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "roles/test",
		Storage:   storage,
		Data:      roleData,
	}
	resp, err := b.HandleRequest(context.Background(), req)
	assertLogicalResponse(t, ExpectedToSucceed, err, resp)

	req = &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "roles/test",
		Storage:   storage,
	}
	resp, err = b.HandleRequest(context.Background(), req)
	assertLogicalResponse(t, ExpectedToSucceed, err, resp)

	req = &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "roles/test",
		Storage:   storage,
		Data: map[string]interface{}{
			"ttl":              "24h",
			"member_of_groups": "different-group,extra-group",
		},
	}
	resp, err = b.HandleRequest(context.Background(), req)
	assertLogicalResponse(t, ExpectedToSucceed, err, resp)

	req = &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "roles/test",
		Storage:   storage,
	}
	resp, err = b.HandleRequest(context.Background(), req)
	assertLogicalResponse(t, ExpectedToSucceed, err, resp)

	if len(resp.Data["member_of_groups"].([]string)) != 2 {
		t.Fatalf("member_of_groups not updated, expected 2 groups got: %v\n", resp.Data)
	}
	if resp.Data["ttl"].(int64) != int64((24 * time.Hour).Seconds()) {
		t.Fatalf("ttl not updated, expected 24h, got: %v\n", resp.Data)
	}

	req = &logical.Request{
		Operation: logical.DeleteOperation,
		Path:      "roles/test",
		Storage:   storage,
	}
	resp, err = b.HandleRequest(context.Background(), req)
	assertLogicalResponse(t, ExpectedToSucceed, err, resp)

	req = &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "roles/test",
		Storage:   storage,
	}
	resp, err = b.HandleRequest(context.Background(), req)
	assertLogicalResponse(t, FailWithLogicalError, err, resp)
}

func TestRole_List(t *testing.T) {
	roles := []struct {
		name string
		data map[string]interface{}
	}{
		{
			"role1",
			map[string]interface{}{
				"username":         "role1-user",
				"member_of_groups": "role1-group",
			},
		},
		{
			"role2",
			map[string]interface{}{
				"username":         "role2-user",
				"member_of_groups": "role2-group",
			},
		},
	}

	b, storage := newBackend(t)

	req := &logical.Request{
		Operation: logical.ListOperation,
		Path:      "roles",
		Storage:   storage,
	}
	resp, err := b.HandleRequest(context.Background(), req)
	assertLogicalResponse(t, ExpectedToSucceed, err, resp)
	if resp.Data["keys"] != nil {
		t.Fatalf("Did not expect any roles to exist!\n%v\n", resp.Data)
	}

	for _, role := range roles {
		req := &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "roles/" + role.name,
			Storage:   storage,
			Data:      role.data,
		}
		resp, err := b.HandleRequest(context.Background(), req)
		assertLogicalResponse(t, ExpectedToSucceed, err, resp)
	}

	req = &logical.Request{
		Operation: logical.ListOperation,
		Path:      "roles",
		Storage:   storage,
	}
	resp, err = b.HandleRequest(context.Background(), req)
	assertLogicalResponse(t, ExpectedToSucceed, err, resp)

	if resp.Data["keys"] == nil {
		t.Fatal("Listing did not return any roles")
	}
	actualCount := len(resp.Data["keys"].([]string))
	expectedCount := len(roles)
	if expectedCount != actualCount {
		t.Fatalf("Incorrect role count listed. Expected: %d Actual: %d\nresp=%v\n", expectedCount, actualCount, resp.Data)
	}
}
