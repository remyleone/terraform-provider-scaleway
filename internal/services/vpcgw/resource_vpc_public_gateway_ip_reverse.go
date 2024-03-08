package vpcgw

import (
	"context"
	"fmt"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/api/vpcgw/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceScalewayVPCPublicGatewayIPReverseDNS() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayVPCPublicGatewayIPReverseDNSCreate,
		ReadContext:   ResourceScalewayVPCPublicGatewayIPReverseDNSRead,
		UpdateContext: ResourceScalewayVPCPublicGatewayIPReverseDNSUpdate,
		DeleteContext: ResourceScalewayVPCPublicGatewayIPReverseDNSDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Default: schema.DefaultTimeout(defaultVPCPublicGatewayIPReverseDNSTimeout),
			Create:  schema.DefaultTimeout(defaultVPCPublicGatewayIPReverseDNSTimeout),
			Update:  schema.DefaultTimeout(defaultVPCPublicGatewayIPReverseDNSTimeout),
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"gateway_ip_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The IP ID",
			},
			"reverse": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The reverse DNS for this IP",
			},
			"zone": locality.ZonalSchema(),
		},
	}
}

func ResourceScalewayVPCPublicGatewayIPReverseDNSCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vpcgwAPI, zone, err := VpcgwAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := vpcgwAPI.GetIP(&vpcgw.GetIPRequest{
		Zone: zone,
		IPID: locality.ExpandID(d.Get("gateway_ip_id")),
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(locality.NewZonedIDString(zone, res.ID))

	if _, ok := d.GetOk("reverse"); ok {
		tflog.Debug(ctx, fmt.Sprintf("updating IP %q reverse to %q\n", d.Id(), d.Get("reverse")))

		updateReverseReq := &vpcgw.UpdateIPRequest{
			Zone: zone,
			IPID: res.ID,
		}

		reverse := d.Get("reverse").(string)
		if len(reverse) > 0 {
			updateReverseReq.Reverse = types.ExpandStringPtr(reverse)
		}

		err := retryUpdateGatewayReverseDNS(ctx, vpcgwAPI, updateReverseReq, d.Timeout(schema.TimeoutCreate))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return ResourceScalewayVPCPublicGatewayIPReverseDNSRead(ctx, d, meta)
}

func ResourceScalewayVPCPublicGatewayIPReverseDNSRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vpcgwAPI, zone, ID, err := VpcgwAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := vpcgwAPI.GetIP(&vpcgw.GetIPRequest{
		IPID: ID,
		Zone: zone,
	}, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("zone", string(zone))
	_ = d.Set("reverse", res.Reverse)

	return nil
}

func ResourceScalewayVPCPublicGatewayIPReverseDNSUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vpcgwAPI, zone, ID, err := VpcgwAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("reverse") {
		tflog.Debug(ctx, fmt.Sprintf("updating IP %q reverse to %q\n", d.Id(), d.Get("reverse")))

		updateReverseReq := &vpcgw.UpdateIPRequest{
			Zone: zone,
			IPID: ID,
		}

		reverse := d.Get("reverse").(string)
		if len(reverse) > 0 {
			updateReverseReq.Reverse = types.ExpandStringPtr(reverse)
		}

		err := retryUpdateGatewayReverseDNS(ctx, vpcgwAPI, updateReverseReq, d.Timeout(schema.TimeoutUpdate))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return ResourceScalewayVPCPublicGatewayIPReverseDNSRead(ctx, d, meta)
}

func ResourceScalewayVPCPublicGatewayIPReverseDNSDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vpcgwAPI, zone, ID, err := VpcgwAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Unset the reverse dns on the IP
	updateReverseReq := &vpcgw.UpdateIPRequest{
		Zone:    zone,
		IPID:    ID,
		Reverse: new(string),
	}
	_, err = vpcgwAPI.UpdateIP(updateReverseReq, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
