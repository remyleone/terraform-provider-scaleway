package lb

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	lbSDK "github.com/scaleway/scaleway-sdk-go/api/lb/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/datasource"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
)

func DataSourceScalewayLbFrontend() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasource.DatasourceSchemaFromResourceSchema(ResourceScalewayLbFrontend().Schema)

	// Set 'Optional' schema elements
	datasource.AddOptionalFieldsToSchema(dsSchema, "name", "lb_id")

	dsSchema["name"].ConflictsWith = []string{"frontend_id"}
	dsSchema["frontend_id"] = &schema.Schema{
		Type:          schema.TypeString,
		Optional:      true,
		Description:   "The ID of the frontend",
		ValidateFunc:  locality.UUIDorUUIDWithLocality(),
		ConflictsWith: []string{"name"},
	}

	return &schema.Resource{
		ReadContext: DataSourceScalewayLbFrontendRead,
		Schema:      dsSchema,
	}
}

func DataSourceScalewayLbFrontendRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, zone, err := lbAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	frontID, ok := d.GetOk("frontend_id")
	if !ok { // Get LB by name.
		frontName := d.Get("name").(string)
		res, err := api.ListFrontends(&lbSDK.ZonedAPIListFrontendsRequest{
			Zone: zone,
			Name: types.ExpandStringPtr(frontName),
			LBID: locality.ExpandID(d.Get("lb_id")),
		}, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}

		foundFront, err := datasource.FindExact(
			res.Frontends,
			func(s *lbSDK.Frontend) bool { return s.Name == frontName },
			frontName,
		)
		if err != nil {
			return diag.FromErr(err)
		}

		frontID = foundFront.ID
	}
	zonedID := locality.DatasourceNewZonedID(frontID, zone)
	d.SetId(zonedID)
	err = d.Set("frontend_id", zonedID)
	if err != nil {
		return diag.FromErr(err)
	}
	return ResourceScalewayLbFrontendRead(ctx, d, meta)
}
