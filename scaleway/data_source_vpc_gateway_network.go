package scaleway

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/api/vpcgw/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/types"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/verify"
)

func dataSourceScalewayVPCGatewayNetwork() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasourceSchemaFromResourceSchema(resourceScalewayVPCGatewayNetwork().Schema)

	// Set 'Optional' schema elements
	searchFields := []string{
		"gateway_id",
		"private_network_id",
		"enable_masquerade",
		"dhcp_id",
	}
	addOptionalFieldsToSchema(dsSchema, searchFields...)

	dsSchema["gateway_network_id"] = &schema.Schema{
		Type:          schema.TypeString,
		Optional:      true,
		Description:   "The ID of the gateway network",
		ValidateFunc:  verify.UUIDorUUIDWithLocality(),
		ConflictsWith: searchFields,
	}

	return &schema.Resource{
		Schema:      dsSchema,
		ReadContext: dataSourceScalewayVPCGatewayNetworkRead,
	}
}

func dataSourceScalewayVPCGatewayNetworkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vpcAPI, zone, err := vpcgwAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	gatewayNetworkID, ok := d.GetOk("gateway_network_id")
	if !ok {
		res, err := vpcAPI.ListGatewayNetworks(&vpcgw.ListGatewayNetworksRequest{
			GatewayID:        types.ExpandStringPtr(locality.ExpandID(d.Get("gateway_id"))),
			PrivateNetworkID: types.ExpandStringPtr(locality.ExpandID(d.Get("private_network_id"))),
			EnableMasquerade: expandBoolPtr(getBool(d, "enable_masquerade")),
			DHCPID:           types.ExpandStringPtr(locality.ExpandID(d.Get("dhcp_id").(string))),
			Zone:             zone,
		}, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
		if res.TotalCount == 0 {
			return diag.FromErr(errors.New("no gateway network found with the filters"))
		}
		if res.TotalCount > 1 {
			return diag.FromErr(fmt.Errorf("%d gateway networks found with filters", res.TotalCount))
		}
		gatewayNetworkID = res.GatewayNetworks[0].ID
	}

	zonedID := datasourceNewZonedID(gatewayNetworkID, zone)
	d.SetId(zonedID)

	_ = d.Set("gateway_network_id", zonedID)

	diags := resourceScalewayVPCGatewayNetworkRead(ctx, d, meta)
	if len(diags) > 0 {
		return append(diags, diag.Errorf("failed to read gateway network state")...)
	}

	if d.Id() == "" {
		return diag.Errorf("gateway network (%s) not found", zonedID)
	}

	return nil
}
