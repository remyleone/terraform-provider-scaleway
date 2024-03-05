package scaleway

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/types"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/verify"
)

func dataSourceScalewayInstancePlacementGroup() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasourceSchemaFromResourceSchema(resourceScalewayInstancePlacementGroup().Schema)

	addOptionalFieldsToSchema(dsSchema, "name", "zone")

	dsSchema["placement_group_id"] = &schema.Schema{
		Type:          schema.TypeString,
		Optional:      true,
		Description:   "The ID of the placementgroup",
		ConflictsWith: []string{"name"},
		ValidateFunc:  verify.UUIDorUUIDWithLocality(),
	}
	dsSchema["project_id"].Optional = true

	return &schema.Resource{
		ReadContext: dataSourceScalewayInstancePlacementGroupRead,
		Schema:      dsSchema,
	}
}

func dataSourceScalewayInstancePlacementGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, zone, err := instanceAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	placementGroupID, placementGroupIDExists := d.GetOk("placement_group_id")
	if !placementGroupIDExists {
		res, err := api.ListPlacementGroups(&instance.ListPlacementGroupsRequest{
			Zone:    zone,
			Name:    types.ExpandStringPtr(d.Get("name")),
			Project: types.ExpandStringPtr(d.Get("project_id")),
		})
		if err != nil {
			return diag.FromErr(err)
		}

		for _, placementGroup := range res.PlacementGroups {
			if placementGroup.Name == d.Get("name").(string) {
				if placementGroupID != "" {
					return diag.Errorf("more than 1 placement group found with the same name %s", d.Get("name"))
				}
				placementGroupID = placementGroup.ID
			}
		}
		if placementGroupID == "" {
			return diag.Errorf("no placementgroup found with the name %s", d.Get("name"))
		}
	}

	zoneID := datasourceNewZonedID(placementGroupID, zone)
	d.SetId(zoneID)
	err = d.Set("placement_group_id", zoneID)
	if err != nil {
		return diag.FromErr(err)
	}

	diags := resourceScalewayInstancePlacementGroupRead(ctx, d, meta)
	if diags != nil {
		return append(diags, diag.Errorf("failed to read placement group state")...)
	}

	if d.Id() == "" {
		return diag.Errorf("placement group (%s) not found", zoneID)
	}

	return nil
}
