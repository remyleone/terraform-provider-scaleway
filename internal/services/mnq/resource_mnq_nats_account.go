package mnq

import (
	"context"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/project"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	mnq "github.com/scaleway/scaleway-sdk-go/api/mnq/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceScalewayMNQNatsAccount() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayMNQNatsAccountCreate,
		ReadContext:   ResourceScalewayMNQNatsAccountRead,
		UpdateContext: ResourceScalewayMNQNatsAccountUpdate,
		DeleteContext: ResourceScalewayMNQNatsAccountDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "The nats account name",
			},
			"endpoint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The endpoint for interact with Nats",
			},
			"region":     locality.RegionalSchema(),
			"project_id": project.ProjectIDSchema(),
		},
	}
}

func ResourceScalewayMNQNatsAccountCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := newMNQNatsAPI(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	account, err := api.CreateNatsAccount(&mnq.NatsAPICreateNatsAccountRequest{
		Region:    region,
		ProjectID: d.Get("project_id").(string),
		Name:      types.ExpandOrGenerateString(d.Get("name").(string), "nats-account"),
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(locality.NewRegionalIDString(region, account.ID))

	return ResourceScalewayMNQNatsAccountRead(ctx, d, meta)
}

func ResourceScalewayMNQNatsAccountRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := mnqNatsAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	account, err := api.GetNatsAccount(&mnq.NatsAPIGetNatsAccountRequest{
		Region:        region,
		NatsAccountID: id,
	}, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", account.Name)
	_ = d.Set("region", account.Region)
	_ = d.Set("project_id", account.ProjectID)
	_ = d.Set("endpoint", account.Endpoint)

	return nil
}

func ResourceScalewayMNQNatsAccountUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := mnqNatsAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &mnq.NatsAPIUpdateNatsAccountRequest{
		Region:        region,
		NatsAccountID: id,
	}

	if d.HasChange("name") {
		req.Name = types.ExpandUpdatedStringPtr(d.Get("name"))
	}

	if _, err := api.UpdateNatsAccount(req, scw.WithContext(ctx)); err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayMNQNatsAccountRead(ctx, d, meta)
}

func ResourceScalewayMNQNatsAccountDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := mnqNatsAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = api.DeleteNatsAccount(&mnq.NatsAPIDeleteNatsAccountRequest{
		Region:        region,
		NatsAccountID: id,
	}, scw.WithContext(ctx))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
