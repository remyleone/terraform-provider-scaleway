package ipam

import (
	"context"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
	"net"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/scaleway/scaleway-sdk-go/api/ipam/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceScalewayIPAMIPReverseDNS() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayIPAMIPReverseDNSCreate,
		ReadContext:   ResourceScalewayIPAMIPReverseDNSRead,
		UpdateContext: ResourceScalewayIPAMIPReverseDNSUpdate,
		DeleteContext: ResourceScalewayIPAMIPReverseDNSDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Default: schema.DefaultTimeout(defaultIPAMIPReverseDNSTimeout),
			Create:  schema.DefaultTimeout(defaultIPAMIPReverseDNSTimeout),
			Update:  schema.DefaultTimeout(defaultIPAMIPReverseDNSTimeout),
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"ipam_ip_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The IPAM IP ID",
			},
			"hostname": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The reverse domain name",
			},
			"address": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The IP corresponding to the hostname",
				ValidateFunc: validation.IsIPAddress,
			},
			"region": locality.RegionalSchema(),
		},
	}
}

func ResourceScalewayIPAMIPReverseDNSCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ipamAPI, region, err := ipamAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := ipamAPI.GetIP(&ipam.GetIPRequest{
		Region: region,
		IPID:   locality.ExpandID(d.Get("ipam_ip_id")),
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(locality.NewRegionalIDString(region, res.ID))
	if hostname, ok := d.GetOk("hostname"); ok {
		reverse := &ipam.Reverse{
			Hostname: hostname.(string),
			Address:  scw.IPPtr(net.ParseIP(d.Get("address").(string))),
		}

		updateReverseReq := &ipam.UpdateIPRequest{
			Region:   region,
			IPID:     res.ID,
			Reverses: []*ipam.Reverse{reverse},
		}

		_, err := ipamAPI.UpdateIP(updateReverseReq, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return ResourceScalewayIPAMIPReverseDNSRead(ctx, d, meta)
}

func ResourceScalewayIPAMIPReverseDNSRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ipamAPI, region, ID, err := ipamAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := ipamAPI.GetIP(&ipam.GetIPRequest{
		Region: region,
		IPID:   ID,
	}, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	managedHostname := d.Get("hostname").(string)
	managedAddress := d.Get("address").(string)
	for _, reverse := range res.Reverses {
		if reverse.Hostname == managedHostname && reverse.Address.String() == managedAddress {
			_ = d.Set("hostname", reverse.Hostname)
			_ = d.Set("address", types.FlattenIPPtr(reverse.Address))
			break
		}
	}

	_ = d.Set("region", region)

	return nil
}

func ResourceScalewayIPAMIPReverseDNSUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ipamAPI, region, ID, err := ipamAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChanges("hostname", "address") {
		reverse := &ipam.Reverse{
			Hostname: d.Get("hostname").(string),
			Address:  scw.IPPtr(net.ParseIP(d.Get("address").(string))),
		}

		updateReverseReq := &ipam.UpdateIPRequest{
			Region:   region,
			IPID:     ID,
			Reverses: []*ipam.Reverse{reverse},
		}

		_, err := ipamAPI.UpdateIP(updateReverseReq, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return ResourceScalewayIPAMIPReverseDNSRead(ctx, d, meta)
}

func ResourceScalewayIPAMIPReverseDNSDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ipamAPI, region, ID, err := ipamAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	updateReverseReq := &ipam.UpdateIPRequest{
		Region:   region,
		IPID:     ID,
		Reverses: []*ipam.Reverse{},
	}

	_, err = ipamAPI.UpdateIP(updateReverseReq, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
