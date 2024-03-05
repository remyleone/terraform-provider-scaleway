package scaleway

import (
	"context"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/project"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	mnq "github.com/scaleway/scaleway-sdk-go/api/mnq/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func resourceScalewayMNQSNS() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScalewayMNQSNSCreate,
		ReadContext:   resourceScalewayMNQSNSRead,
		DeleteContext: resourceScalewayMNQSNSDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"endpoint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Endpoint of the SNS service",
			},
			"region":     regionSchema(),
			"project_id": project.ProjectIDSchema(),
		},
	}
}

func resourceScalewayMNQSNSCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := newMNQSNSAPI(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	sns, err := api.ActivateSns(&mnq.SnsAPIActivateSnsRequest{
		Region:    region,
		ProjectID: d.Get("project_id").(string),
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(newRegionalIDString(region, sns.ProjectID))

	return resourceScalewayMNQSNSRead(ctx, d, meta)
}

func resourceScalewayMNQSNSRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := mnqSNSAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	sns, err := api.GetSnsInfo(&mnq.SnsAPIGetSnsInfoRequest{
		Region:    region,
		ProjectID: id,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set("endpoint", sns.SnsEndpointURL)
	_ = d.Set("region", sns.Region)
	_ = d.Set("project_id", sns.ProjectID)

	return nil
}

func resourceScalewayMNQSNSDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := mnqSNSAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	sns, err := api.GetSnsInfo(&mnq.SnsAPIGetSnsInfoRequest{
		Region:    region,
		ProjectID: id,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	if sns.Status == mnq.SnsInfoStatusDisabled {
		d.SetId("")
		return nil
	}

	_, err = api.DeactivateSns(&mnq.SnsAPIDeactivateSnsRequest{
		Region:    region,
		ProjectID: id,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
