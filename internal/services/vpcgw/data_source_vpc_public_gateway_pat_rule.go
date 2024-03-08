package vpcgw

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/api/vpcgw/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/datasource"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
)

func DataSourcePATRule() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasource.DatasourceSchemaFromResourceSchema(ResourceScalewayVPCPublicGatewayPATRule().Schema)

	dsSchema["pat_rule_id"] = &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		Description:  "The ID of the public gateway PAT rule",
		ValidateFunc: locality.UUIDorUUIDWithLocality(),
	}

	// Set 'Optional' schema elements
	datasource.AddOptionalFieldsToSchema(dsSchema, "zone")

	return &schema.Resource{
		Schema:      dsSchema,
		ReadContext: DataSourceScalewayVPCPublicGatewayPATRuleRead,
	}
}

func DataSourceScalewayVPCPublicGatewayPATRuleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vpcgwAPI, zone, err := VpcgwAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	patRuleIDRaw := d.Get("pat_rule_id")

	zonedID := locality.DatasourceNewZonedID(patRuleIDRaw, zone)
	d.SetId(zonedID)
	_ = d.Set("pat_rule_id", zonedID)

	// check if pat rule exist
	_, err = vpcgwAPI.GetPATRule(&vpcgw.GetPATRuleRequest{
		PatRuleID: locality.ExpandID(patRuleIDRaw),
		Zone:      zone,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayVPCPublicGatewayPATRuleRead(ctx, d, meta)
}
