package tests

import (
	"context"
	"testing"
	"time"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/meta"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway"
	"github.com/stretchr/testify/require"
)

type TestTools struct {
	T                 *testing.T
	Meta              *meta.Meta
	ProviderFactories map[string]func() (*schema.Provider, error)
	Cleanup           func()
}

func NewTestTools(t *testing.T) *TestTools {
	t.Helper()
	ctx := context.Background()
	// Create a http client with recording capabilities
	httpClient, cleanup, err := scaleway.getHTTPRecoder(t, *scaleway.UpdateCassettes)
	require.NoError(t, err)

	// Create meta that will be passed in the provider config
	meta, err := scaleway.buildMeta(ctx, &scaleway.metaConfig{
		providerSchema:   nil,
		terraformVersion: "terraform-tests",
		httpClient:       httpClient,
	})
	require.NoError(t, err)

	if !*scaleway.UpdateCassettes {
		tmp := 0 * time.Second
		scaleway.DefaultWaitRetryInterval = &tmp
	}

	return &TestTools{
		T:    t,
		Meta: meta,
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"scaleway": func() (*schema.Provider, error) {
				return scaleway.Provider(&scaleway.ProviderConfig{Meta: meta})(), nil
			},
		},
		Cleanup: cleanup,
	}
}

func TestAccPreCheck(_ *testing.T) {}
