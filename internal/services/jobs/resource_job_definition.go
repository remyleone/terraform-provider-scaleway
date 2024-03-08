package jobs

import (
	"context"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/difffuncs"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"time"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/project"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	jobs "github.com/scaleway/scaleway-sdk-go/api/jobs/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceDefinition() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayJobDefinitionCreate,
		ReadContext:   ResourceScalewayJobDefinitionRead,
		UpdateContext: ResourceScalewayJobDefinitionUpdate,
		DeleteContext: ResourceScalewayJobDefinitionDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "The job name",
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"cpu_limit": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"memory_limit": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"image_uri": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"command": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"timeout": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				DiffSuppressFunc: difffuncs.DiffSuppressFuncDuration,
			},
			"env": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringLenBetween(0, 1000),
				},
				ValidateDiagFunc: validation.MapKeyLenBetween(0, 100),
			},
			"cron": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"schedule": {
							Type:         schema.TypeString,
							Required:     true,
							RequiredWith: []string{"cron.0"},
						},
						"timezone": {
							Type:         schema.TypeString,
							Required:     true,
							RequiredWith: []string{"cron.0"},
						},
					},
				},
			},
			"region":     locality.RegionalSchema(),
			"project_id": project.ProjectIDSchema(),
		},
	}
}

func ResourceScalewayJobDefinitionCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := NewAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	req := &jobs.CreateJobDefinitionRequest{
		Region:               region,
		Name:                 types.ExpandOrGenerateString(d.Get("name").(string), "job"),
		CPULimit:             uint32(d.Get("cpu_limit").(int)),
		MemoryLimit:          uint32(d.Get("memory_limit").(int)),
		ImageURI:             d.Get("image_uri").(string),
		Command:              d.Get("command").(string),
		ProjectID:            d.Get("project_id").(string),
		EnvironmentVariables: types.ExpandMapStringString(d.Get("env")),
		Description:          d.Get("description").(string),
		CronSchedule:         expandJobDefinitionCron(d.Get("cron")).ToCreateRequest(),
	}

	if timeoutSeconds, ok := d.GetOk("timeout"); ok {
		duration, err := time.ParseDuration(timeoutSeconds.(string))
		if err != nil {
			return diag.Diagnostics{{
				Severity:      diag.Error,
				Summary:       "Invalid timeout, expected Go duration format",
				Detail:        err.Error(),
				AttributePath: cty.GetAttrPath("timeout"),
			}}
		}

		req.JobTimeout = scw.NewDurationFromTimeDuration(duration)
	}

	definition, err := api.CreateJobDefinition(req, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(locality.NewRegionalIDString(region, definition.ID))

	return ResourceScalewayJobDefinitionRead(ctx, d, meta)
}

func ResourceScalewayJobDefinitionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := NewAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	definition, err := api.GetJobDefinition(&jobs.GetJobDefinitionRequest{
		JobDefinitionID: id,
		Region:          region,
	}, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", definition.Name)
	_ = d.Set("cpu_limit", int(definition.CPULimit))
	_ = d.Set("memory_limit", int(definition.MemoryLimit))
	_ = d.Set("image_uri", definition.ImageURI)
	_ = d.Set("command", definition.Command)
	_ = d.Set("env", types.FlattenMap(definition.EnvironmentVariables))
	_ = d.Set("description", definition.Description)
	_ = d.Set("timeout", definition.JobTimeout.ToTimeDuration().String())
	_ = d.Set("cron", flattenJobDefinitionCron(definition.CronSchedule))
	_ = d.Set("region", definition.Region)
	_ = d.Set("project_id", definition.ProjectID)

	return nil
}

func ResourceScalewayJobDefinitionUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := NewAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &jobs.UpdateJobDefinitionRequest{
		Region:          region,
		JobDefinitionID: id,
	}

	if d.HasChange("name") {
		req.Name = types.ExpandUpdatedStringPtr(d.Get("name"))
	}

	if d.HasChange("cpu_limit") {
		req.CPULimit = types.ExpandUint32Ptr(d.Get("cpu_limit"))
	}

	if d.HasChange("memory_limit") {
		req.MemoryLimit = types.ExpandUint32Ptr(d.Get("memory_limit"))
	}

	if d.HasChange("image_uri") {
		req.ImageURI = types.ExpandUpdatedStringPtr(d.Get("image_uri"))
	}

	if d.HasChange("command") {
		req.Command = types.ExpandUpdatedStringPtr(d.Get("command"))
	}

	if d.HasChange("env") {
		req.EnvironmentVariables = types.ExpandMapPtrStringString(d.Get("env"))
	}

	if d.HasChange("description") {
		req.Description = types.ExpandUpdatedStringPtr(d.Get("description"))
	}

	if d.HasChange("timeout") {
		if timeoutSeconds, ok := d.GetOk("timeout"); ok {
			duration, err := time.ParseDuration(timeoutSeconds.(string))
			if err != nil {
				return diag.Diagnostics{{
					Severity:      diag.Error,
					Summary:       "Invalid timeout, expected Go duration format",
					Detail:        err.Error(),
					AttributePath: cty.GetAttrPath("timeout"),
				}}
			}

			req.JobTimeout = scw.NewDurationFromTimeDuration(duration)
		}
	}

	if d.HasChange("cron") {
		req.CronSchedule = expandJobDefinitionCron(d.Get("cron")).ToUpdateRequest()
	}

	if _, err := api.UpdateJobDefinition(req, scw.WithContext(ctx)); err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayJobDefinitionRead(ctx, d, meta)
}

func ResourceScalewayJobDefinitionDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := NewAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = api.DeleteJobDefinition(&jobs.DeleteJobDefinitionRequest{
		Region:          region,
		JobDefinitionID: id,
	}, scw.WithContext(ctx))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}
