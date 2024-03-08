package instance

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/difffuncs"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
)

func ResourceScalewayInstanceIPReverseDNS() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayInstanceIPReverseDNSCreate,
		ReadContext:   ResourceScalewayInstanceIPReverseDNSRead,
		UpdateContext: ResourceScalewayInstanceIPReverseDNSUpdate,
		DeleteContext: ResourceScalewayInstanceIPReverseDNSDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Default: schema.DefaultTimeout(defaultInstanceIPTimeout),
			Create:  schema.DefaultTimeout(defaultInstanceIPReverseDNSTimeout),
			Update:  schema.DefaultTimeout(defaultInstanceIPReverseDNSTimeout),
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"ip_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The IP ID or IP address",
			},
			"reverse": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The reverse DNS for this IP",
			},
			"zone": locality.ZonalSchema(),
		},
		CustomizeDiff: difffuncs.CustomizeDiffLocalityCheck("ip_id"),
	}
}

func ResourceScalewayInstanceIPReverseDNSCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	instanceAPI, zone, err := InstanceAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := instanceAPI.GetIP(&instance.GetIPRequest{
		IP:   locality.ExpandID(d.Get("ip_id")),
		Zone: zone,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(locality.NewZonedIDString(zone, res.IP.ID))

	if _, ok := d.GetOk("reverse"); ok {
		tflog.Debug(ctx, fmt.Sprintf("updating IP %q reverse to %q\n", d.Id(), d.Get("reverse")))

		updateReverseReq := &instance.UpdateIPRequest{
			Zone: zone,
			IP:   res.IP.ID,
		}

		if reverse, ok := d.GetOk("reverse"); ok {
			updateReverseReq.Reverse = &instance.NullableStringValue{Value: reverse.(string)}
		} else {
			updateReverseReq.Reverse = &instance.NullableStringValue{Null: true}
		}

		err := retryUpdateReverseDNS(ctx, instanceAPI, updateReverseReq, d.Timeout(schema.TimeoutCreate))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return ResourceScalewayInstanceIPReverseDNSRead(ctx, d, meta)
}

func ResourceScalewayInstanceIPReverseDNSRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	instanceAPI, zone, ID, err := InstanceAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := instanceAPI.GetIP(&instance.GetIPRequest{
		IP:   ID,
		Zone: zone,
	}, scw.WithContext(ctx))
	if err != nil {
		// We check for 403 because instance API returns 403 for a deleted IP
		if http_errors.Is404Error(err) || http_errors.Is403Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("zone", string(zone))
	_ = d.Set("reverse", res.IP.Reverse)
	return nil
}

func ResourceScalewayInstanceIPReverseDNSUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	instanceAPI, zone, ID, err := InstanceAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("reverse") {
		tflog.Debug(ctx, fmt.Sprintf("updating IP %q reverse to %q\n", d.Id(), d.Get("reverse")))

		updateReverseReq := &instance.UpdateIPRequest{
			Zone: zone,
			IP:   ID,
		}

		if reverse, ok := d.GetOk("reverse"); ok {
			updateReverseReq.Reverse = &instance.NullableStringValue{Value: reverse.(string)}
		} else {
			updateReverseReq.Reverse = &instance.NullableStringValue{Null: true}
		}
		err := retryUpdateReverseDNS(ctx, instanceAPI, updateReverseReq, d.Timeout(schema.TimeoutUpdate))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return ResourceScalewayInstanceIPReverseDNSRead(ctx, d, meta)
}

func ResourceScalewayInstanceIPReverseDNSDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	instanceAPI, zone, ID, err := InstanceAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Unset the reverse dns on the IP
	updateReverseReq := &instance.UpdateIPRequest{
		Zone:    zone,
		IP:      ID,
		Reverse: &instance.NullableStringValue{Null: true},
	}
	_, err = instanceAPI.UpdateIP(updateReverseReq, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
