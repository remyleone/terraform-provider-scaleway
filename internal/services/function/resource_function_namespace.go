package function

import (
	"context"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/project"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/organization"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceScalewayFunctionNamespace() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayFunctionNamespaceCreate,
		ReadContext:   ResourceScalewayFunctionNamespaceRead,
		UpdateContext: ResourceScalewayFunctionNamespaceUpdate,
		DeleteContext: ResourceScalewayFunctionNamespaceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(defaultFunctionNamespaceTimeout),
			Read:    schema.DefaultTimeout(defaultFunctionNamespaceTimeout),
			Update:  schema.DefaultTimeout(defaultFunctionNamespaceTimeout),
			Delete:  schema.DefaultTimeout(defaultFunctionNamespaceTimeout),
			Default: schema.DefaultTimeout(defaultFunctionNamespaceTimeout),
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				ForceNew:    true,
				Optional:    true,
				Description: "The name of the function namespace",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the function namespace",
			},
			"environment_variables": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "The environment variables of the function namespace",
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
				Description: "The environment variables of the function namespace",
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
			"region":          locality.RegionalSchema(),
			"organization_id": organization.OrganizationIDSchema(),
			"project_id":      project.ProjectIDSchema(),
		},
	}
}

func ResourceScalewayFunctionNamespaceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := functionAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	ns, err := api.CreateNamespace(&function.CreateNamespaceRequest{
		Description:                types.ExpandStringPtr(d.Get("description").(string)),
		EnvironmentVariables:       types.ExpandMapPtrStringString(d.Get("environment_variables")),
		SecretEnvironmentVariables: expandFunctionsSecrets(d.Get("secret_environment_variables")),
		Name:                       types.ExpandOrGenerateString(d.Get("name").(string), "func"),
		ProjectID:                  d.Get("project_id").(string),
		Region:                     region,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(locality.NewRegionalIDString(region, ns.ID))

	_, err = waitForFunctionNamespace(ctx, api, region, ns.ID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayFunctionNamespaceRead(ctx, d, meta)
}

func ResourceScalewayFunctionNamespaceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := FunctionAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ns, err := waitForFunctionNamespace(ctx, api, region, id, d.Timeout(schema.TimeoutRead))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("description", ns.Description)
	_ = d.Set("environment_variables", ns.EnvironmentVariables)
	_ = d.Set("name", ns.Name)
	_ = d.Set("organization_id", ns.OrganizationID)
	_ = d.Set("project_id", ns.ProjectID)
	_ = d.Set("region", ns.Region)
	_ = d.Set("registry_endpoint", ns.RegistryEndpoint)
	_ = d.Set("registry_namespace_id", ns.RegistryNamespaceID)

	return nil
}

func ResourceScalewayFunctionNamespaceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := FunctionAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ns, err := waitForFunctionNamespace(ctx, api, region, id, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	req := &function.UpdateNamespaceRequest{
		Region:      ns.Region,
		NamespaceID: ns.ID,
	}

	if d.HasChange("description") {
		req.Description = types.ExpandUpdatedStringPtr(d.Get("description"))
	}

	if d.HasChanges("environment_variables") {
		req.EnvironmentVariables = types.ExpandMapPtrStringString(d.Get("environment_variables"))
	}

	if d.HasChanges("secret_environment_variables") {
		req.SecretEnvironmentVariables = expandFunctionsSecrets(d.Get("secret_environment_variables"))
	}

	if _, err := api.UpdateNamespace(req, scw.WithContext(ctx)); err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayFunctionNamespaceRead(ctx, d, meta)
}

func ResourceScalewayFunctionNamespaceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := FunctionAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForFunctionNamespace(ctx, api, region, id, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = api.DeleteNamespace(&function.DeleteNamespaceRequest{
		Region:      region,
		NamespaceID: id,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForFunctionNamespace(ctx, api, region, id, d.Timeout(schema.TimeoutDelete))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
