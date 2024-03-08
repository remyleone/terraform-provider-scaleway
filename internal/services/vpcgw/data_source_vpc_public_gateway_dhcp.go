package vpcgw

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/datasource"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
)

func DataSourceScalewayVPCPublicGatewayDHCP() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasource.DatasourceSchemaFromResourceSchema(ResourceScalewayVPCPublicGatewayDHCP().Schema)

	dsSchema["dhcp_id"] = &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		Description:  "The ID of the public gateway DHCP configuration",
		ValidateFunc: locality.UUIDorUUIDWithLocality(),
	}

	return &schema.Resource{
		Schema:      dsSchema,
		ReadContext: DataSourceScalewayVPCPublicGatewayDHCPRead,
	}
}

func DataSourceScalewayVPCPublicGatewayDHCPRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	_, zone, err := VpcgwAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	dhcpID, _ := d.GetOk("dhcp_id")

	zonedID := locality.DatasourceNewZonedID(dhcpID, zone)
	d.SetId(zonedID)
	_ = d.Set("dhcp_id", zonedID)
	return ResourceScalewayVPCPublicGatewayDHCPRead(ctx, d, meta)
}
