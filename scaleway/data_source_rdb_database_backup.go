package scaleway

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/api/rdb/v1"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/types"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/verify"
)

func dataSourceScalewayRDBDatabaseBackup() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasourceSchemaFromResourceSchema(resourceScalewayRdbDatabaseBackup().Schema)

	addOptionalFieldsToSchema(dsSchema, "name", "region", "instance_id")

	dsSchema["instance_id"].RequiredWith = []string{"name"}
	dsSchema["backup_id"] = &schema.Schema{
		Type:          schema.TypeString,
		Optional:      true,
		Description:   "The ID of the Backup",
		ConflictsWith: []string{"name", "instance_id"},
		ValidateFunc:  verify.UUIDorUUIDWithLocality(),
	}
	dsSchema["project_id"] = &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		Description:  "The ID of the project to filter the Backup",
		ValidateFunc: verify.UUID(),
	}

	return &schema.Resource{
		ReadContext: dataSourceScalewayRDBDatabaseBackupRead,
		Schema:      dsSchema,
	}
}

func dataSourceScalewayRDBDatabaseBackupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := rdbAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	backupID, backupIDExists := d.GetOk("backup_id")
	if !backupIDExists {
		backupName := d.Get("name").(string)
		res, err := api.ListDatabaseBackups(&rdb.ListDatabaseBackupsRequest{
			Region:     region,
			Name:       types.ExpandStringPtr(backupName),
			InstanceID: types.ExpandStringPtr(locality.ExpandID(d.Get("instance_id"))),
			ProjectID:  types.ExpandStringPtr(d.Get("project_id")),
		})
		if err != nil {
			return diag.FromErr(err)
		}

		foundBackup, err := findExact(
			res.DatabaseBackups,
			func(s *rdb.DatabaseBackup) bool { return s.Name == backupName },
			backupName,
		)
		if err != nil {
			return diag.FromErr(err)
		}

		backupID = foundBackup.ID
	}

	regionID := datasourceNewRegionalID(backupID, region)
	d.SetId(regionID)
	err = d.Set("backup_id", regionID)
	if err != nil {
		return diag.FromErr(err)
	}

	diags := resourceScalewayRdbDatabaseBackupRead(ctx, d, meta)
	if diags != nil {
		return append(diags, diag.Errorf("failed to read database backup state")...)
	}

	if d.Id() == "" {
		return diag.Errorf("database backup (%s) not found", regionID)
	}

	return nil
}
