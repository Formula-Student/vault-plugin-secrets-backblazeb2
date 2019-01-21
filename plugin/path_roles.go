package b2

import (
    "context"
    "fmt"
    "strings"
    "time"

    "github.com/hashicorp/errwrap"
    "github.com/hashicorp/vault/logical"
    "github.com/hashicorp/vault/logical/framework"
)


// List the defined roles
func (b *backend) pathRoles() *framework.Path {
    return &framework.Path{
	Pattern: fmt.Sprintf("roles/?"),
	HelpSynopsis: "List configured roles.",

	Callbacks: map[logical.Operation]framework.OperationFunc{
	    logical.ListOperation: b.pathRolesList,
	},
    }
}

// pathRolesList lists the currently defined roles
func (b *backend) pathRolesList(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
    roles, err := b.ListRoles(ctx, req.Storage)

    if err != nil {
	return nil, err
    }

    return logical.ListResponse(roles), nil
}

// Define the CRUD functions for the roles path
func (b *backend) pathRolesCRUD() *framework.Path {
    return &framework.Path{
	Pattern: fmt.Sprintf("roles/" + framework.GenericNameRegex("role")),
	HelpSynopsis: "Configure a Backblaze B2 role.",
	HelpDescription: "Use this endpoint to set the polices for generated keys in this role.",

	Fields: map[string]*framework.FieldSchema{
	    "role": &framework.FieldSchema{
		Type: framework.TypeString,
		Description: "Role name.",
	    },
	    "capabilities": &framework.FieldSchema{
		Type: framework.TypeCommaStringSlice,
		Description: "Comma-separated list of capabilities",
	    },
	    "bucket": &framework.FieldSchema{
		Type: framework.TypeString,
		Description: "Optional bucket on which to restrict this key.",
	    },
	    "name_prefix": &framework.FieldSchema{
		Type: framework.TypeString,
		Description: "Optional prefix to further restrict a bucket key.",
	    },
	    "default_ttl": &framework.FieldSchema{
		Type: framework.TypeDurationSecond,
		Description: "Optional default TTL to apply to keys.",
	    },
	    "max_ttl": &framework.FieldSchema{
		Type: framework.TypeDurationSecond,
		Description: "Optional maximum TTL to apply to keys.",
	    },
	},

	ExistenceCheck: b.pathRoleExistsCheck,

	Callbacks: map[logical.Operation]framework.OperationFunc{
	    logical.CreateOperation: b.pathRoleWrite,
	    logical.ReadOperation: b.pathRoleRead,
	    logical.UpdateOperation: b.pathRoleWrite,
	    logical.DeleteOperation: b.pathRoleDelete,
	},
    }
}

// pathRoleExistsCheck checks to see if a role exists
func (b *backend) pathRoleExistsCheck(ctx context.Context, req *logical.Request, d *framework.FieldData) (bool, error) {
    role := d.Get("role").(string)
    if r, err := b.GetRole(ctx, req.Storage, role); err != nil || r == nil {
	return false, nil
    }

    return true, nil
}

// pathRoleRead reads information on a current role
func (b *backend) pathRoleRead(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
    role := d.Get("role").(string)

    r, err := b.GetRole(ctx, req.Storage, role)

    if err != nil {
	if err == ErrRoleNotFound {
	    return logical.ErrorResponse(err.Error()), logical.ErrInvalidRequest
	}
	return nil, err
    }

    role_data := map[string]interface{}{
	"capabilities": r.Capabilities,
	"name_prefix": r.NamePrefix,
	"bucket_name": r.BucketName,
	"prefix": r.Prefix,
	"default_ttl": r.DefaultTTL.Seconds(),
	"max_ttl": r.MaxTTL.Seconds(),
    }

    return &logical.Response{
	Data:role_data,
    }, nil
}

// pathRoleWrite creates/updates a role entry
func (b *backend) pathRoleWrite(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
    role := d.Get("role").(string)

    var r Role
    var keys []string

    keys = []string{"name_prefix", "bucket", "prefix"}

    for _, key := range keys {
        if v, ok := d.GetOk(key); ok {
            nv := strings.TrimSpace(v.(string))

            switch key {
	    case "name_prefix":
		r.NamePrefix = nv
	    case "bucket":
		r.BucketName = nv
	    case "prefix":
		r.Prefix = nv
	    }
	}
    }

    // Handle TTLs
    keys = []string{"default_ttl", "max_ttl"}

    for _, key := range keys {
	if v, ok := d.GetOk(key); ok {
	    duration := time.Duration(v.(int)) * time.Second

	    switch key {
	    case "default_ttl":
		r.DefaultTTL = duration
	    case "max_ttl":
		r.MaxTTL = duration
	    }
	}
    }


    // Handle capabilities
    if c, ok := d.GetOk("capabilities"); ok {
	r.Capabilities = c.([]string)
    }

    entry, err := logical.StorageEntryJSON("roles/"+role, &r)
    if err != nil {
	return nil, errwrap.Wrapf("failed to create storage entry: {{err}}", err)
    }

    if err := req.Storage.Put(ctx, entry); err != nil {
	return nil, errwrap.Wrapf("failed to write entry to storage: {{err}}", err)
    }

    return nil, nil
}

// pathRoleDelete deletes a role
func (b *backend) pathRoleDelete(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
    role := d.Get("role").(string)

    _, err := b.GetRole(ctx, req.Storage, role)
    if err != nil {
	if err == ErrRoleNotFound {
	    return logical.ErrorResponse(err.Error()), logical.ErrInvalidRequest
	}
	return nil, err
    }

    if err := req.Storage.Delete(ctx, "roles/"+role); err != nil {
	return nil, errwrap.Wrapf("Failed to delete role from storage: {{err}}", err)
    }

    return nil, nil
}
