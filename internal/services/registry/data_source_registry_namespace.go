package registry

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/api/registry/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/datasource"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
)

func DataSourceScalewayRegistryNamespace() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasource.DatasourceSchemaFromResourceSchema(ResourceScalewayRegistryNamespace().Schema)

	datasource.AddOptionalFieldsToSchema(dsSchema, "name", "region", "project_id")

	dsSchema["name"].ConflictsWith = []string{"namespace_id"}
	dsSchema["namespace_id"] = &schema.Schema{
		Type:          schema.TypeString,
		Optional:      true,
		Description:   "The ID of the registry namespace",
		ValidateFunc:  locality.UUIDorUUIDWithLocality(),
		ConflictsWith: []string{"name"},
	}

	return &schema.Resource{
		ReadContext: DataSourceScalewayRegistryNamespaceRead,

		Schema: dsSchema,
	}
}

func DataSourceScalewayRegistryNamespaceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := RegistryAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	namespaceID, ok := d.GetOk("namespace_id")
	if !ok {
		namespaceName := d.Get("name").(string)
		res, err := api.ListNamespaces(&registry.ListNamespacesRequest{
			Region:    region,
			Name:      types.ExpandStringPtr(namespaceName),
			ProjectID: types.ExpandStringPtr(d.Get("project_id")),
		}, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}

		foundNamespace, err := datasource.FindExact(
			res.Namespaces,
			func(s *registry.Namespace) bool { return s.Name == namespaceName },
			namespaceName,
		)
		if err != nil {
			return diag.FromErr(err)
		}

		namespaceID = foundNamespace.ID
	}

	regionalID := locality.DatasourceNewRegionalID(namespaceID, region)
	d.SetId(regionalID)
	_ = d.Set("namespace_id", regionalID)

	return ResourceScalewayRegistryNamespaceRead(ctx, d, meta)
}
