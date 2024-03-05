package scaleway

import (
	"context"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/scaleway/errors"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/project"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/organization"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/types"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	container "github.com/scaleway/scaleway-sdk-go/api/container/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/api/registry/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func resourceScalewayContainerNamespace() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScalewayContainerNamespaceCreate,
		ReadContext:   resourceScalewayContainerNamespaceRead,
		UpdateContext: resourceScalewayContainerNamespaceUpdate,
		DeleteContext: resourceScalewayContainerNamespaceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(defaultContainerNamespaceTimeout),
			Read:    schema.DefaultTimeout(defaultContainerNamespaceTimeout),
			Update:  schema.DefaultTimeout(defaultContainerNamespaceTimeout),
			Delete:  schema.DefaultTimeout(defaultContainerNamespaceTimeout),
			Default: schema.DefaultTimeout(defaultContainerNamespaceTimeout),
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				ForceNew:    true,
				Optional:    true,
				Description: "The name of the container namespace",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the container namespace",
			},
			"environment_variables": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "The environment variables of the container namespace",
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringLenBetween(0, 1000),
				},
				ValidateDiagFunc: validation.MapKeyLenBetween(0, 100),
			},
			"secret_environment_variables": {
				Type:        schema.TypeMap,
				Optional:    true,
				Sensitive:   true,
				Description: "The secret environment variables of the container namespace",
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringLenBetween(0, 1000),
				},
				ValidateDiagFunc: validation.MapKeyLenBetween(0, 100),
			},
			"registry_endpoint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The endpoint reachable by docker",
			},
			"registry_namespace_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the registry namespace",
			},
			"destroy_registry": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Destroy registry on deletion",
				Deprecated:  "Registry namespace is automatically destroyed with namespace",
			},
			"region":          regionSchema(),
			"organization_id": organization.OrganizationIDSchema(),
			"project_id":      project.ProjectIDSchema(),
		},
	}
}

func resourceScalewayContainerNamespaceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := containerAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	ns, err := api.CreateNamespace(&container.CreateNamespaceRequest{
		Description:                types.ExpandStringPtr(d.Get("description").(string)),
		EnvironmentVariables:       expandMapPtrStringString(d.Get("environment_variables")),
		SecretEnvironmentVariables: expandContainerSecrets(d.Get("secret_environment_variables")),
		Name:                       types.ExpandOrGenerateString(d.Get("name").(string), "ns"),
		ProjectID:                  d.Get("project_id").(string),
		Region:                     region,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(newRegionalIDString(region, ns.ID))

	_, err = waitForContainerNamespace(ctx, api, region, ns.ID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceScalewayContainerNamespaceRead(ctx, d, meta)
}

func resourceScalewayContainerNamespaceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := containerAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ns, err := waitForContainerNamespace(ctx, api, region, id, d.Timeout(schema.TimeoutRead))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("description", flattenStringPtr(ns.Description))
	_ = d.Set("environment_variables", ns.EnvironmentVariables)
	_ = d.Set("name", ns.Name)
	_ = d.Set("organization_id", ns.OrganizationID)
	_ = d.Set("project_id", ns.ProjectID)
	_ = d.Set("region", ns.Region)
	_ = d.Set("registry_endpoint", ns.RegistryEndpoint)
	_ = d.Set("registry_namespace_id", ns.RegistryNamespaceID)

	return nil
}

func resourceScalewayContainerNamespaceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := containerAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ns, err := waitForContainerNamespace(ctx, api, region, id, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		return diag.FromErr(err)
	}

	req := &container.UpdateNamespaceRequest{
		Region:      ns.Region,
		NamespaceID: ns.ID,
	}

	if d.HasChange("description") {
		req.Description = expandUpdatedStringPtr(d.Get("description"))
	}

	if d.HasChanges("environment_variables") {
		req.EnvironmentVariables = expandMapPtrStringString(d.Get("environment_variables"))
	}

	if d.HasChange("secret_environment_variables") {
		req.SecretEnvironmentVariables = expandContainerSecrets(d.Get("secret_environment_variables"))
	}

	if _, err := api.UpdateNamespace(req, scw.WithContext(ctx)); err != nil {
		return diag.FromErr(err)
	}

	return resourceScalewayContainerNamespaceRead(ctx, d, meta)
}

func resourceScalewayContainerNamespaceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := containerAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForContainerNamespace(ctx, api, region, id, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_, err = api.DeleteNamespace(&container.DeleteNamespaceRequest{
		Region:      region,
		NamespaceID: id,
	}, scw.WithContext(ctx))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	_, err = waitForContainerNamespace(ctx, api, region, id, d.Timeout(schema.TimeoutDelete))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	d.SetId("")

	if destroy := d.Get("destroy_registry"); destroy != nil && destroy == true {
		registryAPI, region, err := registryAPIWithRegion(d, meta)
		if err != nil {
			return diag.FromErr(err)
		}

		registryID := d.Get("registry_namespace_id").(string)

		_, err = registryAPI.DeleteNamespace(&registry.DeleteNamespaceRequest{
			Region:      region,
			NamespaceID: registryID,
		})
		if err != nil && !http_errors.Is404Error(err) {
			return diag.FromErr(err)
		}
		_, err = waitForRegistryNamespace(ctx, registryAPI, region, registryID, d.Timeout(schema.TimeoutDelete))
		if err != nil && !http_errors.Is404Error(err) {
			return diag.FromErr(err)
		}
	}

	return nil
}
