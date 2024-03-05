package scaleway

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/api/vpcgw/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/types"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/verify"
)

func dataSourceScalewayVPCPublicGateway() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasourceSchemaFromResourceSchema(resourceScalewayVPCPublicGateway().Schema)

	// Set 'Optional' schema elements
	addOptionalFieldsToSchema(dsSchema, "name", "zone", "project_id")

	dsSchema["name"].ConflictsWith = []string{"public_gateway_id"}
	dsSchema["public_gateway_id"] = &schema.Schema{
		Type:          schema.TypeString,
		Optional:      true,
		Description:   "The ID of the public gateway",
		ValidateFunc:  verify.UUIDorUUIDWithLocality(),
		ConflictsWith: []string{"name"},
	}

	return &schema.Resource{
		Schema:      dsSchema,
		ReadContext: dataSourceScalewayVPCPublicGatewayRead,
	}
}

func dataSourceScalewayVPCPublicGatewayRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vpcgwAPI, zone, err := vpcgwAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	if v, ok := d.GetOk("zone"); ok {
		zone = scw.Zone(v.(string))
	}

	publicGatewayID, ok := d.GetOk("public_gateway_id")
	if !ok {
		gwName := d.Get("name").(string)
		res, err := vpcgwAPI.ListGateways(
			&vpcgw.ListGatewaysRequest{
				Name:      types.ExpandStringPtr(gwName),
				Zone:      zone,
				ProjectID: types.ExpandStringPtr(d.Get("project_id")),
			}, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}

		foundGW, err := findExact(
			res.Gateways,
			func(s *vpcgw.Gateway) bool { return s.Name == gwName },
			gwName,
		)
		if err != nil {
			return diag.FromErr(err)
		}

		publicGatewayID = foundGW.ID
	}

	zonedID := datasourceNewZonedID(publicGatewayID, zone)
	d.SetId(zonedID)
	_ = d.Set("public_gateway_id", zonedID)
	return resourceScalewayVPCPublicGatewayRead(ctx, d, meta)
}
