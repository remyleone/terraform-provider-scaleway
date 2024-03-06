package instance

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	block "github.com/scaleway/scaleway-sdk-go/api/block/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	meta2 "github.com/scaleway/terraform-provider-scaleway/v2/internal/meta"
)

type InstanceBlockAPI struct {
	*instance.API
	blockAPI *block.API
}

// instanceAPIWithZone returns a new instance API and the zone for a Create request
func instanceAndBlockAPIWithZone(d *schema.ResourceData, m interface{}) (*InstanceBlockAPI, scw.Zone, error) {
	meta := m.(*meta2.Meta)
	instanceAPI := instance.NewAPI(meta.GetScwClient())
	blockAPI := block.NewAPI(meta.GetScwClient())

	zone, err := locality.ExtractZone(d, meta)
	if err != nil {
		return nil, "", err
	}

	return &InstanceBlockAPI{
		API:      instanceAPI,
		blockAPI: blockAPI,
	}, zone, nil
}

// instanceAPIWithZoneAndID returns an instance API with zone and ID extracted from the state
func instanceAndBlockAPIWithZoneAndID(m interface{}, zonedID string) (*InstanceBlockAPI, scw.Zone, string, error) {
	meta := m.(*meta2.Meta)
	instanceAPI := instance.NewAPI(meta.GetScwClient())
	blockAPI := block.NewAPI(meta.GetScwClient())

	zone, ID, err := locality.ParseZonedID(zonedID)
	if err != nil {
		return nil, "", "", err
	}

	return &InstanceBlockAPI{
		API:      instanceAPI,
		blockAPI: blockAPI,
	}, zone, ID, nil
}
