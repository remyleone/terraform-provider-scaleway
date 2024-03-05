package scaleway

import (
	"context"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/locality/zonal"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/types"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/project"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/organization"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	flexibleip "github.com/scaleway/scaleway-sdk-go/api/flexibleip/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func dataSourceScalewayFlexibleIPs() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceScalewayFlexibleIPsRead,
		Schema: map[string]*schema.Schema{
			"server_ids": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "Flexible IPs that are attached to these server IDs are listed",
			},
			"tags": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "Flexible IPs with these exact tags are listed",
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
						"description": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"tags": {
							Computed: true,
							Type:     schema.TypeList,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"status": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"ip_address": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"reverse": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"mac_address": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "The MAC address of the server associated with this flexible IP",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "MAC address ID",
									},
									"mac_address": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "MAC address of the Virtual MAC",
									},
									"mac_type": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Type of virtual MAC",
									},
									"status": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Status of virtual MAC",
									},
									"created_at": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The date on which the virtual MAC was created (RFC 3339 format)",
									},
									"updated_at": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The date on which the virtual MAC was last updated (RFC 3339 format)",
									},
									"zone": zonal.Schema(),
								},
							},
						},
						"created_at": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"updated_at": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"zone":            zoneComputedSchema(),
						"organization_id": organization.OrganizationIDSchema(),
						"project_id":      project.ProjectIDSchema(),
					},
				},
			},
			"zone":            zonal.Schema(),
			"organization_id": organization.OrganizationIDSchema(),
			"project_id":      project.ProjectIDSchema(),
		},
	}
}

func dataSourceScalewayFlexibleIPsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	fipAPI, zone, err := fipAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := fipAPI.ListFlexibleIPs(&flexibleip.ListFlexibleIPsRequest{
		Zone:      zone,
		ServerIDs: expandServerIDs(d.Get("server_ids")),
		ProjectID: types.ExpandStringPtr(d.Get("project_id")),
		Tags:      expandStrings(d.Get("tags")),
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	fips := []interface{}(nil)
	for _, fip := range res.FlexibleIPs {
		rawFip := make(map[string]interface{})
		rawFip["id"] = newZonedID(fip.Zone, fip.ID).String()
		rawFip["organization_id"] = fip.OrganizationID
		rawFip["project_id"] = fip.ProjectID
		rawFip["description"] = fip.Description
		if len(fip.Tags) > 0 {
			rawFip["tags"] = fip.Tags
		}
		rawFip["created_at"] = flattenTime(fip.CreatedAt)
		rawFip["updated_at"] = flattenTime(fip.UpdatedAt)
		rawFip["status"] = fip.Status
		ip, err := flattenIPNet(fip.IPAddress)
		if err != nil {
			return diag.FromErr(err)
		}
		rawFip["ip_address"] = ip
		if fip.MacAddress != nil {
			rawFip["mac_address"] = flattenFlexibleIPMacAddress(fip.MacAddress)
		}
		rawFip["reverse"] = fip.Reverse
		rawFip["zone"] = string(zone)

		fips = append(fips, rawFip)
	}

	d.SetId(zone.String())
	_ = d.Set("ips", fips)

	return nil
}
