package scaleway

import (
	"context"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/scaleway/errors"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/locality/zonal"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/types"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/project"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/organization"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	lbSDK "github.com/scaleway/scaleway-sdk-go/api/lb/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func resourceScalewayLbIP() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScalewayLbIPCreate,
		ReadContext:   resourceScalewayLbIPRead,
		UpdateContext: resourceScalewayLbIPUpdate,
		DeleteContext: resourceScalewayLbIPDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Read:    schema.DefaultTimeout(defaultLbLbTimeout),
			Update:  schema.DefaultTimeout(defaultLbLbTimeout),
			Delete:  schema.DefaultTimeout(defaultLbLbTimeout),
			Default: schema.DefaultTimeout(defaultLbLbTimeout),
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{Version: 0, Type: lbUpgradeV1SchemaType(), Upgrade: lbUpgradeV1SchemaUpgradeFunc},
		},
		Schema: map[string]*schema.Schema{
			"reverse": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The reverse domain name for this IP",
			},
			"zone": zonal.Schema(),
			// Computed
			"organization_id": organization.OrganizationIDSchema(),
			"project_id":      project.ProjectIDSchema(),
			"ip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The load-balancer public IP address",
			},
			"lb_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the load balancer attached to this IP, if any",
			},
			"region": regionComputedSchema(),
		},
	}
}

func resourceScalewayLbIPCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lbAPI, zone, err := lbAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	zoneAttribute, ok := d.GetOk("zone")
	if ok {
		zone = scw.Zone(zoneAttribute.(string))
	}

	createReq := &lbSDK.ZonedAPICreateIPRequest{
		Zone:      zone,
		ProjectID: types.ExpandStringPtr(d.Get("project_id")),
		Reverse:   types.ExpandStringPtr(d.Get("reverse")),
	}

	res, err := lbAPI.CreateIP(createReq, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(zonal.NewZonedIDString(zone, res.ID))

	return resourceScalewayLbIPRead(ctx, d, meta)
}

func resourceScalewayLbIPRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lbAPI, zone, ID, err := lbAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ip, err := lbAPI.GetIP(&lbSDK.ZonedAPIGetIPRequest{
		Zone: zone,
		IPID: ID,
	}, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	// check lb state if it is attached
	if ip.LBID != nil {
		_, err = waitForLB(ctx, lbAPI, zone, *ip.LBID, d.Timeout(schema.TimeoutRead))
		if err != nil {
			if http_errors.Is403Error(err) {
				d.SetId("")
				return nil
			}
			return diag.FromErr(err)
		}
	}

	// set the region from zone
	region, err := zone.Region()
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set("region", string(region))
	_ = d.Set("zone", ip.Zone.String())
	_ = d.Set("organization_id", ip.OrganizationID)
	_ = d.Set("project_id", ip.ProjectID)
	_ = d.Set("ip_address", ip.IPAddress)
	_ = d.Set("reverse", ip.Reverse)
	_ = d.Set("lb_id", flattenStringPtr(ip.LBID))

	return nil
}

func resourceScalewayLbIPUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lbAPI, zone, ID, err := lbAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var ip *lbSDK.IP
	err = resource.RetryContext(ctx, d.Timeout(schema.TimeoutUpdate), func() *resource.RetryError {
		res, errGet := lbAPI.GetIP(&lbSDK.ZonedAPIGetIPRequest{
			Zone: zone,
			IPID: ID,
		}, scw.WithContext(ctx))
		if err != nil {
			if http_errors.Is403Error(errGet) {
				return resource.RetryableError(errGet)
			}
			return resource.NonRetryableError(errGet)
		}

		ip = res
		return nil
	})
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	if ip.LBID != nil {
		_, err = waitForLB(ctx, lbAPI, zone, *ip.LBID, d.Timeout(schema.TimeoutUpdate))
		if err != nil {
			if http_errors.Is403Error(err) {
				d.SetId("")
				return nil
			}
			return diag.FromErr(err)
		}
	}

	if d.HasChange("reverse") {
		req := &lbSDK.ZonedAPIUpdateIPRequest{
			Zone:    zone,
			IPID:    ID,
			Reverse: types.ExpandStringPtr(d.Get("reverse")),
		}

		_, err = lbAPI.UpdateIP(req, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if ip.LBID != nil {
		_, err = waitForLB(ctx, lbAPI, zone, *ip.LBID, d.Timeout(schema.TimeoutUpdate))
		if err != nil {
			if http_errors.Is403Error(err) {
				d.SetId("")
				return nil
			}
			return diag.FromErr(err)
		}
	}

	return resourceScalewayLbIPRead(ctx, d, meta)
}

//gocyclo:ignore
func resourceScalewayLbIPDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lbAPI, zone, ID, err := lbAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var ip *lbSDK.IP
	err = resource.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		res, errGet := lbAPI.GetIP(&lbSDK.ZonedAPIGetIPRequest{
			Zone: zone,
			IPID: ID,
		}, scw.WithContext(ctx))
		if err != nil {
			if http_errors.Is403Error(errGet) {
				return resource.RetryableError(errGet)
			}
			return resource.NonRetryableError(errGet)
		}

		ip = res
		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// check lb state
	if ip != nil && ip.LBID != nil {
		_, err = waitForLB(ctx, lbAPI, zone, *ip.LBID, d.Timeout(schema.TimeoutDelete))
		if err != nil {
			if http_errors.Is403Error(err) {
				d.SetId("")
				return nil
			}
			return diag.FromErr(err)
		}
	}

	err = lbAPI.ReleaseIP(&lbSDK.ZonedAPIReleaseIPRequest{
		Zone: zone,
		IPID: ID,
	}, scw.WithContext(ctx))

	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	// check lb state
	if ip != nil && ip.LBID != nil {
		_, err = waitForLB(ctx, lbAPI, zone, *ip.LBID, d.Timeout(schema.TimeoutDelete))
		if err != nil {
			if http_errors.Is404Error(err) || http_errors.Is403Error(err) {
				d.SetId("")
				return nil
			}
			return diag.FromErr(err)
		}
	}

	return nil
}
