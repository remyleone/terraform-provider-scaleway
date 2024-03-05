package scaleway

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	container "github.com/scaleway/scaleway-sdk-go/api/container/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/types"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/verify"
)

func dataSourceScalewayContainer() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasourceSchemaFromResourceSchema(resourceScalewayContainer().Schema)

	addOptionalFieldsToSchema(dsSchema, "name", "region")

	dsSchema["name"].ConflictsWith = []string{"container_id"}
	dsSchema["container_id"] = &schema.Schema{
		Type:          schema.TypeString,
		Optional:      true,
		Description:   "The ID of the Container",
		ValidateFunc:  verify.UUIDorUUIDWithLocality(),
		ConflictsWith: []string{"name"},
	}
	dsSchema["namespace_id"] = &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		Description:  "The ID of the Container namespace",
		ValidateFunc: verify.UUIDorUUIDWithLocality(),
	}
	dsSchema["project_id"] = &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		Description:  "The ID of the project to filter the Container",
		ValidateFunc: verify.UUID(),
	}

	return &schema.Resource{
		ReadContext: dataSourceScalewayContainerRead,
		Schema:      dsSchema,
	}
}

func dataSourceScalewayContainerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := containerAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	containerID, ok := d.GetOk("container_id")
	namespaceID := d.Get("namespace_id")
	if !ok {
		containerName := d.Get("name").(string)
		res, err := api.ListContainers(&container.ListContainersRequest{
			Region:      region,
			Name:        types.ExpandStringPtr(containerName),
			NamespaceID: locality.ExpandID(namespaceID),
			ProjectID:   types.ExpandStringPtr(d.Get("project_id")),
		}, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}

		foundContainer, err := findExact(
			res.Containers,
			func(s *container.Container) bool { return s.Name == containerName },
			containerName,
		)
		if err != nil {
			return diag.FromErr(err)
		}

		containerID = foundContainer.ID
	}

	regionalID := datasourceNewRegionalID(containerID, region)
	d.SetId(regionalID)
	_ = d.Set("container_id", regionalID)

	return resourceScalewayContainerRead(ctx, d, meta)
}
