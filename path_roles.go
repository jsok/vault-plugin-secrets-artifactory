package artifactory

import (
	"context"
	"errors"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

func pathListRoles(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: "roles/?$",

		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.ListOperation: b.pathRoleList,
		},
	}
}

func pathRoles(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: "roles/" + framework.GenericNameRegex("name"),
		Fields: map[string]*framework.FieldSchema{
			"name": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "Name of the role",
			},

			"username": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "User name of the created access token",
			},

			"member_of_groups": &framework.FieldSchema{
				Type:        framework.TypeCommaStringSlice,
				Description: `List of groups that the token is associated with.`,
			},

			"ttl": &framework.FieldSchema{
				Type:        framework.TypeDurationSecond,
				Description: "TTL for the access token created from the role.",
			},
		},

		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.CreateOperation: b.pathRolesCreateUpdate,
			logical.ReadOperation:   b.pathRolesRead,
			logical.UpdateOperation: b.pathRolesCreateUpdate,
			logical.DeleteOperation: b.pathRolesDelete,
		},
	}
}

func (b *backend) pathRoleList(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	entries, err := req.Storage.List(ctx, "role/")
	if err != nil {
		return nil, err
	}

	return logical.ListResponse(entries), nil
}

func (b *backend) role(ctx context.Context, s logical.Storage, name string) (*roleConfig, error) {
	raw, err := s.Get(ctx, "role/"+name)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}

	role := new(roleConfig)
	if err := raw.DecodeJSON(role); err != nil {
		return nil, err
	}

	return role, nil
}

func (b *backend) pathRolesRead(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	name := d.Get("name").(string)
	if name == "" {
		return logical.ErrorResponse("missing name"), nil
	}

	role, err := b.role(ctx, req.Storage, name)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, nil
	}

	// Generate the response
	resp := &logical.Response{
		Data: map[string]interface{}{
			"username":         role.Username,
			"member_of_groups": role.MemberOfGroups,
			"ttl":              int64(role.TTL.Seconds()),
		},
	}
	return resp, nil
}

func (b *backend) pathRolesCreateUpdate(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	roleName := d.Get("name").(string)
	if roleName == "" {
		return logical.ErrorResponse("missing role name"), nil
	}

	// Check if the role already exists
	role, err := b.role(ctx, req.Storage, roleName)
	if err != nil {
		return nil, err
	}

	// Create a new entry object if this is a CreateOperation
	if role == nil {
		if req.Operation == logical.UpdateOperation {
			return nil, errors.New("role entry not found during update operation")
		}
		role = new(roleConfig)
	}

	if username, ok := d.GetOk("username"); ok {
		role.Username = username.(string)
	}
	if memberOfGroups, ok := d.GetOk("member_of_groups"); ok {
		role.MemberOfGroups = memberOfGroups.([]string)
	}
	if len(role.MemberOfGroups) == 0 {
		return logical.ErrorResponse("member_of_groups cannot be empty"), nil
	}

	if tokenTTLRaw, ok := d.GetOk("ttl"); ok {
		role.TTL = time.Duration(tokenTTLRaw.(int)) * time.Second
	} else if req.Operation == logical.CreateOperation {
		role.TTL = time.Duration(d.Get("ttl").(int)) * time.Second
	}

	entry, err := logical.StorageEntryJSON("role/"+roleName, role)
	if err != nil {
		return nil, err
	}
	if err := req.Storage.Put(ctx, entry); err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *backend) pathRolesDelete(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	name := d.Get("name").(string)
	if err := req.Storage.Delete(ctx, "role/"+name); err != nil {
		return nil, err
	}
	return nil, nil
}

type roleConfig struct {
	Username       string        `json:"username"`
	MemberOfGroups []string      `json:"member_of_groups"`
	TTL            time.Duration `json:"lease"`
}
