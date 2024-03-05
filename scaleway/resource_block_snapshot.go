package scaleway

import (
	"context"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/verify"

	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/scaleway/errors"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/locality/zonal"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/project"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/types"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	block "github.com/scaleway/scaleway-sdk-go/api/block/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func resourceScalewayBlockSnapshot() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScalewayBlockSnapshotCreate,
		ReadContext:   resourceScalewayBlockSnapshotRead,
		UpdateContext: resourceScalewayBlockSnapshotUpdate,
		DeleteContext: resourceScalewayBlockSnapshotDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(defaultBlockTimeout),
			Read:    schema.DefaultTimeout(defaultBlockTimeout),
			Delete:  schema.DefaultTimeout(defaultBlockTimeout),
			Default: schema.DefaultTimeout(defaultBlockTimeout),
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "The snapshot name",
			},
			"volume_id": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateFunc:     verify.UUIDorUUIDWithLocality(),
				Description:      "ID of the volume from which creates a snapshot",
				DiffSuppressFunc: diffSuppressFuncLocality,
			},
			"tags": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "The tags associated with the snapshot",
			},
			"zone":       zonal.Schema(),
			"project_id": project.ProjectIDSchema(),
		},
	}
}

func resourceScalewayBlockSnapshotCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, zone, err := blockAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	snapshot, err := api.CreateSnapshot(&block.CreateSnapshotRequest{
		Zone:      zone,
		ProjectID: d.Get("project_id").(string),
		Name:      types.ExpandOrGenerateString(d.Get("name").(string), "snapshot"),
		VolumeID:  locality.ExpandID(d.Get("volume_id")),
		Tags:      expandStrings(d.Get("tags")),
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(zonal.NewZonedIDString(zone, snapshot.ID))

	_, err = waitForBlockSnapshot(ctx, api, zone, snapshot.ID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceScalewayBlockSnapshotRead(ctx, d, meta)
}

func resourceScalewayBlockSnapshotRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, zone, id, err := blockAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	snapshot, err := waitForBlockSnapshot(ctx, api, zone, id, d.Timeout(schema.TimeoutRead))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", snapshot.Name)
	_ = d.Set("zone", snapshot.Zone)
	_ = d.Set("project_id", snapshot.ProjectID)
	if snapshot.ParentVolume != nil {
		_ = d.Set("volume_id", snapshot.ParentVolume.ID)
	} else {
		_ = d.Set("volume_id", "")
	}
	_ = d.Set("tags", snapshot.Tags)

	return nil
}

func resourceScalewayBlockSnapshotUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, zone, id, err := blockAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	snapshot, err := waitForBlockSnapshot(ctx, api, zone, id, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	req := &block.UpdateSnapshotRequest{
		Zone:       zone,
		SnapshotID: snapshot.ID,
	}

	if d.HasChange("name") {
		req.Name = expandUpdatedStringPtr(d.Get("name"))
	}

	if d.HasChange("tags") {
		req.Tags = expandUpdatedStringsPtr(d.Get("tags"))
	}

	if _, err := api.UpdateSnapshot(req, scw.WithContext(ctx)); err != nil {
		return diag.FromErr(err)
	}

	return resourceScalewayBlockSnapshotRead(ctx, d, meta)
}

func resourceScalewayBlockSnapshotDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, zone, id, err := blockAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForBlockSnapshot(ctx, api, zone, id, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		return diag.FromErr(err)
	}

	err = api.DeleteSnapshot(&block.DeleteSnapshotRequest{
		Zone:       zone,
		SnapshotID: id,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForBlockSnapshot(ctx, api, zone, id, d.Timeout(schema.TimeoutDelete))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
