package locality

import (
	"fmt"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/meta"
	"strings"
)

func AllRegions() []string {
	regions := make([]string, 0, len(scw.AllRegions))
	for _, z := range scw.AllRegions {
		regions = append(regions, z.String())
	}

	return regions
}

func DatasourceNewRegionalID(idI interface{}, fallBackRegion scw.Region) string {
	region, id, err := ParseRegionalID(idI.(string))
	if err != nil {
		id = idI.(string)
		region = fallBackRegion
	}

	return NewRegionalIDString(region, id)
}

// ExtractRegion will try to guess the region from the following:
//   - region field of the resource data
//   - default region from config
func ExtractRegion(d meta.TerraformResourceData, meta *meta.Meta) (scw.Region, error) {
	rawRegion, exist := d.GetOk("region")
	if exist {
		return scw.ParseRegion(rawRegion.(string))
	}

	region, exist := meta.GetScwClient().GetDefaultRegion()
	if exist {
		return region, nil
	}

	return "", ErrRegionNotFound
}

// ExtractRegionWithDefault will try to guess the region from the following:
//   - region field of the resource data
//   - default region given in argument
//   - default region from config
func ExtractRegionWithDefault(d meta.TerraformResourceData, meta *meta.Meta, defaultRegion scw.Region) (scw.Region, error) {
	rawRegion, exist := d.GetOk("region")
	if exist {
		return scw.ParseRegion(rawRegion.(string))
	}

	if defaultRegion != "" {
		return defaultRegion, nil
	}

	region, exist := meta.GetScwClient().GetDefaultRegion()
	if exist {
		return region, nil
	}

	return "", ErrRegionNotFound
}

// NewRegionalIDString constructs a unique identifier based on resource region and id
func NewRegionalIDString(region scw.Region, id string) string {
	return fmt.Sprintf("%s/%s", region, id)
}

func ExpandRegionalID(id interface{}) RegionalID {
	regionalID := RegionalID{}
	tab := strings.Split(id.(string), "/")
	if len(tab) != 2 {
		regionalID.ID = id.(string)
	} else {
		region, _ := scw.ParseRegion(tab[0])
		regionalID.ID = tab[1]
		regionalID.Region = region
	}

	return regionalID
}

func NewRegionalID(region scw.Region, id string) RegionalID {
	return RegionalID{
		ID:     id,
		Region: region,
	}
}

// RegionalID represents an ID that is linked with a region, eg fr-par/11111111-1111-1111-1111-111111111111
type RegionalID struct {
	ID     string
	Region scw.Region
}

func (z RegionalID) String() string {
	return fmt.Sprintf("%s/%s", z.Region, z.ID)
}
