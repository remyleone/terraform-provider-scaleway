package function

import (
	"context"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/difffuncs"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/verify"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceScalewayFunctionCron() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayFunctionCronCreate,
		ReadContext:   ResourceScalewayFunctionCronRead,
		UpdateContext: ResourceScalewayFunctionCronUpdate,
		DeleteContext: ResourceScalewayFunctionCronDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Default: schema.DefaultTimeout(defaultFunctionCronTimeout),
			Read:    schema.DefaultTimeout(defaultFunctionCronTimeout),
			Update:  schema.DefaultTimeout(defaultFunctionCronTimeout),
			Delete:  schema.DefaultTimeout(defaultFunctionCronTimeout),
			Create:  schema.DefaultTimeout(defaultFunctionCronTimeout),
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"function_id": {
				Type:        schema.TypeString,
				Description: "The ID of the function to create a cron for.",
				Required:    true,
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
				Description: "Functions arguments as json object to pass through during execution.",
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The name of the cron job.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Cron job status.",
			},
			"region": locality.RegionalSchema(),
		},
		CustomizeDiff: difffuncs.CustomizeDiffLocalityCheck("function_id"),
	}
}

func ResourceScalewayFunctionCronCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := functionAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	functionID := locality.ExpandID(d.Get("function_id").(string))
	f, err := waitForFunction(ctx, api, region, functionID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	request := &function.CreateCronRequest{
		FunctionID: f.ID,
		Schedule:   d.Get("schedule").(string),
		Region:     region,
		Name:       types.ExpandStringPtr(d.Get("name")),
	}

	if args, ok := d.GetOk("args"); ok {
		jsonObj, err := scw.DecodeJSONObject(args.(string), scw.NoEscape)
		if err != nil {
			return diag.FromErr(err)
		}
		request.Args = &jsonObj
	}

	cron, err := api.CreateCron(request, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForFunctionCron(ctx, api, region, cron.ID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(locality.NewRegionalIDString(region, cron.ID))

	return ResourceScalewayFunctionCronRead(ctx, d, meta)
}

func ResourceScalewayFunctionCronRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := functionAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	cron, err := waitForFunctionCron(ctx, api, region, id, d.Timeout(schema.TimeoutRead))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("function_id", locality.NewRegionalID(region, cron.FunctionID).String())
	_ = d.Set("schedule", cron.Schedule)
	_ = d.Set("name", cron.Name)
	args, err := scw.EncodeJSONObject(*cron.Args, scw.NoEscape)
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set("args", args)
	_ = d.Set("status", cron.Status)
	_ = d.Set("region", region.String())
	return nil
}

func ResourceScalewayFunctionCronUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := functionAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	cron, err := waitForFunctionCron(ctx, api, region, id, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		return diag.FromErr(err)
	}

	req := &function.UpdateCronRequest{
		Region: region,
		CronID: cron.ID,
	}
	shouldUpdate := false
	if d.HasChange("name") {
		req.Name = types.ExpandStringPtr(d.Get("name").(string))
		shouldUpdate = true
	}
	if d.HasChange("schedule") {
		req.Schedule = types.ExpandStringPtr(d.Get("schedule").(string))
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

	if shouldUpdate {
		_, err = api.UpdateCron(req, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return ResourceScalewayFunctionCronRead(ctx, d, meta)
}

func ResourceScalewayFunctionCronDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := functionAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	cron, err := waitForFunctionCron(ctx, api, region, id, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_, err = api.DeleteCron(&function.DeleteCronRequest{
		Region: region,
		CronID: cron.ID,
	}, scw.WithContext(ctx))

	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
