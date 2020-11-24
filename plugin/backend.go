package b2

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	b2client "github.com/kurin/blazer/b2"
)

type backend struct {
	*framework.Backend

	client *b2client.Client

	// We're going to have to be able to rotate the client
	// if the mount configured credentials change, use
	// this to protect it
	clientMutex *sync.RWMutex
}

var _ logical.Factory = Factory

// Factory returns a configured instance of the B2 backend
func Factory(ctx context.Context, c *logical.BackendConfig) (logical.Backend, error) {
	b, err := newBackend()
	if err != nil {
		return nil, err
	}

	if c == nil {
		return nil, fmt.Errorf("configuration passed into backend is nil")
	}

	if err := b.Setup(ctx, c); err != nil {
		return nil, err
	}

	b.Logger().Info("Plugin successfully initialized")
	return b, nil
}

func newBackend() (*backend, error) {
	b := &backend{
		client:      (*b2client.Client)(nil),
		clientMutex: new(sync.RWMutex),
	}

	b.Backend = &framework.Backend{
		BackendType: logical.TypeLogical,
		Help:        "The B2 secrets backend provisions API keys for the Backblaze B2 service",
		Paths: framework.PathAppend(
			[]*framework.Path{
				// path_config.go
				// ^config
				b.pathConfigCRUD(),

				// path_config_rotate.go
				// ^config/rotate
				b.pathConfigRotate(),

				// path_roles.go
				// ^roles (LIST)
				b.pathRoles(),
				// ^roles/<role>
				b.pathRolesCRUD(),

				// path_keys.go
				// ^keys/<role>
				b.pathKeysRead(),
			},
		),

		Secrets: []*framework.Secret{
			b.b2ApplicationsKeys(),
		},
	}

	return b, nil
}
