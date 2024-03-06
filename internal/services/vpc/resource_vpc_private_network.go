package vpc

import (
	"context"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/verify"

	meta2 "github.com/scaleway/terraform-provider-scaleway/v2/internal/meta"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/project"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/organization"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/scaleway/scaleway-sdk-go/api/vpc/v2"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceScalewayVPCPrivateNetwork() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayVPCPrivateNetworkCreate,
		ReadContext:   ResourceScalewayVPCPrivateNetworkRead,
		UpdateContext: ResourceScalewayVPCPrivateNetworkUpdate,
		DeleteContext: ResourceScalewayVPCPrivateNetworkDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{Version: 0, Type: vpcPrivateNetworkUpgradeV1SchemaType(), Upgrade: vpcPrivateNetworkV1SUpgradeFunc},
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of the private network",
				Computed:    true,
			},
			"ipv4_subnet": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "The IPv4 subnet associated with the private network",
				Computed:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"subnet": {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							ForceNew:     true,
							Description:  "The subnet CIDR",
							ValidateFunc: validation.IsCIDRNetwork(0, 32),
						},
						// computed
						"id": {
							Type:        schema.TypeString,
							Description: "The subnet ID",
							Computed:    true,
						},
						"created_at": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The date and time of the creation of the subnet",
						},
						"updated_at": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The date and time of the last update of the subnet",
						},
						"address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The network address of the subnet in dotted decimal notation, e.g., '192.168.0.0' for a '192.168.0.0/24' subnet",
						},
						"subnet_mask": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The subnet mask expressed in dotted decimal notation, e.g., '255.255.255.0' for a /24 subnet",
						},
						"prefix_length": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The length of the network prefix, e.g., 24 for a 255.255.255.0 mask",
						},
					},
				},
			},
			"ipv6_subnets": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The IPv6 subnet associated with the private network",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"subnet": {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							ForceNew:     true,
							Description:  "The subnet CIDR",
							ValidateFunc: validation.IsCIDRNetwork(0, 128),
						},
						// computed
						"id": {
							Type:        schema.TypeString,
							Description: "The subnet ID",
							Computed:    true,
						},
						"created_at": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The date and time of the creation of the subnet",
						},
						"updated_at": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The date and time of the last update of the subnet",
						},
						"address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The network address of the subnet in dotted decimal notation, e.g., '192.168.0.0' for a '192.168.0.0/24' subnet",
						},
						"subnet_mask": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The subnet mask expressed in dotted decimal notation, e.g., '255.255.255.0' for a /24 subnet",
						},
						"prefix_length": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The length of the network prefix, e.g., 24 for a 255.255.255.0 mask",
						},
					},
				},
			},
			"tags": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "The tags associated with private network",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"is_regional": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Deprecated:  "This field is deprecated and will be removed in the next major version",
				Description: "Defines whether the private network is Regional. By default, it will be Zonal",
			},
			"vpc_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "The VPC in which to create the private network",
			},
			"project_id": project.ProjectIDSchema(),
			"zone": {
				Type:             schema.TypeString,
				Description:      "The zone you want to attach the resource to",
				Optional:         true,
				Computed:         true,
				Deprecated:       "This field is deprecated and will be removed in the next major version, please use `region` instead",
				ValidateDiagFunc: verify.StringInSliceWithWarning(locality.AllZones(), "zone"),
			},
			"region": locality.RegionalSchema(),
			// Computed elements
			"organization_id": organization.OrganizationIDSchema(),
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the creation of the private network",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the last update of the private network",
			},
		},
	}
}

func ResourceScalewayVPCPrivateNetworkCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vpcAPI, region, err := vpcAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	ipv4Subnets, ipv6Subnets, err := expandSubnets(d)
	if err != nil {
		return diag.FromErr(err)
	}

	req := &vpc.CreatePrivateNetworkRequest{
		Name:      types.ExpandOrGenerateString(d.Get("name"), "pn"),
		Tags:      types.ExpandStrings(d.Get("tags")),
		ProjectID: d.Get("project_id").(string),
		Region:    region,
	}

	if _, ok := d.GetOk("vpc_id"); ok {
		vpcID := locality.ExpandRegionalID(d.Get("vpc_id").(string)).ID
		req.VpcID = types.ExpandUpdatedStringPtr(vpcID)
	}

	if ipv4Subnets != nil {
		req.Subnets = append(req.Subnets, ipv4Subnets...)
	}

	if ipv6Subnets != nil {
		req.Subnets = append(req.Subnets, ipv6Subnets...)
	}

	pn, err := vpcAPI.CreatePrivateNetwork(req, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(locality.NewRegionalIDString(region, pn.ID))

	return ResourceScalewayVPCPrivateNetworkRead(ctx, d, meta)
}

func ResourceScalewayVPCPrivateNetworkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vpcAPI, region, ID, err := vpcAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	pn, err := vpcAPI.GetPrivateNetwork(&vpc.GetPrivateNetworkRequest{
		PrivateNetworkID: ID,
		Region:           region,
	}, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	zone, err := locality.ExtractZone(d, meta.(*meta2.Meta))
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set("name", pn.Name)
	_ = d.Set("vpc_id", locality.NewRegionalIDString(region, pn.VpcID))
	_ = d.Set("organization_id", pn.OrganizationID)
	_ = d.Set("project_id", pn.ProjectID)
	_ = d.Set("created_at", types.FlattenTime(pn.CreatedAt))
	_ = d.Set("updated_at", types.FlattenTime(pn.UpdatedAt))
	_ = d.Set("tags", pn.Tags)
	_ = d.Set("region", region)
	_ = d.Set("is_regional", true)
	_ = d.Set("zone", zone)

	ipv4Subnet, ipv6Subnets := FlattenAndSortSubnets(pn.Subnets)
	_ = d.Set("ipv4_subnet", ipv4Subnet)
	_ = d.Set("ipv6_subnets", ipv6Subnets)

	return nil
}

func ResourceScalewayVPCPrivateNetworkUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vpcAPI, region, ID, err := vpcAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = vpcAPI.UpdatePrivateNetwork(&vpc.UpdatePrivateNetworkRequest{
		PrivateNetworkID: ID,
		Region:           region,
		Name:             scw.StringPtr(d.Get("name").(string)),
		Tags:             types.ExpandUpdatedStringsPtr(d.Get("tags")),
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayVPCPrivateNetworkRead(ctx, d, meta)
}

func ResourceScalewayVPCPrivateNetworkDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vpcAPI, region, ID, err := vpcAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = retry.RetryContext(ctx, defaultVPCPrivateNetworkRetryInterval, func() *retry.RetryError {
		err := vpcAPI.DeletePrivateNetwork(&vpc.DeletePrivateNetworkRequest{
			PrivateNetworkID: ID,
			Region:           region,
		}, scw.WithContext(ctx))
		if err != nil {
			if http_errors.Is412Error(err) {
				return retry.RetryableError(err)
			} else if !http_errors.Is404Error(err) {
				return retry.NonRetryableError(err)
			}
		}

		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
