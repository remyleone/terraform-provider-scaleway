package lb

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/datasource"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
)

func DataSourceScalewayLbRoute() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasource.DatasourceSchemaFromResourceSchema(ResourceScalewayLbRoute().Schema)

	dsSchema["route_id"] = &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		Description:  "The ID of the route",
		ValidateFunc: locality.UUIDorUUIDWithLocality(),
	}

	return &schema.Resource{
		ReadContext: DataSourceScalewayLbRouteRead,
		Schema:      dsSchema,
	}
}

func DataSourceScalewayLbRouteRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	_, zone, err := lbAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	routeID, _ := d.GetOk("route_id")

	zonedID := locality.DatasourceNewZonedID(routeID, zone)
	d.SetId(zonedID)
	err = d.Set("route_id", zonedID)
	if err != nil {
		return diag.FromErr(err)
	}
	return ResourceScalewayLbRouteRead(ctx, d, meta)
}
