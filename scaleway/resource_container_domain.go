package scaleway

import (
	"context"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/scaleway/errors"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/verify"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	container "github.com/scaleway/scaleway-sdk-go/api/container/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func resourceScalewayContainerDomain() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScalewayContainerDomainCreate,
		ReadContext:   resourceScalewayContainerDomainRead,
		DeleteContext: resourceScalewayContainerDomainDelete,
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
				ValidateFunc:     verify.UUIDorUUIDWithLocality(),
				DiffSuppressFunc: diffSuppressFuncLocality,
			},
			"url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL used to query the container",
			},
			"region": regionSchema(),
		},
		CustomizeDiff: customizeDiffLocalityCheck("container_id"),
	}
}

func resourceScalewayContainerDomainCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	d.SetId(newRegionalIDString(region, domain.ID))

	return resourceScalewayContainerDomainRead(ctx, d, meta)
}

func resourceScalewayContainerDomainRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, domainID, err := containerAPIWithRegionAndID(meta, d.Id())
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

func resourceScalewayContainerDomainDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, domainID, err := containerAPIWithRegionAndID(meta, d.Id())
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
