package scaleway

import (
	"context"
	"fmt"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/scaleway/errors"
	"net"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/project"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/scaleway/scaleway-sdk-go/api/ipam/v1"
	"github.com/scaleway/scaleway-sdk-go/api/vpc/v2"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func resourceScalewayIPAMIP() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScalewayIPAMIPCreate,
		ReadContext:   resourceScalewayIPAMIPRead,
		UpdateContext: resourceScalewayIPAMIPUpdate,
		DeleteContext: resourceScalewayIPAMIPDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"address": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ForceNew:         true,
				Description:      "Request a specific IP in the requested source pool",
				ValidateFunc:     validation.IsIPAddress,
				DiffSuppressFunc: diffSuppressFuncStandaloneIPandCIDR,
			},
			"source": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "The source in which to book the IP",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"zonal": {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "Zone the IP lives in if the IP is a public zoned one",
						},
						"private_network_id": {
							Type:             schema.TypeString,
							Optional:         true,
							Computed:         true,
							Description:      "Private Network the IP lives in if the IP is a private IP",
							DiffSuppressFunc: diffSuppressFuncLocality,
						},
						"subnet_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "Private Network subnet the IP lives in if the IP is a private IP in a Private Network",
						},
					},
				},
			},
			"is_ipv6": {
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Default:     false,
				Description: "Request an IPv6 instead of an IPv4",
			},
			"tags": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "The tags associated with the IP",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"project_id": project.ProjectIDSchema(),
			"region":     regionSchema(),
			// Computed elements
			"resource": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The IP resource",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Type of resource the IP is attached to",
						},
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "ID of the resource the IP is attached to",
						},
						"mac_address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "MAC of the resource the IP is attached to",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Name of the resource the IP is attached to",
						},
					},
				},
			},
			"reverses": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The reverses DNS for this IP",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"hostname": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The reverse domain name",
						},
						"address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The IP corresponding to the hostname",
						},
					},
				},
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the creation of the IP",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the last update of the IP",
			},
			"zone": zoneComputedSchema(),
		},
	}
}

func resourceScalewayIPAMIPCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ipamAPI, region, err := ipamAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	req := &ipam.BookIPRequest{
		Region:    region,
		ProjectID: d.Get("project_id").(string),
		IsIPv6:    d.Get("is_ipv6").(bool),
		Tags:      expandStrings(d.Get("tags")),
	}

	address, addressOk := d.GetOk("address")
	if addressOk {
		addressStr := address.(string)
		parsedIP, _, err := net.ParseCIDR(addressStr)
		if err != nil {
			parsedIP = net.ParseIP(addressStr)
			if parsedIP == nil {
				return diag.FromErr(fmt.Errorf("error parsing IP address: %s", err))
			}
		}
		req.Address = scw.IPPtr(parsedIP)
	}

	if source, ok := d.GetOk("source"); ok {
		req.Source = expandIPSource(source)
	}

	res, err := ipamAPI.BookIP(req, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(newRegionalIDString(region, res.ID))

	return resourceScalewayIPAMIPRead(ctx, d, meta)
}

func resourceScalewayIPAMIPRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ipamAPI, region, ID, err := ipamAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	vpcAPI, err := vpcAPI(meta)
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

	privateNetworkID := ""
	if source, ok := d.GetOk("source"); ok {
		sourceData := expandIPSource(source)
		if sourceData.PrivateNetworkID != nil {
			pn, err := vpcAPI.GetPrivateNetwork(&vpc.GetPrivateNetworkRequest{
				PrivateNetworkID: *sourceData.PrivateNetworkID,
				Region:           region,
			}, scw.WithContext(ctx))
			if err != nil {
				return diag.FromErr(err)
			}

			ipv4Subnets, ipv6Subnets := flattenAndSortSubnets(pn.Subnets)
			var found bool

			if d.Get("is_ipv6").(bool) {
				found = checkSubnetIDInFlattenedSubnets(*res.Source.SubnetID, ipv6Subnets)
			} else {
				found = checkSubnetIDInFlattenedSubnets(*res.Source.SubnetID, ipv4Subnets)
			}

			if found {
				privateNetworkID = pn.ID
			}
		}
	}

	address, err := flattenIPNet(res.Address)
	if err != nil {
		return diag.FromErr(err)
	}
	_ = d.Set("address", address)
	_ = d.Set("source", flattenIPSource(res.Source, privateNetworkID))
	_ = d.Set("resource", flattenIPResource(res.Resource))
	_ = d.Set("project_id", res.ProjectID)
	_ = d.Set("created_at", flattenTime(res.CreatedAt))
	_ = d.Set("updated_at", flattenTime(res.UpdatedAt))
	_ = d.Set("is_ipv6", res.IsIPv6)
	_ = d.Set("region", region)
	if res.Zone != nil {
		_ = d.Set("zone", res.Zone.String())
	}
	if len(res.Tags) > 0 {
		_ = d.Set("tags", res.Tags)
	}
	_ = d.Set("reverses", flattenIPReverses(res.Reverses))

	return nil
}

func resourceScalewayIPAMIPUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ipamAPI, region, ID, err := ipamAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = ipamAPI.UpdateIP(&ipam.UpdateIPRequest{
		IPID:   ID,
		Region: region,
		Tags:   expandUpdatedStringsPtr(d.Get("tags")),
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceScalewayIPAMIPRead(ctx, d, meta)
}

func resourceScalewayIPAMIPDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ipamAPI, region, ID, err := ipamAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = ipamAPI.ReleaseIP(&ipam.ReleaseIPRequest{
		Region: region,
		IPID:   ID,
	}, scw.WithContext(ctx))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
