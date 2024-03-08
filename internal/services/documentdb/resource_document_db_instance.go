package documentdb

import (
	"context"
	"errors"
	"fmt"
	documentdb "github.com/scaleway/scaleway-sdk-go/api/documentdb/v1beta1"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/difffuncs"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/project"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceScalewayDocumentDBInstance() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayDocumentDBInstanceCreate,
		ReadContext:   ResourceScalewayDocumentDBInstanceRead,
		UpdateContext: ResourceScalewayDocumentDBInstanceUpdate,
		DeleteContext: ResourceScalewayDocumentDBInstanceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(defaultDocumentDBInstanceTimeout),
			Read:    schema.DefaultTimeout(defaultDocumentDBInstanceTimeout),
			Update:  schema.DefaultTimeout(defaultDocumentDBInstanceTimeout),
			Delete:  schema.DefaultTimeout(defaultDocumentDBInstanceTimeout),
			Default: schema.DefaultTimeout(defaultDocumentDBInstanceTimeout),
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "The document db instance name",
			},
			"node_type": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "The type of database instance you want to create",
				DiffSuppressFunc: difffuncs.DiffSuppressFuncIgnoreCase,
			},
			"engine": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Database's engine version id",
			},
			"is_ha_cluster": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable or disable high availability for the database instance",
			},
			"user_name": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Optional:    true,
				Description: "Identifier for the first user of the database instance",
			},
			"password": {
				Type:        schema.TypeString,
				Sensitive:   true,
				Optional:    true,
				ForceNew:    true,
				Description: "Password for the first user of the database instance",
			},
			"volume_type": {
				Type:     schema.TypeString,
				Default:  documentdb.VolumeTypeBssd,
				Optional: true,
				ValidateFunc: validation.StringInSlice([]string{
					documentdb.VolumeTypeLssd.String(),
					documentdb.VolumeTypeBssd.String(),
				}, false),
				Description: "Type of volume where data are stored",
			},
			"volume_size_in_gb": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Volume size (in GB) when volume_type is not lssd",
			},
			"tags": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "List of tags [\"tag1\", \"tag2\", ...] attached to a database instance",
			},
			"telemetry_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: " Enable telemetry to collects basic anonymous usage data and sends them to FerretDB telemetry service",
			},
			"region":     locality.RegionalSchema(),
			"project_id": project.ProjectIDSchema(),
		},
	}
}

func ResourceScalewayDocumentDBInstanceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := DocumentDBAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	createReq := &documentdb.CreateInstanceRequest{
		Region:      region,
		ProjectID:   types.ExpandStringPtr(d.Get("project_id")),
		Name:        types.ExpandOrGenerateString(d.Get("name").(string), "document-instance"),
		NodeType:    d.Get("node_type").(string),
		Engine:      d.Get("engine").(string),
		IsHaCluster: d.Get("is_ha_cluster").(bool),
		UserName:    d.Get("user_name").(string),
		Password:    d.Get("password").(string),
		Tags:        types.ExpandStrings(d.Get("tags")),
		VolumeType:  documentdb.VolumeType(d.Get("volume_type").(string)),
	}

	if size, ok := d.GetOk("volume_size_in_gb"); ok {
		if createReq.VolumeType != documentdb.VolumeTypeBssd {
			return diag.FromErr(fmt.Errorf("volume_size_in_gb should be used with volume_type %s only", documentdb.VolumeTypeBssd.String()))
		}
		createReq.VolumeSize = scw.Size(uint64(size.(int)) * uint64(scw.GB))
	}

	if d.Get("telemetry_enabled").(bool) {
		createReq.InitSettings = append(createReq.InitSettings, &documentdb.InstanceSetting{
			Name:  telemetryDocumentDBReporting,
			Value: "true",
		})
	}

	instance, err := api.CreateInstance(createReq, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(locality.NewRegionalIDString(region, instance.ID))

	_, err = waitForDocumentDBInstance(ctx, api, region, instance.ID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayDocumentDBInstanceRead(ctx, d, meta)
}

func ResourceScalewayDocumentDBInstanceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := DocumentDBAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	instance, err := waitForDocumentDBInstance(ctx, api, region, id, d.Timeout(schema.TimeoutRead))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", instance.Name)
	_ = d.Set("node_type", instance.NodeType)
	_ = d.Set("engine", instance.Engine)
	_ = d.Set("is_ha_cluster", instance.IsHaCluster)
	_ = d.Set("region", instance.Region)
	_ = d.Set("project_id", instance.ProjectID)
	_ = d.Set("tags", instance.Tags)

	if instance.Volume != nil {
		_ = d.Set("volume_type", instance.Volume.Type)
		_ = d.Set("volume_size_in_gb", int(instance.Volume.Size/scw.GB))
	}

	_ = setInitSettings(d, instance.InitSettings)

	return nil
}

