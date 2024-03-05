package zonal

import (
	"fmt"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/meta"

	"github.com/scaleway/scaleway-sdk-go/scw"
)

// NewZonedIDString constructs a unique identifier based on resource zone and id
func NewZonedIDString(zone scw.Zone, id string) string {
	return fmt.Sprintf("%s/%s", zone, id)
}

// ParseZonedID parses a zonedID and extracts the resource zone and id.
func ParseZonedID(zonedID string) (zone scw.Zone, id string, err error) {
	locality, id, err := locality.ParseLocalizedID(zonedID)
	if err != nil {
		return zone, id, err
	}

	zone, err = scw.ParseZone(locality)
	return
}

// ExtractZone will try to guess the zone from the following:
//   - zone field of the resource data
//   - default zone from config
func ExtractZone(d meta.TerraformResourceData, meta *meta.Meta) (scw.Zone, error) {
	rawZone, exist := d.GetOk("zone")
	if exist {
		return scw.ParseZone(rawZone.(string))
	}

	zone, exist := meta.GetScwClient().GetDefaultZone()
	if exist {
		return zone, nil
	}

	return "", ErrZoneNotFound
}
