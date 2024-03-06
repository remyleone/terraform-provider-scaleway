package account

import (
	"context"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/verify"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	accountV3 "github.com/scaleway/scaleway-sdk-go/api/account/v3"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceScalewayAccountProject() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayAccountProjectCreate,
		ReadContext:   ResourceScalewayAccountProjectRead,
		UpdateContext: ResourceScalewayAccountProjectUpdate,
		DeleteContext: ResourceScalewayAccountProjectDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "The name of the project",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the project",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the creation of the Project (Format ISO 8601)",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the last update of the Project (Format ISO 8601)",
			},
			"organization_id": {
				Type:         schema.TypeString,
				Description:  "The organization_id you want to attach the resource to",
				Optional:     true,
				ForceNew:     true,
				Computed:     true,
				ValidateFunc: verify.UUID(),
			},
		},
	}
}

func ResourceScalewayAccountProjectCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	accountAPI := accountV3ProjectAPI(meta)

	request := &accountV3.ProjectAPICreateProjectRequest{
		Name:        types.ExpandOrGenerateString(d.Get("name"), "project"),
		Description: d.Get("description").(string),
	}

	if organisationIDRaw, exist := d.GetOk("organization_id"); exist {
		request.OrganizationID = organisationIDRaw.(string)
	}

	res, err := accountAPI.CreateProject(request, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(res.ID)

	return ResourceScalewayAccountProjectRead(ctx, d, meta)
}

func ResourceScalewayAccountProjectRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	accountAPI := accountV3ProjectAPI(meta)
	res, err := accountAPI.GetProject(&accountV3.ProjectAPIGetProjectRequest{
		ProjectID: d.Id(),
	}, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", res.Name)
	_ = d.Set("description", res.Description)
	_ = d.Set("created_at", types.FlattenTime(res.CreatedAt))
	_ = d.Set("updated_at", types.FlattenTime(res.UpdatedAt))
	_ = d.Set("organization_id", res.OrganizationID)

	return nil
}

func ResourceScalewayAccountProjectUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	accountAPI := accountV3ProjectAPI(meta)

	req := &accountV3.ProjectAPIUpdateProjectRequest{
		ProjectID: d.Id(),
	}

	hasChanged := false

	if d.HasChange("name") {
		req.Name = types.ExpandUpdatedStringPtr(d.Get("name"))
		hasChanged = true
	}
	if d.HasChange("description") {
		req.Description = types.ExpandUpdatedStringPtr(d.Get("description"))
		hasChanged = true
	}

	if hasChanged {
		_, err := accountAPI.UpdateProject(req, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return ResourceScalewayAccountProjectRead(ctx, d, meta)
}

func ResourceScalewayAccountProjectDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	accountAPI := accountV3ProjectAPI(meta)

	err := accountAPI.DeleteProject(&accountV3.ProjectAPIDeleteProjectRequest{
		ProjectID: d.Id(),
	}, scw.WithContext(ctx))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
