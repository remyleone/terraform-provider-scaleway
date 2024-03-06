package mnq

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	mnq "github.com/scaleway/scaleway-sdk-go/api/mnq/v1beta1"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/datasource"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
)

func DataSourceScalewayMNQSQS() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasource.DatasourceSchemaFromResourceSchema(ResourceScalewayMNQSQS().Schema)

	datasource.AddOptionalFieldsToSchema(dsSchema, "region", "project_id")

	return &schema.Resource{
		ReadContext: DataSourceScalewayMNQSQSRead,
		Schema:      dsSchema,
	}
}

func DataSourceScalewayMNQSQSRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := newMNQSQSAPI(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	sqs, err := api.GetSqsInfo(&mnq.SqsAPIGetSqsInfoRequest{
		Region:    region,
		ProjectID: d.Get("project_id").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	if sqs.Status != mnq.SqsInfoStatusEnabled {
		return diag.FromErr(fmt.Errorf("expected mnq sqs status to be enabled, got: %s", sqs.Status))
	}

	regionID := locality.DatasourceNewRegionalID(sqs.ProjectID, region)
	d.SetId(regionID)

	diags := ResourceScalewayMNQSQSRead(ctx, d, meta)
	if diags != nil {
		return append(diags, diag.Errorf("failed to read sqs state")...)
	}

	if d.Id() == "" {
		return diag.Errorf("sqs (%s) not found", regionID)
	}

	return nil
}
