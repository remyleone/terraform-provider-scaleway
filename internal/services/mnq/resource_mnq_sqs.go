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

func ResourceScalewayMNQSQS() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayMNQSQSCreate,
		ReadContext:   ResourceScalewayMNQSQSRead,
		DeleteContext: ResourceScalewayMNQSQSDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"endpoint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Endpoint of the SQS service",
			},
			"region":     locality.RegionalSchema(),
			"project_id": project.ProjectIDSchema(),
		},
	}
}

func ResourceScalewayMNQSQSCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := newMNQSQSAPI(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	sqs, err := api.ActivateSqs(&mnq.SqsAPIActivateSqsRequest{
		Region:    region,
		ProjectID: d.Get("project_id").(string),
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(locality.NewRegionalIDString(region, sqs.ProjectID))

	return ResourceScalewayMNQSQSRead(ctx, d, meta)
}

func ResourceScalewayMNQSQSRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := mnqSQSAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	sqs, err := api.GetSqsInfo(&mnq.SqsAPIGetSqsInfoRequest{
		Region:    region,
		ProjectID: id,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set("endpoint", sqs.SqsEndpointURL)
	_ = d.Set("region", sqs.Region)
	_ = d.Set("project_id", sqs.ProjectID)

	return nil
}

func ResourceScalewayMNQSQSDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := mnqSQSAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	sqs, err := api.GetSqsInfo(&mnq.SqsAPIGetSqsInfoRequest{
		Region:    region,
		ProjectID: id,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	if sqs.Status == mnq.SqsInfoStatusDisabled {
		d.SetId("")
		return nil
	}

	_, err = api.DeactivateSqs(&mnq.SqsAPIDeactivateSqsRequest{
		Region:    region,
		ProjectID: id,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
