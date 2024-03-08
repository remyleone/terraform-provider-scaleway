package block

import (
	"context"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/difffuncs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"

	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/project"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	block "github.com/scaleway/scaleway-sdk-go/api/block/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceScalewayBlockVolume() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayBlockVolumeCreate,
		ReadContext:   ResourceScalewayBlockVolumeRead,
		UpdateContext: ResourceScalewayBlockVolumeUpdate,
		DeleteContext: ResourceScalewayBlockVolumeDelete,
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
				Description: "The volume name",
			},
			"iops": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The maximum IO/s expected, must match available options",
				ForceNew:    true,
			},
			"size_in_gb": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				Description:  "The volume size in GB",
				ExactlyOneOf: []string{"snapshot_id"}, // TODO: Allow size with snapshot to change created volume size
			},
			"snapshot_id": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Description:      "The snapshot to create the volume from",
				ExactlyOneOf:     []string{"size_in_gb"},
				DiffSuppressFunc: difffuncs.DiffSuppressFuncLocality,
			},
			"tags": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "The tags associated with the volume",
			},
			"zone":       locality.ZonalSchema(),
			"project_id": project.ProjectIDSchema(),
		},
		CustomizeDiff: customdiff.All(
			customDiffCannotShrink("size_in_gb"),
		),
	}
}

func ResourceScalewayBlockVolumeCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, zone, err := BlockAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	req := &block.CreateVolumeRequest{
		Zone:      zone,
		Name:      types.ExpandOrGenerateString(d.Get("name").(string), "volume"),
		ProjectID: d.Get("project_id").(string),
		Tags:      types.ExpandStrings(d.Get("tags")),
		PerfIops:  types.ExpandUint32Ptr(d.Get("iops")),
	}

	if iops, ok := d.GetOk("iops"); ok {
		req.PerfIops = types.ExpandUint32Ptr(iops)
	}

	if size, ok := d.GetOk("size_in_gb"); ok {
		volumeSizeInBytes := scw.Size(size.(int)) * scw.GB
		req.FromEmpty = &block.CreateVolumeRequestFromEmpty{
			Size: volumeSizeInBytes,
		}
	}

	if snapshotID, ok := d.GetOk("snapshot_id"); ok {
		req.FromSnapshot = &block.CreateVolumeRequestFromSnapshot{
			SnapshotID: locality.ExpandID(snapshotID.(string)),
		}
	}

	volume, err := api.CreateVolume(req, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(locality.NewZonedIDString(zone, volume.ID))

	_, err = waitForBlockVolume(ctx, api, zone, volume.ID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayBlockVolumeRead(ctx, d, meta)
}

func ResourceScalewayBlockVolumeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, zone, id, err := BlockAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	volume, err := waitForBlockVolume(ctx, api, zone, id, d.Timeout(schema.TimeoutRead))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", volume.Name)

	if volume.Specs != nil {
		_ = d.Set("iops", types.FlattenUint32Ptr(volume.Specs.PerfIops))
	}
	_ = d.Set("size_in_gb", int(volume.Size/scw.GB))
	_ = d.Set("zone", volume.Zone)
	_ = d.Set("project_id", volume.ProjectID)
	_ = d.Set("tags", volume.Tags)

	if volume.ParentSnapshotID != nil {
		_ = d.Set("snapshot_id", locality.NewZonedIDString(zone, *volume.ParentSnapshotID))
	} else {
		_ = d.Set("snapshot_id", "")
	}

	return nil
}

func ResourceScalewayBlockVolumeUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, zone, id, err := BlockAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	volume, err := waitForBlockVolume(ctx, api, zone, id, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	req := &block.UpdateVolumeRequest{
		Zone:     zone,
		VolumeID: volume.ID,
	}

	if d.HasChange("name") {
		req.Name = types.ExpandUpdatedStringPtr(d.Get("name"))
	}

	if d.HasChange("size") {
		volumeSizeInBytes := scw.Size(uint64(d.Get("size").(int)) * types.Gb)
		req.Size = &volumeSizeInBytes
	}

	if d.HasChange("tags") {
		req.Tags = types.ExpandUpdatedStringsPtr(d.Get("tags"))
	}

	if _, err := api.UpdateVolume(req, scw.WithContext(ctx)); err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayBlockVolumeRead(ctx, d, meta)
}

func ResourceScalewayBlockVolumeDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, zone, id, err := BlockAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForBlockVolume(ctx, api, zone, id, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		return diag.FromErr(err)
	}

	err = api.DeleteVolume(&block.DeleteVolumeRequest{
		Zone:     zone,
		VolumeID: id,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForBlockVolume(ctx, api, zone, id, d.Timeout(schema.TimeoutDelete))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
