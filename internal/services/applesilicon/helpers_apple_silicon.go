package applesilicon

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	applesilicon "github.com/scaleway/scaleway-sdk-go/api/applesilicon/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	meta2 "github.com/scaleway/terraform-provider-scaleway/v2/internal/meta"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/transport"
	"time"
)

const (
	defaultAppleSiliconServerTimeout       = 2 * time.Minute
	defaultAppleSiliconServerRetryInterval = 5 * time.Second
)

// asAPIWithZone returns a new apple silicon API and the zone
func asAPIWithZone(d *schema.ResourceData, m interface{}) (*applesilicon.API, scw.Zone, error) {
	meta := m.(*meta2.Meta)
	asAPI := applesilicon.NewAPI(meta.GetScwClient())

	zone, err := locality.ExtractZone(d, meta)
	if err != nil {
		return nil, "", err
	}
	return asAPI, zone, nil
}

// asAPIWithZoneAndID returns an apple silicon API with zone and ID extracted from the state
func asAPIWithZoneAndID(m interface{}, id string) (*applesilicon.API, scw.Zone, string, error) {
	meta := m.(*meta2.Meta)
	asAPI := applesilicon.NewAPI(meta.GetScwClient())

	zone, ID, err := locality.ParseZonedID(id)
	if err != nil {
		return nil, "", "", err
	}
	return asAPI, zone, ID, nil
}

func waitForAppleSiliconServer(ctx context.Context, api *applesilicon.API, zone scw.Zone, serverID string, timeout time.Duration) (*applesilicon.Server, error) {
	retryInterval := defaultAppleSiliconServerRetryInterval
	if transport.DefaultWaitRetryInterval != nil {
		retryInterval = *transport.DefaultWaitRetryInterval
	}

	server, err := api.WaitForServer(&applesilicon.WaitForServerRequest{
		ServerID:      serverID,
		Zone:          zone,
		Timeout:       scw.TimeDurationPtr(timeout),
		RetryInterval: &retryInterval,
	}, scw.WithContext(ctx))

	return server, err
}
