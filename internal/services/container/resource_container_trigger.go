package container

import (
	"context"
	"fmt"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/difffuncs"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	container "github.com/scaleway/scaleway-sdk-go/api/container/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceScalewayContainerTrigger() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayContainerTriggerCreate,
		ReadContext:   ResourceScalewayContainerTriggerRead,
		UpdateContext: ResourceScalewayContainerTriggerUpdate,
		DeleteContext: ResourceScalewayContainerTriggerDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Default: schema.DefaultTimeout(defaultContainerTimeout),
			Read:    schema.DefaultTimeout(defaultContainerTimeout),
			Update:  schema.DefaultTimeout(defaultContainerTimeout),
			Delete:  schema.DefaultTimeout(defaultContainerTimeout),
			Create:  schema.DefaultTimeout(defaultContainerTimeout),
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"container_id": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The ID of the container to create a trigger for",
				ValidateFunc: locality.UUIDorUUIDWithLocality(),
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "The trigger name",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The trigger description",
			},
			"sqs": {
				Type:          schema.TypeList,
				MaxItems:      1,
				Description:   "Config for sqs based trigger using scaleway mnq",
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"nats"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"namespace_id": {
							Optional:         true,
							ForceNew:         true,
							Type:             schema.TypeString,
							Description:      "ID of the mnq namespace",
							DiffSuppressFunc: difffuncs.DiffSuppressFuncLocality,
						},
						"queue": {
							Required:    true,
							ForceNew:    true,
							Type:        schema.TypeString,
							Description: "Name of the queue",
						},
						"project_id": {
							Computed:    true,
							Optional:    true,
							ForceNew:    true,
							Type:        schema.TypeString,
							Description: "Project ID of the project where the mnq sqs exists, defaults to provider project_id",
						},
						"region": {
							Computed:    true,
							Optional:    true,
							ForceNew:    true,
							Type:        schema.TypeString,
							Description: "Region where the mnq sqs exists, defaults to function's region",
						},
					},
				},
			},
			"nats": {
				Type:          schema.TypeList,
				MaxItems:      1,
				Description:   "Config for nats based trigger using scaleway mnq",
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"sqs"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"account_id": {
							Optional:         true,
							ForceNew:         true,
							Type:             schema.TypeString,
							Description:      "ID of the mnq nats account",
							DiffSuppressFunc: difffuncs.DiffSuppressFuncLocality,
						},
						"subject": {
							Required:    true,
							ForceNew:    true,
							Type:        schema.TypeString,
							Description: "Subject to listen to",
						},
						"project_id": {
							Computed:    true,
							Optional:    true,
							ForceNew:    true,
							Type:        schema.TypeString,
							Description: "Project ID of the project where the mnq sqs exists, defaults to provider project_id",
						},
						"region": {
							Computed:    true,
							Optional:    true,
							ForceNew:    true,
							Type:        schema.TypeString,
							Description: "Region where the mnq sqs exists, defaults to function's region",
						},
					},
				},
			},
			"region": locality.RegionalSchema(),
		},
		CustomizeDiff: difffuncs.CustomizeDiffLocalityCheck("container_id"),
	}
}

func ResourceScalewayContainerTriggerCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := containerAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	req := &container.CreateTriggerRequest{
		Region:      region,
		Name:        types.ExpandOrGenerateString(d.Get("name").(string), "trigger"),
		ContainerID: locality.ExpandID(d.Get("container_id")),
		Description: types.ExpandStringPtr(d.Get("description")),
	}

	if scwSqs, isScwSqs := d.GetOk("sqs.0"); isScwSqs {
		err := completeContainerTriggerMnqCreationConfig(scwSqs, d, meta, region)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to complete sqs config: %w", err))
		}

		_ = d.Set("sqs", []any{scwSqs})
		req.ScwSqsConfig = expandContainerTriggerMnqSqsCreationConfig(scwSqs)
	}

	if scwNats, isScwNats := d.GetOk("nats.0"); isScwNats {
		err := completeContainerTriggerMnqCreationConfig(scwNats, d, meta, region)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to complete nats config: %w", err))
		}

		_ = d.Set("nats", []any{scwNats})
		req.ScwNatsConfig = expandContainerTriggerMnqNatsCreationConfig(scwNats)
	}

	trigger, err := api.CreateTrigger(req, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(locality.NewRegionalIDString(region, trigger.ID))

	_, err = waitForContainerTrigger(ctx, api, region, trigger.ID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayContainerTriggerRead(ctx, d, meta)
}

func ResourceScalewayContainerTriggerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := ContainerAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	trigger, err := waitForContainerTrigger(ctx, api, region, id, d.Timeout(schema.TimeoutRead))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", trigger.Name)
	_ = d.Set("description", trigger.Description)

	diags := diag.Diagnostics(nil)

	if trigger.Status == container.TriggerStatusError {
		errMsg := ""
		if trigger.ErrorMessage != nil {
			errMsg = *trigger.ErrorMessage
		}
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Trigger in error state",
			Detail:   errMsg,
		})
	}

	return diags
}

func ResourceScalewayContainerTriggerUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := ContainerAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	trigger, err := waitForContainerTrigger(ctx, api, region, id, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	req := &container.UpdateTriggerRequest{
		Region:    region,
		TriggerID: trigger.ID,
	}

	if d.HasChange("name") {
		req.Name = types.ExpandUpdatedStringPtr(d.Get("name"))
	}

	if d.HasChange("description") {
		req.Description = types.ExpandUpdatedStringPtr(d.Get("description"))
	}

	if _, err := api.UpdateTrigger(req, scw.WithContext(ctx)); err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayContainerTriggerRead(ctx, d, meta)
}

func ResourceScalewayContainerTriggerDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := ContainerAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForContainerTrigger(ctx, api, region, id, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = api.DeleteTrigger(&container.DeleteTriggerRequest{
		Region:    region,
		TriggerID: id,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForContainerTrigger(ctx, api, region, id, d.Timeout(schema.TimeoutDelete))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
