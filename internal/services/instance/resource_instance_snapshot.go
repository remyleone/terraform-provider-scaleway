package instance

import (
	"context"
	"fmt"
	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/difffuncs"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/transport"
	"time"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/project"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/organization"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceScalewayInstanceSnapshot() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayInstanceSnapshotCreate,
		ReadContext:   ResourceScalewayInstanceSnapshotRead,
		UpdateContext: ResourceScalewayInstanceSnapshotUpdate,
		DeleteContext: ResourceScalewayInstanceSnapshotDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(defaultInstanceSnapshotWaitTimeout),
			Delete:  schema.DefaultTimeout(defaultInstanceSnapshotWaitTimeout),
			Default: schema.DefaultTimeout(defaultInstanceSnapshotWaitTimeout),
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The name of the snapshot",
			},
			"volume_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Description:   "ID of the volume to take a snapshot from",
				ValidateFunc:  locality.UUIDorUUIDWithLocality(),
				ConflictsWith: []string{"import"},
			},
			"type": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "The snapshot's volume type",
				ValidateFunc: validation.StringInSlice([]string{
					instance.SnapshotVolumeTypeUnknownVolumeType.String(),
					instance.SnapshotVolumeTypeBSSD.String(),
					instance.SnapshotVolumeTypeLSSD.String(),
					instance.SnapshotVolumeTypeUnified.String(),
				}, false),
			},
			"size_in_gb": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The size of the snapshot in gigabyte",
			},
			"tags": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "The tags associated with the snapshot",
			},
			"import": {
				Type:     schema.TypeList,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bucket": {
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
							Description: "Bucket containing qcow",
						},
						"key": {
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
							Description: "Key of the qcow file in the specified bucket",
						},
					},
				},
				Optional:      true,
				Description:   "Import snapshot from a qcow",
				ConflictsWith: []string{"volume_id"},
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the creation of the snapshot",
			},
			"zone":            locality.ZonalSchema(),
			"organization_id": organization.OrganizationIDSchema(),
			"project_id":      project.ProjectIDSchema(),
		},
		CustomizeDiff: difffuncs.CustomizeDiffLocalityCheck("volume_id"),
	}
}

func ResourceScalewayInstanceSnapshotCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	instanceAPI, zone, err := InstanceAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	req := &instance.CreateSnapshotRequest{
		Zone:    zone,
		Project: types.ExpandStringPtr(d.Get("project_id")),
		Name:    types.ExpandOrGenerateString(d.Get("name"), "snap"),
	}

	if volumeType, ok := d.GetOk("type"); ok {
		volumeType := instance.SnapshotVolumeType(volumeType.(string))
		req.VolumeType = volumeType
	}

	req.Tags = types.ExpandStringsPtr(d.Get("tags"))

	if volumeID, volumeIDExist := d.GetOk("volume_id"); volumeIDExist {
		req.VolumeID = scw.StringPtr(locality.ExpandZonedID(volumeID).ID)
	}

	if _, isImported := d.GetOk("import"); isImported {
		req.Bucket = types.ExpandStringPtr(d.Get("import.0.bucket"))
		req.Key = types.ExpandStringPtr(d.Get("import.0.key"))
	}

	res, err := instanceAPI.CreateSnapshot(req, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(locality.NewZonedIDString(zone, res.Snapshot.ID))

	_, err = instanceAPI.WaitForSnapshot(&instance.WaitForSnapshotRequest{
		SnapshotID:    res.Snapshot.ID,
		Zone:          zone,
		RetryInterval: transport.DefaultWaitRetryInterval,
		Timeout:       scw.TimeDurationPtr(d.Timeout(schema.TimeoutCreate)),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayInstanceSnapshotRead(ctx, d, meta)
}

func ResourceScalewayInstanceSnapshotRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	instanceAPI, zone, id, err := InstanceAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	snapshot, err := instanceAPI.GetSnapshot(&instance.GetSnapshotRequest{
		SnapshotID: id,
		Zone:       zone,
	}, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", snapshot.Snapshot.Name)
	_ = d.Set("created_at", snapshot.Snapshot.CreationDate.Format(time.RFC3339))
	_ = d.Set("type", snapshot.Snapshot.VolumeType.String())
	_ = d.Set("tags", snapshot.Snapshot.Tags)

	return nil
}

func ResourceScalewayInstanceSnapshotUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	instanceAPI, zone, id, err := InstanceAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &instance.UpdateSnapshotRequest{
		SnapshotID: id,
		Zone:       zone,
		Name:       scw.StringPtr(d.Get("name").(string)),
		Tags:       scw.StringsPtr([]string{}),
	}

	tags := types.ExpandStrings(d.Get("tags"))
	if d.HasChange("tags") && len(tags) > 0 {
		req.Tags = scw.StringsPtr(types.ExpandStrings(d.Get("tags")))
	}

	_, err = instanceAPI.UpdateSnapshot(req, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(fmt.Errorf("couldn't update snapshot: %s", err))
	}

	return ResourceScalewayInstanceSnapshotRead(ctx, d, meta)
}

func ResourceScalewayInstanceSnapshotDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	instanceAPI, zone, id, err := InstanceAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForInstanceSnapshot(ctx, instanceAPI, zone, id, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		return diag.FromErr(err)
	}

	err = instanceAPI.DeleteSnapshot(&instance.DeleteSnapshotRequest{
		SnapshotID: id,
		Zone:       zone,
	}, scw.WithContext(ctx))
	if err != nil {
		if !http_errors.Is404Error(err) {
			return diag.FromErr(err)
		}
	}

	_, err = waitForInstanceSnapshot(ctx, instanceAPI, zone, id, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		if !http_errors.Is404Error(err) {
			return diag.FromErr(err)
		}
	}

	return nil
}
