package container

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	container "github.com/scaleway/scaleway-sdk-go/api/container/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/difffuncs"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
)

func ResourceScalewayContainerDomain() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayContainerDomainCreate,
		ReadContext:   ResourceScalewayContainerDomainRead,
		DeleteContext: ResourceScalewayContainerDomainDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(defaultContainerDomainTimeout),
			Read:    schema.DefaultTimeout(defaultContainerDomainTimeout),
			Update:  schema.DefaultTimeout(defaultContainerDomainTimeout),
			Delete:  schema.DefaultTimeout(defaultContainerDomainTimeout),
			Default: schema.DefaultTimeout(defaultContainerDomainTimeout),
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"hostname": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Domain's hostname",
			},
			"container_id": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "Container the domain will be bound to",
				ValidateFunc:     locality.UUIDorUUIDWithLocality(),
				DiffSuppressFunc: difffuncs.DiffSuppressFuncLocality,
			},
			"url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL used to query the container",
			},
			"region": locality.RegionalSchema(),
		},
		CustomizeDiff: difffuncs.CustomizeDiffLocalityCheck("container_id"),
	}
}

func ResourceScalewayContainerDomainCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := containerAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	hostname := d.Get("hostname").(string)
	containerID := locality.ExpandID(d.Get("container_id"))

	_, err = waitForContainer(ctx, api, containerID, region, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	req := &container.CreateDomainRequest{
		Region:      region,
		Hostname:    hostname,
		ContainerID: containerID,
	}

	domain, err := retryCreateContainerDomain(ctx, api, req, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForContainerDomain(ctx, api, domain.ID, region, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(locality.NewRegionalIDString(region, domain.ID))

	return ResourceScalewayContainerDomainRead(ctx, d, meta)
}

func ResourceScalewayContainerDomainRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, domainID, err := ContainerAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	domain, err := waitForContainerDomain(ctx, api, domainID, region, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("hostname", domain.Hostname)
	_ = d.Set("container_id", domain.ContainerID)
	_ = d.Set("url", domain.URL)
	_ = d.Set("region", region)

	return nil
}

func ResourceScalewayContainerDomainDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, domainID, err := ContainerAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForContainerDomain(ctx, api, domainID, region, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_, err = api.DeleteDomain(&container.DeleteDomainRequest{
		Region:   region,
		DomainID: domainID,
	}, scw.WithContext(ctx))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
