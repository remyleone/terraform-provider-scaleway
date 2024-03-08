package iam

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	iam "github.com/scaleway/scaleway-sdk-go/api/iam/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/organization"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/project"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
	"strings"
)

func ResourceScalewayIamSSKKey() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayIamSSKKeyCreate,
		ReadContext:   ResourceScalewayIamSSHKeyRead,
		UpdateContext: ResourceScalewayIamSSKKeyUpdate,
		DeleteContext: ResourceScalewayIamSSKKeyDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "The name of the iam SSH key",
			},
			"public_key": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The public SSH key",
				// We don't consider trailing \n as diff
				DiffSuppressFunc: func(_, oldValue, newValue string, _ *schema.ResourceData) bool {
					return strings.Trim(oldValue, "\n") == strings.Trim(newValue, "\n")
				},
			},
			"fingerprint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The fingerprint of the iam SSH key",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the creation of the iam SSH Key",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the last update of the iam SSH Key",
			},
			"organization_id": organization.OrganizationIDSchema(),
			"project_id":      project.ProjectIDSchema(),
			"disabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "The SSH key status",
			},
		},
	}
}

func ResourceScalewayIamSSKKeyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	iamAPI := IAMAPI(meta)

	res, err := iamAPI.CreateSSHKey(&iam.CreateSSHKeyRequest{
		Name:      d.Get("name").(string),
		PublicKey: strings.Trim(d.Get("public_key").(string), "\n"),
		ProjectID: (d.Get("project_id")).(string),
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	if _, disabledExists := d.GetOk("disabled"); disabledExists {
		_, err = iamAPI.UpdateSSHKey(&iam.UpdateSSHKeyRequest{
			SSHKeyID: d.Id(),
			Disabled: types.ExpandBoolPtr(types.GetBool(d, "disabled")),
		}, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(res.ID)

	return ResourceScalewayIamSSHKeyRead(ctx, d, meta)
}

func ResourceScalewayIamSSHKeyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	iamAPI := IAMAPI(meta)

	res, err := iamAPI.GetSSHKey(&iam.GetSSHKeyRequest{
		SSHKeyID: d.Id(),
	}, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", res.Name)
	_ = d.Set("public_key", res.PublicKey)
	_ = d.Set("fingerprint", res.Fingerprint)
	_ = d.Set("created_at", types.FlattenTime(res.CreatedAt))
	_ = d.Set("updated_at", types.FlattenTime(res.UpdatedAt))
	_ = d.Set("organization_id", res.OrganizationID)
	_ = d.Set("project_id", res.ProjectID)
	_ = d.Set("disabled", res.Disabled)

	return nil
}

func ResourceScalewayIamSSKKeyUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	iamAPI := IAMAPI(meta)

	req := &iam.UpdateSSHKeyRequest{
		SSHKeyID: d.Id(),
	}

	hasUpdated := false

	if d.HasChange("name") {
		req.Name = types.ExpandStringPtr(d.Get("name"))
		hasUpdated = true
	}

	if d.HasChange("disabled") {
		if _, disabledExists := d.GetOk("disabled"); !disabledExists {
			_, err := iamAPI.UpdateSSHKey(&iam.UpdateSSHKeyRequest{
				SSHKeyID: d.Id(),
				Disabled: types.ExpandBoolPtr(false),
			})
			if err != nil {
				return diag.FromErr(err)
			}
		} else {
			_, err := iamAPI.UpdateSSHKey(&iam.UpdateSSHKeyRequest{
				SSHKeyID: d.Id(),
				Disabled: types.ExpandBoolPtr(types.GetBool(d, "disabled")),
			})
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if hasUpdated {
		_, err := iamAPI.UpdateSSHKey(req, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return ResourceScalewayIamSSHKeyRead(ctx, d, meta)
}

func ResourceScalewayIamSSKKeyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	iamAPI := IAMAPI(meta)

	err := iamAPI.DeleteSSHKey(&iam.DeleteSSHKeyRequest{
		SSHKeyID: d.Id(),
	}, scw.WithContext(ctx))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
