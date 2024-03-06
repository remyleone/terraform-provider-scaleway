package documentdb

import (
	"context"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/datasource"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceScalewayDocumentDBDatabase() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasource.DatasourceSchemaFromResourceSchema(ResourceScalewayDocumentDBDatabase().Schema)

	datasource.AddOptionalFieldsToSchema(dsSchema, "name", "region")

	dsSchema["instance_id"].Required = true
	dsSchema["instance_id"].Computed = false

	return &schema.Resource{
		ReadContext: DataSourceScalewayDocumentDBDatabaseRead,
		Schema:      dsSchema,
	}
}

func DataSourceScalewayDocumentDBDatabaseRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	_, region, err := documentDBAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	instanceID := locality.ExpandID(d.Get("instance_id").(string))
	databaseName := d.Get("name").(string)

	id := ResourceScalewayDocumentDBDatabaseID(region, instanceID, databaseName)
	d.SetId(id)
	if err != nil {
		return diag.FromErr(err)
	}

	diags := ResourceScalewayDocumentDBDatabaseRead(ctx, d, meta)
	if diags != nil {
		return append(diags, diag.Errorf("failed to read database state")...)
	}

	if d.Id() == "" {
		return diag.Errorf("database (%s) not found", databaseName)
	}

	return nil
}
