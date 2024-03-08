package iot

import (
	"context"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/difffuncs"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"time"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	iot "github.com/scaleway/scaleway-sdk-go/api/iot/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceScalewayIotNetwork() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayIotNetworkCreate,
		ReadContext:   ResourceScalewayIotNetworkRead,
		DeleteContext: ResourceScalewayIotNetworkDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Delete:  schema.DefaultTimeout(defaultIoTHubTimeout),
			Default: schema.DefaultTimeout(defaultIoTHubTimeout),
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"hub_id": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "The ID of the hub on which this network will be created",
				DiffSuppressFunc: difffuncs.DiffSuppressFuncLocality,
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the network",
			},
			"type": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The type of the network",
				ValidateFunc: validation.StringInSlice([]string{
					iot.NetworkNetworkTypeSigfox.String(),
					iot.NetworkNetworkTypeRest.String(),
				}, false),
			},
			"topic_prefix": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "The prefix that will be prepended to all topics for this Network",
			},
			// Computed elements
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the creation of the network",
			},
			"endpoint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The endpoint to use when interacting with the network",
			},
			"secret": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The endpoint key to keep secret",
				Sensitive:   true,
			},
		},
	}
}

func ResourceScalewayIotNetworkCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	iotAPI, region, err := NewAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	req := &iot.CreateNetworkRequest{
		Region: region,
		Name:   types.ExpandOrGenerateString(d.Get("name"), "network"),
		Type:   iot.NetworkNetworkType(d.Get("type").(string)),
		HubID:  locality.ExpandID(d.Get("hub_id")),
	}

	if topicPrefix, ok := d.GetOk("topic_prefix"); ok {
		req.TopicPrefix = topicPrefix.(string)
	}

	res, err := iotAPI.CreateNetwork(req, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(locality.NewRegionalIDString(region, res.Network.ID))

	// Secret key cannot be retrieved later
	_ = d.Set("secret", res.Secret)

	return ResourceScalewayIotNetworkRead(ctx, d, meta)
}

func ResourceScalewayIotNetworkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	iotAPI, region, networkID, err := NewAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	network, err := iotAPI.GetNetwork(&iot.GetNetworkRequest{
		Region:    region,
		NetworkID: networkID,
	}, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", network.Name)
	_ = d.Set("type", network.Type.String())
	_ = d.Set("endpoint", network.Endpoint)
	_ = d.Set("hub_id", locality.NewRegionalID(region, network.HubID).String())
	_ = d.Set("created_at", network.CreatedAt.Format(time.RFC3339))
	_ = d.Set("topic_prefix", network.TopicPrefix)

	return nil
}

func ResourceScalewayIotNetworkDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	iotAPI, region, networkID, err := NewAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	hubID := locality.ExpandZonedID(d.Get("hub_id").(string)).ID

	err = iotAPI.DeleteNetwork(&iot.DeleteNetworkRequest{
		Region:    region,
		NetworkID: networkID,
	}, scw.WithContext(ctx))
	if err != nil {
		if !http_errors.Is404Error(err) {
			return diag.FromErr(err)
		}
	}

	_, err = waitIotHub(ctx, iotAPI, region, hubID, d.Timeout(schema.TimeoutDelete))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
