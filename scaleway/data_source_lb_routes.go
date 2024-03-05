package scaleway

import (
	"context"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/types"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/locality/zonal"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/project"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/organization"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/api/lb/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func dataSourceScalewayLbRoutes() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceScalewayLbRoutesRead,
		Schema: map[string]*schema.Schema{
			"frontend_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Routes with a frontend id like it are listed.",
			},
			"routes": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"frontend_id": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"backend_id": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"match_sni": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"match_host_header": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"created_at": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"update_at": {
							Computed: true,
							Type:     schema.TypeString,
						},
					},
				},
			},
			"zone":            zonal.Schema(),
			"organization_id": organization.OrganizationIDSchema(),
			"project_id":      project.ProjectIDSchema(),
		},
	}
}

func dataSourceScalewayLbRoutesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lbAPI, zone, err := lbAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	_, frontID, err := zonal.ParseZonedID(d.Get("frontend_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := lbAPI.ListRoutes(&lb.ZonedAPIListRoutesRequest{
		Zone:       zone,
		FrontendID: types.ExpandStringPtr(frontID),
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	routes := []interface{}(nil)
	for _, route := range res.Routes {
		rawRoute := make(map[string]interface{})
		rawRoute["id"] = newZonedID(zone, route.ID).String()
		rawRoute["frontend_id"] = route.FrontendID
		rawRoute["backend_id"] = route.BackendID
		rawRoute["created_at"] = flattenTime(route.CreatedAt)
		rawRoute["update_at"] = flattenTime(route.UpdatedAt)
		rawRoute["match_sni"] = flattenStringPtr(route.Match.Sni)
		rawRoute["match_host_header"] = flattenStringPtr(route.Match.HostHeader)

		routes = append(routes, rawRoute)
	}

	d.SetId(zone.String())
	_ = d.Set("routes", routes)

	return nil
}
