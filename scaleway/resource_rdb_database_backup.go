package scaleway

import (
	"context"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/scaleway/errors"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/verify"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/types"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/api/rdb/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func resourceScalewayRdbDatabaseBackup() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScalewayRdbDatabaseBackupCreate,
		ReadContext:   resourceScalewayRdbDatabaseBackupRead,
		UpdateContext: resourceScalewayRdbDatabaseBackupUpdate,
		DeleteContext: resourceScalewayRdbDatabaseBackupDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(defaultRdbInstanceTimeout),
			Read:    schema.DefaultTimeout(defaultRdbInstanceTimeout),
			Update:  schema.DefaultTimeout(defaultRdbInstanceTimeout),
			Delete:  schema.DefaultTimeout(defaultRdbInstanceTimeout),
			Default: schema.DefaultTimeout(defaultRdbInstanceTimeout),
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"instance_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: verify.UUIDorUUIDWithLocality(),
				Description:  "Instance on which the user is created",
			},
			"database_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the database of this backup.",
			},
			"name": {
				Type:        schema.TypeString,
				Description: "Name of the backup.",
				Optional:    true,
				Computed:    true,
			},
			"size": {
				Type:        schema.TypeInt,
				Description: "Size of the backup (in bytes).",
				Computed:    true,
			},
			"instance_name": {
				Type:        schema.TypeString,
				Description: "Name of the instance of the backup.",
				Computed:    true,
			},
			"expires_at": {
				Type:             schema.TypeString,
				Description:      "Expiration date (Format ISO 8601). Cannot be removed.",
				Optional:         true,
				ValidateDiagFunc: validateDate(),
			},
			"created_at": {
				Type:        schema.TypeString,
				Description: "Creation date (Format ISO 8601).",
				Computed:    true,
			},
			"updated_at": {
				Type:        schema.TypeString,
				Description: "Updated date (Format ISO 8601).",
				Computed:    true,
			},
			// Common
			"region": regionSchema(),
		},
		CustomizeDiff: customizeDiffLocalityCheck("instance_id"),
	}
}

func resourceScalewayRdbDatabaseBackupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	rdbAPI, region, err := rdbAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	instanceID := d.Get("instance_id").(string)

	createReq := &rdb.CreateDatabaseBackupRequest{
		Region:       region,
		InstanceID:   locality.ExpandID(instanceID),
		DatabaseName: d.Get("database_name").(string),
		Name:         types.ExpandOrGenerateString(d.Get("name"), "backup"),
		ExpiresAt:    expandTimePtr(d.Get("expires_at")),
	}

	dbBackup, err := rdbAPI.CreateDatabaseBackup(createReq, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(newRegionalIDString(region, dbBackup.ID))

	_, err = waitForRDBDatabaseBackup(ctx, rdbAPI, region, dbBackup.ID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceScalewayRdbDatabaseBackupRead(ctx, d, meta)
}

func resourceScalewayRdbDatabaseBackupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	rdbAPI, region, id, err := rdbAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	dbBackup, err := waitForRDBDatabaseBackup(ctx, rdbAPI, region, id, d.Timeout(schema.TimeoutRead))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("instance_id", newRegionalID(region, dbBackup.InstanceID).String())
	_ = d.Set("name", dbBackup.Name)
	_ = d.Set("database_name", dbBackup.DatabaseName)
	_ = d.Set("instance_name", dbBackup.InstanceName)
	_ = d.Set("expires_at", flattenTime(dbBackup.ExpiresAt))
	_ = d.Set("created_at", flattenTime(dbBackup.CreatedAt))
	_ = d.Set("updated_at", flattenTime(dbBackup.UpdatedAt))
	_ = d.Set("size", flattenSize(dbBackup.Size))
	_ = d.Set("region", dbBackup.Region)

	d.SetId(newRegionalIDString(region, dbBackup.ID))

	return nil
}

func resourceScalewayRdbDatabaseBackupUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	rdbAPI, region, id, err := rdbAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("expires_at") && d.Get("expires_at").(string) == "" {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       "Invalid expires_at",
				Detail:        "You cannot remove expires_at after it was set.",
				AttributePath: cty.GetAttrPath("expires_at"),
			},
		}
	}

	req := &rdb.UpdateDatabaseBackupRequest{
		Region:           region,
		DatabaseBackupID: id,
		Name:             types.ExpandStringPtr(d.Get("name")),
		ExpiresAt:        expandTimePtr(d.Get("expires_at")),
	}

	_, err = rdbAPI.UpdateDatabaseBackup(req, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForRDBDatabaseBackup(ctx, rdbAPI, region, id, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceScalewayRdbDatabaseBackupRead(ctx, d, meta)
}

func resourceScalewayRdbDatabaseBackupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	rdbAPI, region, id, err := rdbAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForRDBDatabaseBackup(ctx, rdbAPI, region, id, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = rdbAPI.DeleteDatabaseBackup(&rdb.DeleteDatabaseBackupRequest{
		DatabaseBackupID: id,
		Region:           region,
	}, scw.WithContext(ctx))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	_, err = waitForRDBDatabaseBackup(ctx, rdbAPI, region, id, d.Timeout(schema.TimeoutUpdate))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
