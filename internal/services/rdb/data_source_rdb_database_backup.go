package rdb

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/api/rdb/v1"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/datasource"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/verify"
)

func DataSourceScalewayRDBDatabaseBackup() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasource.DatasourceSchemaFromResourceSchema(ResourceScalewayRdbDatabaseBackup().Schema)

	datasource.AddOptionalFieldsToSchema(dsSchema, "name", "region", "instance_id")

	dsSchema["instance_id"].RequiredWith = []string{"name"}
	dsSchema["backup_id"] = &schema.Schema{
		Type:          schema.TypeString,
		Optional:      true,
		Description:   "The ID of the Backup",
		ConflictsWith: []string{"name", "instance_id"},
		ValidateFunc:  locality.UUIDorUUIDWithLocality(),
	}
	dsSchema["project_id"] = &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		Description:  "The ID of the project to filter the Backup",
		ValidateFunc: verify.UUID(),
	}

	return &schema.Resource{
		ReadContext: DataSourceScalewayRDBDatabaseBackupRead,
		Schema:      dsSchema,
	}
}

func DataSourceScalewayRDBDatabaseBackupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := RdbAPIWithRegion(d, meta)
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

		foundBackup, err := datasource.FindExact(
			res.DatabaseBackups,
			func(s *rdb.DatabaseBackup) bool { return s.Name == backupName },
			backupName,
		)
		if err != nil {
			return diag.FromErr(err)
		}

		backupID = foundBackup.ID
	}

	regionID := locality.DatasourceNewRegionalID(backupID, region)
	d.SetId(regionID)
	err = d.Set("backup_id", regionID)
	if err != nil {
		return diag.FromErr(err)
	}

	diags := ResourceScalewayRdbDatabaseBackupRead(ctx, d, meta)
	if diags != nil {
		return append(diags, diag.Errorf("failed to read database backup state")...)
	}

	if d.Id() == "" {
		return diag.Errorf("database backup (%s) not found", regionID)
	}

	return nil
}
