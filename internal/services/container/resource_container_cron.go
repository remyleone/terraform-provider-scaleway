package container

import (
	"context"
	"fmt"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/verify"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	container "github.com/scaleway/scaleway-sdk-go/api/container/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceScalewayContainerCron() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayContainerCronCreate,
		ReadContext:   ResourceScalewayContainerCronRead,
		UpdateContext: ResourceScalewayContainerCronUpdate,
		DeleteContext: ResourceScalewayContainerCronDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(defaultContainerCronTimeout),
			Read:    schema.DefaultTimeout(defaultContainerCronTimeout),
			Update:  schema.DefaultTimeout(defaultContainerCronTimeout),
			Delete:  schema.DefaultTimeout(defaultContainerCronTimeout),
			Default: schema.DefaultTimeout(defaultContainerCronTimeout),
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"container_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The Container ID to link with your trigger.",
			},
			"schedule": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: verify.ValidateCronExpression(),
				Description:  "Cron format string, e.g. 0 * * * * or @hourly, as schedule time of its jobs to be created and executed.",
			},
			"args": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Cron arguments as json object to pass through during execution.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Cron job status.",
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Cron job name",
			},
			"region": locality.RegionalSchema(),
		},
	}
}

func ResourceScalewayContainerCronCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := containerAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	jsonObj, err := scw.DecodeJSONObject(d.Get("args").(string), scw.NoEscape)
	if err != nil {
		return diag.FromErr(err)
	}

	containerID := locality.ExpandID(d.Get("container_id").(string))
	schedule := d.Get("schedule").(string)
	req := &container.CreateCronRequest{
		ContainerID: containerID,
		Region:      region,
		Schedule:    schedule,
		Name:        types.ExpandStringPtr(d.Get("name")),
		Args:        &jsonObj,
	}

	res, err := api.CreateCron(req, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Info(ctx, fmt.Sprintf("[INFO] Submitted new cron job: %#v", res.Schedule))
	_, err = waitForContainerCron(ctx, api, res.ID, region, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}
	tflog.Info(ctx, "[INFO] cron job ready")

	d.SetId(locality.NewRegionalIDString(region, res.ID))

	return ResourceScalewayContainerCronRead(ctx, d, meta)
}

func ResourceScalewayContainerCronRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, containerCronID, err := ContainerAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	cron, err := waitForContainerCron(ctx, api, containerCronID, region, d.Timeout(schema.TimeoutRead))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	args, err := scw.EncodeJSONObject(*cron.Args, scw.NoEscape)
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set("container_id", locality.NewRegionalID(region, cron.ContainerID).String())
	_ = d.Set("schedule", cron.Schedule)
	_ = d.Set("args", args)
	_ = d.Set("status", cron.Status)
	_ = d.Set("name", cron.Name)

	return nil
}

func ResourceScalewayContainerCronUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, containerCronID, err := ContainerAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &container.UpdateCronRequest{
		ContainerID: scw.StringPtr(locality.ExpandID(d.Get("container_id"))),
		CronID:      locality.ExpandID(containerCronID),
		Region:      region,
	}

	shouldUpdate := false
	if d.HasChange("schedule") {
		req.Schedule = scw.StringPtr(d.Get("schedule").(string))
		shouldUpdate = true
	}

	if d.HasChange("args") {
		jsonObj, err := scw.DecodeJSONObject(d.Get("args").(string), scw.NoEscape)
		if err != nil {
			return diag.FromErr(err)
		}
		shouldUpdate = true
		req.Args = &jsonObj
	}

	if d.HasChange("name") {
		req.Name = scw.StringPtr(d.Get("name").(string))
		shouldUpdate = true
	}

	if shouldUpdate {
		cron, err := api.UpdateCron(req, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}

		tflog.Info(ctx, fmt.Sprintf("[INFO] Updated cron job: %#v", req.Schedule))
		_, err = waitForContainerCron(ctx, api, cron.ID, region, d.Timeout(schema.TimeoutUpdate))
		if err != nil {
			return diag.FromErr(err)
		}
	}
	tflog.Info(ctx, "[INFO] cron job ready")

	return ResourceScalewayContainerCronRead(ctx, d, meta)
}

func ResourceScalewayContainerCronDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, containerCronID, err := ContainerAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForContainerCron(ctx, api, containerCronID, region, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = api.DeleteCron(&container.DeleteCronRequest{
		Region: region,
		CronID: containerCronID,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}
	tflog.Info(ctx, "[INFO] cron job deleted")
	return nil
}
