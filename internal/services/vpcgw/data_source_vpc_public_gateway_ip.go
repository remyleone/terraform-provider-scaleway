package vpcgw

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/datasource"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
)

func DataSourceScalewayVPCPublicGatewayIP() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasource.DatasourceSchemaFromResourceSchema(ResourceScalewayVPCPublicGatewayIP().Schema)

	dsSchema["ip_id"] = &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		Description:  "The ID of the IP",
		ValidateFunc: locality.UUIDorUUIDWithLocality(),
	}

	return &schema.Resource{
		Schema:      dsSchema,
		ReadContext: DataSourceScalewayVPCPublicGatewayIPRead,
	}
}

func DataSourceScalewayVPCPublicGatewayIPRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	_, zone, err := VpcgwAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	ipID, _ := d.GetOk("ip_id")

	zonedID := locality.DatasourceNewZonedID(ipID, zone)
	d.SetId(zonedID)
	_ = d.Set("ip_id", zonedID)
	return ResourceScalewayVPCPublicGatewayIPRead(ctx, d, meta)
}
