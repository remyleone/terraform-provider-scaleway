package registry

import (
	"context"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/project"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/organization"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/api/registry/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceScalewayRegistryNamespace() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayRegistryNamespaceCreate,
		ReadContext:   ResourceScalewayRegistryNamespaceRead,
		UpdateContext: ResourceScalewayRegistryNamespaceUpdate,
		DeleteContext: ResourceScalewayRegistryNamespaceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(defaultRegistryNamespaceTimeout),
			Read:    schema.DefaultTimeout(defaultRegistryNamespaceTimeout),
			Update:  schema.DefaultTimeout(defaultRegistryNamespaceTimeout),
			Delete:  schema.DefaultTimeout(defaultRegistryNamespaceTimeout),
			Default: schema.DefaultTimeout(defaultRegistryNamespaceTimeout),
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the container registry namespace",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the container registry namespace",
			},
			"is_public": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Define the default visibity policy",
			},
			"endpoint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The endpoint reachable by docker",
			},
			"region":          locality.RegionalSchema(),
			"organization_id": organization.OrganizationIDSchema(),
			"project_id":      project.ProjectIDSchema(),
		},
	}
}

func ResourceScalewayRegistryNamespaceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := RegistryAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	ns, err := api.CreateNamespace(&registry.CreateNamespaceRequest{
		Region:      region,
		ProjectID:   types.ExpandStringPtr(d.Get("project_id")),
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		IsPublic:    d.Get("is_public").(bool),
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(locality.NewRegionalIDString(region, ns.ID))

	_, err = WaitForRegistryNamespace(ctx, api, region, ns.ID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayRegistryNamespaceRead(ctx, d, meta)
}

func ResourceScalewayRegistryNamespaceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := RegistryAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ns, err := WaitForRegistryNamespace(ctx, api, region, id, d.Timeout(schema.TimeoutRead))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", ns.Name)
	_ = d.Set("description", ns.Description)
	_ = d.Set("organization_id", ns.OrganizationID)
	_ = d.Set("project_id", ns.ProjectID)
	_ = d.Set("is_public", ns.IsPublic)
	_ = d.Set("endpoint", ns.Endpoint)
	_ = d.Set("region", ns.Region)

	return nil
}

func ResourceScalewayRegistryNamespaceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := RegistryAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = WaitForRegistryNamespace(ctx, api, region, id, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	if d.HasChanges("description", "is_public") {
		if _, err := api.UpdateNamespace(&registry.UpdateNamespaceRequest{
			Region:      region,
			NamespaceID: id,
			Description: types.ExpandUpdatedStringPtr(d.Get("description")),
			IsPublic:    scw.BoolPtr(d.Get("is_public").(bool)),
		}, scw.WithContext(ctx)); err != nil {
			return diag.FromErr(err)
		}
	}

	return ResourceScalewayRegistryNamespaceRead(ctx, d, meta)
}

func ResourceScalewayRegistryNamespaceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := RegistryAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = WaitForRegistryNamespace(ctx, api, region, id, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_, err = api.DeleteNamespace(&registry.DeleteNamespaceRequest{
		Region:      region,
		NamespaceID: id,
	}, scw.WithContext(ctx))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	_, err = WaitForRegistryNamespaceDelete(ctx, api, region, id, d.Timeout(schema.TimeoutDelete))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
