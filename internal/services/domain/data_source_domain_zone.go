package domain

import (
	"context"
	"fmt"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/datasource"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceScalewayDomainZone() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasource.DatasourceSchemaFromResourceSchema(ResourceScalewayDomainZone().Schema)

	datasource.AddOptionalFieldsToSchema(dsSchema, "domain", "subdomain")

	return &schema.Resource{
		ReadContext: DataSourceScalewayDomainZoneRead,
		Schema:      dsSchema,
	}
}

func DataSourceScalewayDomainZoneRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	d.SetId(fmt.Sprintf("%s.%s", d.Get("subdomain").(string), d.Get("domain").(string)))

	return ResourceScalewayDomainZoneRead(ctx, d, meta)
}
