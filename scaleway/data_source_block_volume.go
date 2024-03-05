package scaleway

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	block "github.com/scaleway/scaleway-sdk-go/api/block/v1alpha1"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/types"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/verify"
)

func dataSourceScalewayBlockVolume() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasourceSchemaFromResourceSchema(resourceScalewayBlockVolume().Schema)

	addOptionalFieldsToSchema(dsSchema, "name", "zone", "project_id")

	dsSchema["volume_id"] = &schema.Schema{
		Type:          schema.TypeString,
		Optional:      true,
		Description:   "The ID of the volume",
		ConflictsWith: []string{"name"},
		ValidateFunc:  verify.UUIDorUUIDWithLocality(),
	}

	return &schema.Resource{
		ReadContext: dataSourceScalewayBlockVolumeRead,
		Schema:      dsSchema,
	}
}

func dataSourceScalewayBlockVolumeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, zone, err := blockAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	volumeID, volumeIDExists := d.GetOk("volume_id")
	if !volumeIDExists {
		res, err := api.ListVolumes(&block.ListVolumesRequest{
			Zone:      zone,
			Name:      types.ExpandStringPtr(d.Get("name")),
			ProjectID: types.ExpandStringPtr(d.Get("project_id")),
		})
		if err != nil {
			return diag.FromErr(err)
		}
		for _, volume := range res.Volumes {
			if volume.Name == d.Get("name").(string) {
				if volumeID != "" {
					return diag.Errorf("more than 1 volume found with the same name %s", d.Get("name"))
				}
				volumeID = volume.ID
			}
		}
		if volumeID == "" {
			return diag.Errorf("no volume found with the name %s", d.Get("name"))
		}
	}

	zoneID := datasourceNewZonedID(volumeID, zone)
	d.SetId(zoneID)
	err = d.Set("volume_id", zoneID)
	if err != nil {
		return diag.FromErr(err)
	}

	diags := resourceScalewayBlockVolumeRead(ctx, d, meta)
	if diags != nil {
		return append(diags, diag.Errorf("failed to read volume state")...)
	}

	if d.Id() == "" {
		return diag.Errorf("volume (%s) not found", zoneID)
	}

	return nil
}
