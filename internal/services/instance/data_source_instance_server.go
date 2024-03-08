package instance

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	instanceSDK "github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/datasource"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
)

func DataSourceScalewayInstanceServer() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasource.DatasourceSchemaFromResourceSchema(ResourceScalewayInstanceServer().Schema)

	// Set 'Optional' schema elements
	datasource.AddOptionalFieldsToSchema(dsSchema, "name", "zone", "project_id")

	dsSchema["name"].ConflictsWith = []string{"server_id"}
	dsSchema["server_id"] = &schema.Schema{
		Type:          schema.TypeString,
		Optional:      true,
		Description:   "The ID of the server",
		ValidateFunc:  locality.UUIDorUUIDWithLocality(),
		ConflictsWith: []string{"name"},
	}

	return &schema.Resource{
		ReadContext: DataSourceScalewayInstanceServerRead,

		Schema: dsSchema,
	}
}

func DataSourceScalewayInstanceServerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	instanceAPI, zone, err := InstanceAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	serverID, ok := d.GetOk("server_id")
	if !ok {
		serverName := d.Get("name").(string)
		res, err := instanceAPI.ListServers(&instanceSDK.ListServersRequest{
			Zone:    zone,
			Name:    types.ExpandStringPtr(serverName),
			Project: types.ExpandStringPtr(d.Get("project_id")),
		}, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}

		foundServer, err := datasource.FindExact(
			res.Servers,
			func(s *instanceSDK.Server) bool { return s.Name == serverName },
			serverName,
		)
		if err != nil {
			return diag.FromErr(err)
		}

		serverID = foundServer.ID
	}

	zonedID := locality.DatasourceNewZonedID(serverID, zone)
	d.SetId(zonedID)
	_ = d.Set("server_id", zonedID)
	return ResourceScalewayInstanceServerRead(ctx, d, meta)
}
