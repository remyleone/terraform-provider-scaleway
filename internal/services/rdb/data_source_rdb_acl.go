package rdb

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/datasource"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
)

func DataSourceScalewayRDBACL() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasource.DatasourceSchemaFromResourceSchema(ResourceScalewayRdbACL().Schema)

	dsSchema["instance_id"].Computed = false
	dsSchema["instance_id"].Required = true

	// Set 'Optional' schema elements
	datasource.AddOptionalFieldsToSchema(dsSchema, "region")
	return &schema.Resource{
		ReadContext: DataSourceScalewayRDBACLRead,
		Schema:      dsSchema,
	}
}

func DataSourceScalewayRDBACLRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	_, region, err := rdbAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	instanceID, _ := d.GetOk("instance_id")

	_, _, err = locality.ParseLocalizedID(instanceID.(string))
	regionalID := instanceID
	if err != nil {
		regionalID = locality.DatasourceNewRegionalID(instanceID, region)
	}

	d.SetId(regionalID.(string))
	err = d.Set("instance_id", regionalID)
	if err != nil {
		return diag.FromErr(err)
	}
	return ResourceScalewayRdbACLRead(ctx, d, meta)
}
