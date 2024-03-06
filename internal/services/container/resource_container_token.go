package container

import (
	"context"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/difffuncs"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/verify"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	container "github.com/scaleway/scaleway-sdk-go/api/container/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceScalewayContainerToken() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayContainerTokenCreate,
		ReadContext:   ResourceScalewayContainerTokenRead,
		DeleteContext: ResourceScalewayContainerTokenDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"container_id": {
				Type:             schema.TypeString,
				ForceNew:         true,
				Optional:         true,
				ExactlyOneOf:     []string{"namespace_id"},
				DiffSuppressFunc: difffuncs.DiffSuppressFuncLocality,
			},
			"namespace_id": {
				Type:             schema.TypeString,
				ForceNew:         true,
				Optional:         true,
				ExactlyOneOf:     []string{"container_id"},
				DiffSuppressFunc: difffuncs.DiffSuppressFuncLocality,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"expires_at": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				ValidateDiagFunc: verify.ValidateDate(),
				DiffSuppressFunc: difffuncs.DiffSuppressFuncTimeRFC3339,
			},
			"token": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},

			"region": locality.RegionalSchema(),
		},
		CustomizeDiff: difffuncs.CustomizeDiffLocalityCheck("container_id", "namespace_id"),
	}
}

func ResourceScalewayContainerTokenCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := containerAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	token, err := api.CreateToken(&container.CreateTokenRequest{
		Region:      region,
		ContainerID: types.ExpandStringPtr(locality.ExpandID(d.Get("container_id"))),
		NamespaceID: types.ExpandStringPtr(locality.ExpandID(d.Get("namespace_id"))),
		Description: types.ExpandStringPtr(d.Get("description")),
		ExpiresAt:   types.ExpandTimePtr(d.Get("expires_at")),
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(locality.NewRegionalIDString(region, token.ID))

	_ = d.Set("token", token.Token)

	return ResourceScalewayContainerTokenRead(ctx, d, meta)
}

func ResourceScalewayContainerTokenRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, ID, err := ContainerAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	token, err := api.GetToken(&container.GetTokenRequest{
		Region:  region,
		TokenID: ID,
	}, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("container_id", types.FlattenStringPtr(token.ContainerID))
	_ = d.Set("namespace_id", types.FlattenStringPtr(token.NamespaceID))
	_ = d.Set("description", types.FlattenStringPtr(token.Description))
	_ = d.Set("expires_at", types.FlattenTime(token.ExpiresAt))

	return nil
}

func ResourceScalewayContainerTokenDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, ID, err := ContainerAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = api.DeleteToken(&container.DeleteTokenRequest{
		Region:  region,
		TokenID: ID,
	}, scw.WithContext(ctx))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}
