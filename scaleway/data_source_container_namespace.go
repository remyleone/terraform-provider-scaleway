package scaleway

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	container "github.com/scaleway/scaleway-sdk-go/api/container/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/types"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/verify"
)

func dataSourceScalewayContainerNamespace() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasourceSchemaFromResourceSchema(resourceScalewayContainerNamespace().Schema)

	addOptionalFieldsToSchema(dsSchema, "name", "region", "project_id")

	dsSchema["name"].ConflictsWith = []string{"namespace_id"}
	dsSchema["namespace_id"] = &schema.Schema{
		Type:          schema.TypeString,
		Optional:      true,
		Description:   "The ID of the Container namespace",
		ValidateFunc:  verify.UUIDorUUIDWithLocality(),
		ConflictsWith: []string{"name"},
	}

	return &schema.Resource{
		ReadContext: dataSourceScalewayContainerNamespaceRead,
		Schema:      dsSchema,
	}
}

func dataSourceScalewayContainerNamespaceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := containerAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	namespaceID, ok := d.GetOk("namespace_id")
	if !ok {
		namespaceName := d.Get("name").(string)
		res, err := api.ListNamespaces(&container.ListNamespacesRequest{
			Region:    region,
			Name:      types.ExpandStringPtr(namespaceName),
			ProjectID: types.ExpandStringPtr(d.Get("project_id")),
		}, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}

		foundNamespace, err := findExact(
			res.Namespaces,
			func(s *container.Namespace) bool { return s.Name == namespaceName },
			namespaceName,
		)
		if err != nil {
			return diag.FromErr(err)
		}

		namespaceID = foundNamespace.ID
	}

	regionalID := datasourceNewRegionalID(namespaceID, region)
	d.SetId(regionalID)
	_ = d.Set("namespace_id", regionalID)

	return resourceScalewayContainerNamespaceRead(ctx, d, meta)
}
