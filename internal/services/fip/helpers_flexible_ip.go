package fip

import (
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
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

// FipAPIWithZone returns an lb API WITH zone for a Create request
func FipAPIWithZone(d *schema.ResourceData, m interface{}) (*flexibleip.API, scw.Zone, error) {
	meta := m.(*meta2.Meta)
	flexibleipAPI := flexibleip.NewAPI(meta.GetScwClient())

	zone, err := locality.ExtractZone(d, meta)
	if err != nil {
		return nil, "", err
	}
	return flexibleipAPI, zone, nil
}

// FipAPIWithZoneAndID returns an flexibleip API with zone and ID extracted from the state
func FipAPIWithZoneAndID(m interface{}, id string) (*flexibleip.API, scw.Zone, string, error) {
	meta := m.(*meta2.Meta)
	fipAPI := flexibleip.NewAPI(meta.GetScwClient())

	zone, ID, err := locality.ParseZonedID(id)
	if err != nil {
		return nil, "", "", err
	}
	return fipAPI, zone, ID, nil
}