func setInitSettings(d *schema.ResourceData, settings []*documentdb.InstanceSetting) error {
	for _, s := range settings {
		if s.Name == telemetryDocumentDBReporting {
			_ = d.Set("telemetry_enabled", s.Value)
		}
	}

	return nil
}

func ResourceScalewayDocumentDBInstanceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := DocumentDBAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	instance, err := waitForDocumentDBInstance(ctx, api, region, id, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	req := &documentdb.UpdateInstanceRequest{
		Region:     region,
		InstanceID: instance.ID,
	}

	if d.HasChange("name") {
		req.Name = types.ExpandUpdatedStringPtr(d.Get("name"))
	}

	if d.HasChange("tags") {
		req.Tags = types.ExpandUpdatedStringsPtr(d.Get("tags"))
	}

	_, err = waitForDocumentDBInstance(ctx, api, region, id, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		return diag.FromErr(err)
	}

	if _, err := api.UpdateInstance(req, scw.WithContext(ctx)); err != nil {
		return diag.FromErr(err)
	}

	upgradeRequests := []*documentdb.UpgradeInstanceRequest(nil)

	if d.HasChanges("volume_type", "volume_size_in_gb") {
		volType := documentdb.VolumeType(d.Get("volume_type").(string))

		switch volType {
		case documentdb.VolumeTypeBssd:
			if d.HasChange("volume_type") {
				upgradeRequests = append(upgradeRequests,
					&documentdb.UpgradeInstanceRequest{
						Region:     region,
						InstanceID: id,
						VolumeType: &volType,
					})
			}
			if d.HasChange("volume_size_in_gb") {
				oldSizeInterface, newSizeInterface := d.GetChange("volume_size_in_gb")
				oldSize := uint64(oldSizeInterface.(int))
				newSize := uint64(newSizeInterface.(int))
				if newSize < oldSize {
					return diag.FromErr(errors.New("volume_size_in_gb cannot be decreased"))
				}

				if newSize%5 != 0 {
					return diag.FromErr(errors.New("volume_size_in_gb must be a multiple of 5"))
				}

				upgradeRequests = append(upgradeRequests,
					&documentdb.UpgradeInstanceRequest{
						Region:     region,
						InstanceID: id,
						VolumeSize: scw.Uint64Ptr(newSize * uint64(scw.GB)),
					})
			}
		case documentdb.VolumeTypeLssd:
			_, ok := d.GetOk("volume_size_in_gb")
			if d.HasChange("volume_size_in_gb") && ok {
				return diag.FromErr(fmt.Errorf("volume_size_in_gb should be used with volume_type %s only", documentdb.VolumeTypeBssd.String()))
			}
			if d.HasChange("volume_type") {
				upgradeRequests = append(upgradeRequests,
					&documentdb.UpgradeInstanceRequest{
						Region:     region,
						InstanceID: id,
						VolumeType: &volType,
					})
			}
		default:
			return diag.FromErr(fmt.Errorf("unknown volume_type %s", volType.String()))
		}

		if d.HasChanges("node_type") {
			upgradeRequests = append(upgradeRequests, &documentdb.UpgradeInstanceRequest{
				Region:     region,
				InstanceID: id,
				NodeType:   types.ExpandStringPtr(d.Get("node_type")),
			})
		}

		if d.HasChange("is_ha_cluster") {
			upgradeRequests = append(upgradeRequests, &documentdb.UpgradeInstanceRequest{
				Region:     region,
				InstanceID: id,
				EnableHa:   types.ExpandBoolPtr(d.Get("is_ha_cluster")),
			})
		}
	}

	for _, upgradeRequest := range upgradeRequests {
		_, err = waitForDocumentDBInstance(ctx, api, region, id, d.Timeout(schema.TimeoutUpdate))
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = api.UpgradeInstance(upgradeRequest, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = waitForDocumentDBInstance(ctx, api, region, id, d.Timeout(schema.TimeoutUpdate))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return ResourceScalewayDocumentDBInstanceRead(ctx, d, meta)
}

func ResourceScalewayDocumentDBInstanceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := DocumentDBAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForDocumentDBInstance(ctx, api, region, id, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = api.DeleteInstance(&documentdb.DeleteInstanceRequest{
		Region:     region,
		InstanceID: id,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForDocumentDBInstance(ctx, api, region, id, d.Timeout(schema.TimeoutDelete))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
