package tem

import (
	"context"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/transport"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
	"time"

	meta2 "github.com/scaleway/terraform-provider-scaleway/v2/internal/meta"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	tem "github.com/scaleway/scaleway-sdk-go/api/tem/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

const (
	DefaultTemDomainTimeout       = 5 * time.Minute
	defaultTemDomainRetryInterval = 15 * time.Second
)

// TemAPIWithRegion returns a new Tem API and the region for a Create request
func TemAPIWithRegion(d *schema.ResourceData, m interface{}) (*tem.API, scw.Region, error) {
	meta := m.(*meta2.Meta)
	api := tem.NewAPI(meta.GetScwClient())

	region, err := locality.ExtractRegion(d, meta)
	if err != nil {
		return nil, "", err
	}
	return api, region, nil
}

// TemAPIWithRegionAndID returns a Tem API with zone and ID extracted from the state
func TemAPIWithRegionAndID(m interface{}, id string) (*tem.API, scw.Region, string, error) {
	meta := m.(*meta2.Meta)
	api := tem.NewAPI(meta.GetScwClient())

	region, id, err := locality.ParseRegionalID(id)
	if err != nil {
		return nil, "", "", err
	}
	return api, region, id, nil
}

func WaitForTemDomain(ctx context.Context, api *tem.API, region scw.Region, id string, timeout time.Duration) (*tem.Domain, error) {
	retryInterval := defaultTemDomainRetryInterval
	if transport.DefaultWaitRetryInterval != nil {
		retryInterval = *transport.DefaultWaitRetryInterval
	}

	domain, err := api.WaitForDomain(&tem.WaitForDomainRequest{
		Region:        region,
		DomainID:      id,
		RetryInterval: &retryInterval,
		Timeout:       scw.TimeDurationPtr(timeout),
	}, scw.WithContext(ctx))

	return domain, err
}

func flattenDomainReputation(reputation *tem.DomainReputation) interface{} {
	if reputation == nil {
		return nil
	}
	return []map[string]interface{}{
		{
			"status":             reputation.Status.String(),
			"score":              reputation.Score,
			"scored_at":          types.FlattenTime(reputation.ScoredAt),
			"previous_score":     types.FlattenUint32Ptr(reputation.PreviousScore),
			"previous_scored_at": types.FlattenTime(reputation.PreviousScoredAt),
		},
	}
}
