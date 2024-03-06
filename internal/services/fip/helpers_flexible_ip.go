package fip

import (
	"context"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
"github.com/scaleway/terraform-provider-scaleway/v2/internal/transport"
"time"

meta2 "github.com/scaleway/terraform-provider-scaleway/v2/internal/meta"

"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
flexibleip "github.com/scaleway/scaleway-sdk-go/api/flexibleip/v1alpha1"
"github.com/scaleway/scaleway-sdk-go/scw"
)

const (
	defaultFlexibleIPTimeout = 1 * time.Minute
	retryFlexibleIPInterval  = 5 * time.Second
)

// fipAPIWithZone returns an lb API WITH zone for a Create request
func fipAPIWithZone(d *schema.ResourceData, m interface{}) (*flexibleip.API, scw.Zone, error) {
	meta := m.(*meta2.Meta)
	flexibleipAPI := flexibleip.NewAPI(meta.GetScwClient())

	zone, err := locality.ExtractZone(d, meta)
	if err != nil {
		return nil, "", err
	}
	return flexibleipAPI, zone, nil
}

// fipAPIWithZoneAndID returns an flexibleip API with zone and ID extracted from the state
func fipAPIWithZoneAndID(m interface{}, id string) (*flexibleip.API, scw.Zone, string, error) {
	meta := m.(*meta2.Meta)
	fipAPI := flexibleip.NewAPI(meta.GetScwClient())

	zone, ID, err := locality.ParseZonedID(id)
	if err != nil {
		return nil, "", "", err
	}
	return fipAPI, zone, ID, nil
}

func waitFlexibleIP(ctx context.Context, api *flexibleip.API, zone scw.Zone, id string, timeout time.Duration) (*flexibleip.FlexibleIP, error) {
	retryInterval := retryFlexibleIPInterval
	if transport.DefaultWaitRetryInterval != nil {
		retryInterval = *transport.DefaultWaitRetryInterval
	}

	return api.WaitForFlexibleIP(&flexibleip.WaitForFlexibleIPRequest{
		FipID:         id,
		Zone:          zone,
		Timeout:       scw.TimeDurationPtr(timeout),
		RetryInterval: &retryInterval,
	}, scw.WithContext(ctx))
}

func flattenFlexibleIPMacAddress(mac *flexibleip.MACAddress) interface{} {
	if mac == nil {
		return nil
	}
	return []map[string]interface{}{
		{
			"id":          mac.ID,
			"mac_address": mac.MacAddress,
			"mac_type":    mac.MacType,
			"status":      mac.Status,
			"created_at":  types.FlattenTime(mac.CreatedAt),
			"updated_at":  types.FlattenTime(mac.UpdatedAt),
			"zone":        mac.Zone,
		},
	}
}

func expandServerIDs(data interface{}) []string {
	expandedIDs := make([]string, 0, len(data.([]interface{})))
	for _, s := range data.([]interface{}) {
		if s == nil {
			s = ""
		}
		expandedID := locality.ExpandID(s.(string))
		expandedIDs = append(expandedIDs, expandedID)
	}
	return expandedIDs
}
