package scaleway

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/types"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/verify"
)

func dataSourceScalewayInstanceSecurityGroup() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasourceSchemaFromResourceSchema(resourceScalewayInstanceSecurityGroup().Schema)

	// Set 'Optional' schema elements
	addOptionalFieldsToSchema(dsSchema, "name", "zone", "project_id")

	dsSchema["name"].ConflictsWith = []string{"security_group_id"}
	dsSchema["security_group_id"] = &schema.Schema{
		Type:          schema.TypeString,
		Optional:      true,
		Description:   "The ID of the security group",
		ValidateFunc:  verify.UUIDorUUIDWithLocality(),
		ConflictsWith: []string{"name"},
	}

	return &schema.Resource{
		ReadContext: dataSourceScalewayInstanceSecurityGroupRead,

		Schema: dsSchema,
	}
}

func dataSourceScalewayInstanceSecurityGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	instanceAPI, zone, err := instanceAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	securityGroupID, ok := d.GetOk("security_group_id")
	if !ok {
		sgName := d.Get("name").(string)
		res, err := instanceAPI.ListSecurityGroups(&instance.ListSecurityGroupsRequest{
			Zone:    zone,
			Name:    types.ExpandStringPtr(sgName),
			Project: types.ExpandStringPtr(d.Get("project_id")),
		}, scw.WithAllPages(), scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}

		foundSG, err := findExact(
			res.SecurityGroups,
			func(s *instance.SecurityGroup) bool { return s.Name == sgName },
			sgName,
		)
		if err != nil {
			return diag.FromErr(err)
		}

		securityGroupID = foundSG.ID
	}

	zonedID := datasourceNewZonedID(securityGroupID, zone)
	d.SetId(zonedID)
	_ = d.Set("security_group_id", zonedID)
	return resourceScalewayInstanceSecurityGroupRead(ctx, d, meta)
}
