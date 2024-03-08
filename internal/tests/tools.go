package tests

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/meta"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/provider"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/transport"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type TestTools struct {
	T                 *testing.T
	meta              *meta.Meta
	ProviderFactories map[string]func() (*schema.Provider, error)
	Cleanup           func()
}

func (tt TestTools) GetMeta() *meta.Meta {
	return tt.meta
}

func NewTestTools(t *testing.T) *TestTools {
	t.Helper()
	ctx := context.Background()
	// Create a http client with recording capabilities
	httpClient, cleanup, err := getHTTPRecoder(t, *UpdateCassettes)
	require.NoError(t, err)

	// Create meta that will be passed in the provider config
	meta, err := meta.BuildMeta(ctx, &meta.MetaConfig{
		ProviderSchema:   nil,
		TerraformVersion: "terraform-tests",
		HttpClient:       httpClient,
	})
	require.NoError(t, err)

	if !*UpdateCassettes {
		tmp := 0 * time.Second
		transport.DefaultWaitRetryInterval = &tmp
	}

	return &TestTools{
		T:    t,
		meta: meta,
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"scaleway": func() (*schema.Provider, error) {
				return provider.Provider(&provider.ProviderConfig{Meta: meta})(), nil
			},
		},
		Cleanup: cleanup,
	}
}

func TestAccPreCheck(_ *testing.T) {}
