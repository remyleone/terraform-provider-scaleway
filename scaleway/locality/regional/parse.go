package regional

import (
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/locality"
)

// ParseRegionalID parses a regionalID and extracts the resource region and id.
func ParseRegionalID(regionalID string) (region scw.Region, id string, err error) {
	locality, id, err := locality.ParseLocalizedID(regionalID)
	if err != nil {
		return
	}

	region, err = scw.ParseRegion(locality)
	return
}
