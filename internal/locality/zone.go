package locality

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/meta"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/verify"
)

// ZonalSchema returns a standard schema for a zone
func ZonalSchema() *schema.Schema {
	return &schema.Schema{
		Type:             schema.TypeString,
		Description:      "The zone you want to attach the resource to",
		Optional:         true,
		ForceNew:         true,
		Computed:         true,
		ValidateDiagFunc: verify.StringInSliceWithWarning(AllZones(), "zone"),
	}
}

func AllZones() []string {
	zones := make([]string, 0, len(scw.AllZones))
	for _, z := range scw.AllZones {
		zones = append(zones, z.String())
	}

	return zones
}

func DatasourceNewZonedID(idI interface{}, fallBackZone scw.Zone) string {
	zone, id, err := ParseZonedID(idI.(string))
	if err != nil {
		id = idI.(string)
		zone = fallBackZone
	}

	return NewZonedIDString(zone, id)
}

// NewZonedIDString constructs a unique identifier based on resource zone and id
func NewZonedIDString(zone scw.Zone, id string) string {
	return fmt.Sprintf("%s/%s", zone, id)
}

// ParseZonedID parses a zonedID and extracts the resource zone and id.
func ParseZonedID(zonedID string) (zone scw.Zone, id string, err error) {
	locality, id, err := ParseLocalizedID(zonedID)
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

// ZonedID represents an ID that is linked with a zone, eg fr-par-1/11111111-1111-1111-1111-111111111111
type ZonedID struct {
	ID   string
	Zone scw.Zone
}

func (z ZonedID) String() string {
	return fmt.Sprintf("%s/%s", z.Zone, z.ID)
}

func NewZonedID(zone scw.Zone, id string) ZonedID {
	return ZonedID{
		ID:   id,
		Zone: zone,
	}
}
