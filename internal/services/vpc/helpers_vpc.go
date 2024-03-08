package vpc

import (
	"context"
	"errors"
	"fmt"
	locality2 "github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
	"net"
	"strconv"
	"strings"
	"time"

	meta2 "github.com/scaleway/terraform-provider-scaleway/v2/internal/meta"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/api/vpc/v2"
	"github.com/scaleway/scaleway-sdk-go/scw"
	validator "github.com/scaleway/scaleway-sdk-go/validation"
)

const defaultVPCPrivateNetworkRetryInterval = 30 * time.Second

// vpcAPIWithRegion returns a new VPC API and the region for a Create request
func vpcAPIWithRegion(d *schema.ResourceData, m interface{}) (*vpc.API, scw.Region, error) {
	meta := m.(*meta2.Meta)
	vpcAPI := vpc.NewAPI(meta.GetScwClient())

	region, err := locality2.ExtractRegion(d, meta)
	if err != nil {
		return nil, "", err
	}
	return vpcAPI, region, err
}

// VpcAPIWithRegionAndID returns a new VPC API with locality and ID extracted from the state
func VpcAPIWithRegionAndID(m interface{}, id string) (*vpc.API, scw.Region, string, error) {
	meta := m.(*meta2.Meta)
	vpcAPI := vpc.NewAPI(meta.GetScwClient())

	region, ID, err := locality2.ParseRegionalID(id)
	if err != nil {
		return nil, "", "", err
	}
	return vpcAPI, region, ID, err
}

func VPCAPI(m interface{}) (*vpc.API, error) {
	meta, ok := m.(*meta2.Meta)
	if !ok {
		return nil, fmt.Errorf("wrong type: %T", m)
	}

	return vpc.NewAPI(meta.GetScwClient()), nil
}

func expandSubnets(d *schema.ResourceData) (ipv4Subnets []scw.IPNet, ipv6Subnets []scw.IPNet, err error) {
	if v, ok := d.GetOk("ipv4_subnet"); ok {
		for _, s := range v.([]interface{}) {
			rawSubnet := s.(map[string]interface{})
			ipNet, err := types.ExpandIPNet(rawSubnet["subnet"].(string))
			if err != nil {
				return nil, nil, err
			}
			ipv4Subnets = append(ipv4Subnets, ipNet)
		}
	}

	if v, ok := d.GetOk("ipv6_subnets"); ok {
		for _, s := range v.(*schema.Set).List() {
			rawSubnet := s.(map[string]interface{})
			ipNet, err := types.ExpandIPNet(rawSubnet["subnet"].(string))
			if err != nil {
				return nil, nil, err
			}
			ipv6Subnets = append(ipv6Subnets, ipNet)
		}
	}
	return
}

func FlattenAndSortSubnets(sub interface{}) (interface{}, interface{}) {
	switch subnets := sub.(type) {
	case []scw.IPNet:
		return FlattenAndSortIPNetSubnets(subnets)
	case []*vpc.Subnet:
		return FlattenAndSortSubnetV2s(subnets)
	default:
		return "", nil
	}
}

func FlattenAndSortIPNetSubnets(subnets []scw.IPNet) (interface{}, interface{}) {
	if subnets == nil {
		return "", nil
	}

	flattenedipv4Subnets := []map[string]interface{}(nil)
	flattenedipv6Subnets := []map[string]interface{}(nil)

	for _, s := range subnets {
		// If it's an IPv4 subnet
		if s.IP.To4() != nil {
			sub, err := types.FlattenIPNet(s)
			if err != nil {
				return "", nil
			}
			flattenedipv4Subnets = append(flattenedipv4Subnets, map[string]interface{}{
				"subnet":        sub,
				"address":       s.IP.String(),
				"subnet_mask":   maskHexToDottedDecimal(s.Mask),
				"prefix_length": getPrefixLength(s.Mask),
			})
		} else {
			sub, err := types.FlattenIPNet(s)
			if err != nil {
				return "", nil
			}
			flattenedipv6Subnets = append(flattenedipv6Subnets, map[string]interface{}{
				"subnet":        sub,
				"address":       s.IP.String(),
				"subnet_mask":   maskHexToDottedDecimal(s.IPNet.Mask),
				"prefix_length": getPrefixLength(s.Mask),
			})
		}
	}

	return flattenedipv4Subnets, flattenedipv6Subnets
}

func FlattenAndSortSubnetV2s(subnets []*vpc.Subnet) (interface{}, interface{}) {
	if subnets == nil {
		return "", nil
	}

	flattenedipv4Subnets := []map[string]interface{}(nil)
	flattenedipv6Subnets := []map[string]interface{}(nil)

	for _, s := range subnets {
		// If it's an IPv4 subnet
		if s.Subnet.IP.To4() != nil {
			sub, err := types.FlattenIPNet(s.Subnet)
			if err != nil {
				return "", nil
			}
			flattenedipv4Subnets = append(flattenedipv4Subnets, map[string]interface{}{
				"id":            s.ID,
				"created_at":    types.FlattenTime(s.CreatedAt),
				"updated_at":    types.FlattenTime(s.UpdatedAt),
				"subnet":        sub,
				"address":       s.Subnet.IP.String(),
				"subnet_mask":   maskHexToDottedDecimal(s.Subnet.Mask),
				"prefix_length": getPrefixLength(s.Subnet.Mask),
			})
		} else {
			sub, err := types.FlattenIPNet(s.Subnet)
			if err != nil {
				return "", nil
			}
			flattenedipv6Subnets = append(flattenedipv6Subnets, map[string]interface{}{
				"id":            s.ID,
				"created_at":    types.FlattenTime(s.CreatedAt),
				"updated_at":    types.FlattenTime(s.UpdatedAt),
				"subnet":        sub,
				"address":       s.Subnet.IP.String(),
				"subnet_mask":   maskHexToDottedDecimal(s.Subnet.Mask),
				"prefix_length": getPrefixLength(s.Subnet.Mask),
			})
		}
	}

	return flattenedipv4Subnets, flattenedipv6Subnets
}

func maskHexToDottedDecimal(mask net.IPMask) string {
	if len(mask) != net.IPv4len && len(mask) != net.IPv6len {
		return ""
	}

	parts := make([]string, len(mask))
	for i, part := range mask {
		parts[i] = strconv.Itoa(int(part))
	}
	return strings.Join(parts, ".")
}

func getPrefixLength(mask net.IPMask) int {
	ones, _ := mask.Size()
	return ones
}

func vpcPrivateNetworkUpgradeV1SchemaType() cty.Type {
	return cty.Object(map[string]cty.Type{
		"id": cty.String,
	})
}

func vpcPrivateNetworkV1SUpgradeFunc(_ context.Context, rawState map[string]interface{}, _ interface{}) (map[string]interface{}, error) {
	var err error

	ID, exist := rawState["id"]
	if !exist {
		return nil, errors.New("upgrade: id not exist")
	}
	rawState["id"], err = vpcPrivateNetworkUpgradeV1ZonalToRegionalID(ID.(string))
	if err != nil {
		return nil, err
	}

	return rawState, nil
}

func vpcPrivateNetworkUpgradeV1ZonalToRegionalID(element string) (string, error) {
	locality, id, err := locality2.ParseLocalizedID(element)
	// return error if can't parse
	if err != nil {
		return "", fmt.Errorf("upgrade: could not retrieve the locality from `%s`", element)
	}
	// if locality is already regional return
	if validator.IsRegion(locality) {
		return element, nil
	}

	fetchRegion, err := scw.Zone(locality).Region()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s", fetchRegion.String(), id), nil
}
