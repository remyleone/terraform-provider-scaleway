package scaleway

import (
	"context"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/scaleway/errors"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/types"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/verify"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/project"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	iam "github.com/scaleway/scaleway-sdk-go/api/iam/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func resourceScalewayIamAPIKey() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScalewayIamAPIKeyCreate,
		ReadContext:   resourceScalewayIamAPIKeyRead,
		UpdateContext: resourceScalewayIamAPIKeyUpdate,
		DeleteContext: resourceScalewayIamAPIKeyDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the iam api key",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the creation of the iam api key",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the last update of the iam api key",
			},
			"expires_at": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Description:      "The date and time of the expiration of the iam api key. Cannot be changed afterwards",
				ValidateDiagFunc: validateDate(),
			},
			"access_key": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The access key of the iam api key",
			},
			"secret_key": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The secret Key of the iam api key",
				Sensitive:   true,
			},
			"application_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Description:   "ID of the application attached to the api key",
				ConflictsWith: []string{"user_id"},
				ValidateFunc:  verify.UUID(),
			},
			"user_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "ID of the user attached to the api key",
				ConflictsWith: []string{"application_id"},
				ValidateFunc:  verify.UUID(),
			},
			"editable": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether or not the iam api key is editable",
			},
			"creation_ip": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IPv4 Address of the device which created the API key",
			},
			"default_project_id": project.ProjectIDSchema(),
		},
	}
}

func resourceScalewayIamAPIKeyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	iamAPI := iamAPI(meta)
	res, err := iamAPI.CreateAPIKey(&iam.CreateAPIKeyRequest{
		ApplicationID:    types.ExpandStringPtr(d.Get("application_id")),
		UserID:           types.ExpandStringPtr(d.Get("user_id")),
		ExpiresAt:        expandTimePtr(d.Get("expires_at")),
		DefaultProjectID: types.ExpandStringPtr(d.Get("default_project_id")),
		Description:      d.Get("description").(string),
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set("secret_key", res.SecretKey)

	d.SetId(res.AccessKey)

	return resourceScalewayIamAPIKeyRead(ctx, d, meta)
}

func resourceScalewayIamAPIKeyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := iamAPI(meta)
	res, err := api.GetAPIKey(&iam.GetAPIKeyRequest{
		AccessKey: d.Id(),
	}, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	_ = d.Set("description", res.Description)
	_ = d.Set("created_at", flattenTime(res.CreatedAt))
	_ = d.Set("updated_at", flattenTime(res.UpdatedAt))
	_ = d.Set("expires_at", flattenTime(res.ExpiresAt))
	_ = d.Set("access_key", res.AccessKey)

	if res.ApplicationID != nil {
		_ = d.Set("application_id", res.ApplicationID)
	}
	if res.UserID != nil {
		_ = d.Set("user_id", res.UserID)
	}

	_ = d.Set("editable", res.Editable)
	_ = d.Set("creation_ip", res.CreationIP)
	_ = d.Set("default_project_id", res.DefaultProjectID)

	return nil
}

func resourceScalewayIamAPIKeyUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := iamAPI(meta)

	req := &iam.UpdateAPIKeyRequest{
		AccessKey: d.Id(),
	}

	hasChanged := false

	if d.HasChange("description") {
		req.Description = expandUpdatedStringPtr(d.Get("description"))
		hasChanged = true
	}

	if d.HasChange("default_project_id") {
		req.DefaultProjectID = types.ExpandStringPtr(d.Get("default_project_id"))
		hasChanged = true
	}

	if hasChanged {
		_, err := api.UpdateAPIKey(req, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceScalewayIamAPIKeyRead(ctx, d, meta)
}

func resourceScalewayIamAPIKeyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := iamAPI(meta)

	err := api.DeleteAPIKey(&iam.DeleteAPIKeyRequest{
		AccessKey: d.Id(),
	}, scw.WithContext(ctx))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
