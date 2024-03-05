package scaleway

import (
	"context"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/scaleway/errors"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/project"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/types"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	mnq "github.com/scaleway/scaleway-sdk-go/api/mnq/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func resourceScalewayMNQNatsAccount() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScalewayMNQNatsAccountCreate,
		ReadContext:   resourceScalewayMNQNatsAccountRead,
		UpdateContext: resourceScalewayMNQNatsAccountUpdate,
		DeleteContext: resourceScalewayMNQNatsAccountDelete,
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
			"region":     regionSchema(),
			"project_id": project.ProjectIDSchema(),
		},
	}
}

func resourceScalewayMNQNatsAccountCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	d.SetId(newRegionalIDString(region, account.ID))

	return resourceScalewayMNQNatsAccountRead(ctx, d, meta)
}

func resourceScalewayMNQNatsAccountRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

func resourceScalewayMNQNatsAccountUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := mnqNatsAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &mnq.NatsAPIUpdateNatsAccountRequest{
		Region:        region,
		NatsAccountID: id,
	}

	if d.HasChange("name") {
		req.Name = expandUpdatedStringPtr(d.Get("name"))
	}

	if _, err := api.UpdateNatsAccount(req, scw.WithContext(ctx)); err != nil {
		return diag.FromErr(err)
	}

	return resourceScalewayMNQNatsAccountRead(ctx, d, meta)
}

func resourceScalewayMNQNatsAccountDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
