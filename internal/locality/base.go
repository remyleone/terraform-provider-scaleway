package locality

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/meta"
	"strings"
)

// CompareLocalities compare two localities
// They are equal if they are the same or if one is a zone contained in a region
func CompareLocalities(loc1, loc2 string) bool {
	if loc1 == loc2 {
		return true
	}
	if strings.HasPrefix(loc1, loc2) || strings.HasPrefix(loc2, loc1) {
		return true
	}
	return false
}

// GetLocality find the locality of a resource
// Will try to get the zone if available then use region
// Will also use default zone or region if available
func GetLocality(diff *schema.ResourceDiff, meta *meta.Meta) string {
	var locality string

	rawStateType := diff.GetRawState().Type()

	if rawStateType.HasAttribute("zone") {
		zone, _ := ExtractZone(diff, meta)
		locality = zone.String()
	} else if rawStateType.HasAttribute("region") {
		region, _ := ExtractRegion(diff, meta)
		locality = region.String()
	}
	return locality
}

// NewZonedNestedIDString constructs a unique identifier based on resource zone, inner and outer IDs
func NewZonedNestedIDString(zone scw.Zone, outerID, innerID string) string {
	return fmt.Sprintf("%s/%s/%s", zone, outerID, innerID)
}
