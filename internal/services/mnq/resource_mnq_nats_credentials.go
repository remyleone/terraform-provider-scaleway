package mnq

import (
	"context"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/difffuncs"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	mnq "github.com/scaleway/scaleway-sdk-go/api/mnq/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceScalewayMNQNatsCredentials() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayMNQNatsCredentialsCreate,
		ReadContext:   ResourceScalewayMNQNatsCredentialsRead,
		UpdateContext: ResourceScalewayMNQNatsCredentialsUpdate,
		DeleteContext: ResourceScalewayMNQNatsCredentialsDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"account_id": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "ID of the nats account",
				DiffSuppressFunc: difffuncs.DiffSuppressFuncLocality,
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "The nats credentials name",
			},
			"file": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The credentials file",
			},
			"region": locality.RegionalSchema(),
		},
	}
}

func ResourceScalewayMNQNatsCredentialsCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := NewMNQNatsAPI(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	credentials, err := api.CreateNatsCredentials(&mnq.NatsAPICreateNatsCredentialsRequest{
		Region:        region,
		NatsAccountID: locality.ExpandID(d.Get("account_id").(string)),
		Name:          types.ExpandOrGenerateString(d.Get("name").(string), "nats-credentials"),
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set("file", credentials.Credentials.Content)

	d.SetId(locality.NewRegionalIDString(region, credentials.ID))

	return ResourceScalewayMNQNatsCredentialsRead(ctx, d, meta)
}

func ResourceScalewayMNQNatsCredentialsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := MnqNatsAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	credentials, err := api.GetNatsCredentials(&mnq.NatsAPIGetNatsCredentialsRequest{
		Region:            region,
		NatsCredentialsID: id,
	}, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("account_id", credentials.NatsAccountID)
	_ = d.Set("name", credentials.Name)
	_ = d.Set("region", region)

	return nil
}

func ResourceScalewayMNQNatsCredentialsUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := MnqNatsAPIWithRegionAndID(meta, d.Id())
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

func ResourceScalewayMNQNatsCredentialsDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := MnqNatsAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = api.DeleteNatsCredentials(&mnq.NatsAPIDeleteNatsCredentialsRequest{
		Region:            region,
		NatsCredentialsID: id,
	}, scw.WithContext(ctx))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
