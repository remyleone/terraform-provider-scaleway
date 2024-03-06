package lb

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	lbSDK "github.com/scaleway/scaleway-sdk-go/api/lb/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/datasource"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
)

func DataSourceScalewayLbIP() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasource.DatasourceSchemaFromResourceSchema(ResourceScalewayLbIP().Schema)

	dsSchema["ip_address"] = &schema.Schema{
		Type:          schema.TypeString,
		Optional:      true,
		Description:   "The IP address",
		ConflictsWith: []string{"ip_id"},
	}
	dsSchema["ip_id"] = &schema.Schema{
		Type:          schema.TypeString,
		Optional:      true,
		Description:   "The ID of the IP address",
		ConflictsWith: []string{"ip_address"},
		ValidateFunc:  locality.UUIDorUUIDWithLocality(),
	}
	dsSchema["project_id"].Optional = true

	return &schema.Resource{
		ReadContext:   DataSourceScalewayLbIPRead,
		Schema:        dsSchema,
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{Version: 0, Type: lbUpgradeV1SchemaType(), Upgrade: lbUpgradeV1SchemaUpgradeFunc},
		},
	}
}

func DataSourceScalewayLbIPRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, zone, err := lbAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	ipID, ok := d.GetOk("ip_id")
	if !ok { // Get IP by region and IP address.
		res, err := api.ListIPs(&lbSDK.ZonedAPIListIPsRequest{
			Zone:      zone,
			IPAddress: types.ExpandStringPtr(d.Get("ip_address")),
			ProjectID: types.ExpandStringPtr(d.Get("project_id")),
		}, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
		if len(res.IPs) == 0 {
			return diag.FromErr(fmt.Errorf("no ips found with the address %s", d.Get("ip_address")))
		}
		if len(res.IPs) > 1 {
			return diag.FromErr(fmt.Errorf("%d ips found with the same address %s", len(res.IPs), d.Get("ip_address")))
		}
		ipID = res.IPs[0].ID
	}

	zoneID := locality.DatasourceNewZonedID(ipID, zone)
	d.SetId(zoneID)
	err = d.Set("ip_id", zoneID)
	if err != nil {
		return diag.FromErr(err)
	}
	return ResourceScalewayLbIPRead(ctx, d, meta)
}
