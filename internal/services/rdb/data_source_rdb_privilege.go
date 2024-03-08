package rdb

import (
	"context"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/datasource"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceScalewayRDBPrivilege() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasource.DatasourceSchemaFromResourceSchema(ResourceScalewayRdbPrivilege().Schema)

	datasource.FixDatasourceSchemaFlags(dsSchema, true, "instance_id", "user_name", "database_name")

	// Set 'Optional' schema elements
	datasource.AddOptionalFieldsToSchema(dsSchema, "region")
	return &schema.Resource{
		ReadContext: DataSourceScalewayRDBPrivilegeRead,
		Schema:      dsSchema,
	}
}

// dataSourceScalewayRDBPrivilegeRead
func DataSourceScalewayRDBPrivilegeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	_, region, err := RdbAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	instanceID := locality.ExpandID(d.Get("instance_id").(string))
	userName, _ := d.Get("user_name").(string)
	databaseName, _ := d.Get("database_name").(string)

	d.SetId(ResourceScalewayRdbUserPrivilegeID(region, instanceID, databaseName, userName))
	return ResourceScalewayRdbPrivilegeRead(ctx, d, meta)
}
