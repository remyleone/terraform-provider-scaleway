package mnq

import (
	"context"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/project"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	mnq "github.com/scaleway/scaleway-sdk-go/api/mnq/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceScalewayMNQSNS() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayMNQSNSCreate,
		ReadContext:   ResourceScalewayMNQSNSRead,
		DeleteContext: ResourceScalewayMNQSNSDelete,
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
			"region":     locality.RegionalSchema(),
			"project_id": project.ProjectIDSchema(),
		},
	}
}

func ResourceScalewayMNQSNSCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := NewSNSAPI(d, meta)
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

	d.SetId(locality.NewRegionalIDString(region, sns.ProjectID))

	return ResourceScalewayMNQSNSRead(ctx, d, meta)
}

func ResourceScalewayMNQSNSRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := MnqSNSAPIWithRegionAndID(meta, d.Id())
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

func ResourceScalewayMNQSNSDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := MnqSNSAPIWithRegionAndID(meta, d.Id())
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
