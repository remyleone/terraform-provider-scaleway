package lb

import (
	"context"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/project"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/organization"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/scaleway/scaleway-sdk-go/api/lb/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func DataSourceScalewayLbIPs() *schema.Resource {
	return &schema.Resource{
		ReadContext: DataSourceScalewayLbIPsRead,
		Schema: map[string]*schema.Schema{
			"ip_cidr_range": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsCIDR,
				Description:  "IPs within a CIDR block like it are listed.",
			},
			"ips": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"ip_address": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"lb_id": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"reverse": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"zone":            locality.ZonalSchema(),
						"organization_id": organization.OrganizationIDSchema(),
						"project_id":      project.ProjectIDSchema(),
					},
				},
			},
			"zone":            locality.ZonalSchema(),
			"organization_id": organization.OrganizationIDSchema(),
			"project_id":      project.ProjectIDSchema(),
		},
	}
}

func DataSourceScalewayLbIPsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lbAPI, zone, err := LbAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	res, err := lbAPI.ListIPs(&lb.ZonedAPIListIPsRequest{
		Zone:      zone,
		ProjectID: types.ExpandStringPtr(d.Get("project_id")),
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	var filteredList []*lb.IP
	for i := range res.IPs {
		if ipv4Match(d.Get("ip_cidr_range").(string), res.IPs[i].IPAddress) {
			filteredList = append(filteredList, res.IPs[i])
		}
	}

	ips := []interface{}(nil)
	for _, ip := range filteredList {
		rawIP := make(map[string]interface{})
		rawIP["id"] = locality.NewZonedID(ip.Zone, ip.ID).String()
		rawIP["ip_address"] = ip.IPAddress
		rawIP["lb_id"] = types.FlattenStringPtr(ip.LBID)
		rawIP["reverse"] = ip.Reverse
		rawIP["zone"] = string(zone)
		rawIP["organization_id"] = ip.OrganizationID
		rawIP["project_id"] = ip.ProjectID

		ips = append(ips, rawIP)
	}

	d.SetId(zone.String())
	_ = d.Set("ips", ips)

	return nil
}
