package scaleway

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/api/marketplace/v2"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/locality/zonal"
	meta2 "github.com/scaleway/terraform-provider-scaleway/v2/scaleway/meta"
)

// marketplaceAPIWithZone returns a new marketplace API and the zone for a Create request
func marketplaceAPIWithZone(d *schema.ResourceData, m interface{}) (*marketplace.API, scw.Zone, error) {
	meta := m.(*meta2.Meta)
	marketplaceAPI := marketplace.NewAPI(meta.GetScwClient())

	zone, err := zonal.ExtractZone(d, meta)
	if err != nil {
		return nil, "", err
	}
	return marketplaceAPI, zone, nil
}
