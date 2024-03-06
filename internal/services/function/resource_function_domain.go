package function

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/difffuncs"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
)

func ResourceScalewayFunctionDomain() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayFunctionDomainCreate,
		ReadContext:   ResourceScalewayFunctionDomainRead,
		DeleteContext: ResourceScalewayFunctionDomainDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Default: schema.DefaultTimeout(defaultFunctionTimeout),
			Read:    schema.DefaultTimeout(defaultFunctionTimeout),
			Update:  schema.DefaultTimeout(defaultFunctionTimeout),
			Delete:  schema.DefaultTimeout(defaultFunctionTimeout),
			Create:  schema.DefaultTimeout(defaultFunctionTimeout),
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"function_id": {
				Type:             schema.TypeString,
				Description:      "The ID of the function",
				Required:         true,
				ForceNew:         true,
				ValidateFunc:     locality.UUIDorUUIDWithLocality(),
				DiffSuppressFunc: difffuncs.DiffSuppressFuncLocality,
			},
			"hostname": {
				Type:        schema.TypeString,
				Description: "The hostname that should be redirected to the function",
				Required:    true,
				ForceNew:    true,
			},
			"url": {
				Type:        schema.TypeString,
				Description: "URL to use to trigger the function",
				Computed:    true,
			},
			"region": locality.RegionalSchema(),
		},
		CustomizeDiff: difffuncs.CustomizeDiffLocalityCheck("function_id"),
	}
}

func ResourceScalewayFunctionDomainCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := functionAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	functionID := locality.ExpandRegionalID(d.Get("function_id").(string)).ID
	_, err = waitForFunction(ctx, api, region, functionID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	hostname := d.Get("hostname").(string)

	req := &function.CreateDomainRequest{
		Region:     region,
		FunctionID: functionID,
		Hostname:   hostname,
	}

	domain, err := retryCreateFunctionDomain(ctx, api, req, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(locality.NewRegionalIDString(region, domain.ID))

	_, err = waitForFunctionDomain(ctx, api, region, domain.ID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayFunctionDomainRead(ctx, d, meta)
}

func ResourceScalewayFunctionDomainRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := functionAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	domain, err := waitForFunctionDomain(ctx, api, region, id, d.Timeout(schema.TimeoutRead))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("hostname", domain.Hostname)
	_ = d.Set("function_id", locality.NewRegionalIDString(region, domain.FunctionID))
	_ = d.Set("url", domain.URL)
	_ = d.Set("region", region)

	return nil
}

func ResourceScalewayFunctionDomainDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := functionAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForFunctionDomain(ctx, api, region, id, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return nil
	}

	_, err = api.DeleteDomain(&function.DeleteDomainRequest{
		DomainID: id,
		Region:   region,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForFunctionDomain(ctx, api, region, id, d.Timeout(schema.TimeoutDelete))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
