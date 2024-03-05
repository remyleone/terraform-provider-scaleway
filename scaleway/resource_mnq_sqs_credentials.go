package scaleway

import (
	"context"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/scaleway/errors"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/project"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/types"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	mnq "github.com/scaleway/scaleway-sdk-go/api/mnq/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func resourceScalewayMNQSQSCredentials() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScalewayMNQSQSCredentialsCreate,
		ReadContext:   resourceScalewayMNQSQSCredentialsRead,
		UpdateContext: resourceScalewayMNQSQSCredentialsUpdate,
		DeleteContext: resourceScalewayMNQSQSCredentialsDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "The credentials name",
			},
			"permissions": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"can_publish": {
							Type:        schema.TypeBool,
							Computed:    true,
							Optional:    true,
							Description: "Allow publish messages to the service",
						},
						"can_receive": {
							Type:        schema.TypeBool,
							Computed:    true,
							Optional:    true,
							Description: "Allow receive messages from the service",
						},
						"can_manage": {
							Type:        schema.TypeBool,
							Computed:    true,
							Optional:    true,
							Description: "Allow manage the associated resource",
						},
					},
				},
			},
			"region":     regionSchema(),
			"project_id": project.ProjectIDSchema(),

			// Computed
			"access_key": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SQS credentials access key",
				Sensitive:   true,
			},
			"secret_key": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SQS credentials secret key",
				Sensitive:   true,
			},
		},
	}
}

func resourceScalewayMNQSQSCredentialsCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := newMNQSQSAPI(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	credentials, err := api.CreateSqsCredentials(&mnq.SqsAPICreateSqsCredentialsRequest{
		Region:    region,
		ProjectID: d.Get("project_id").(string),
		Name:      types.ExpandOrGenerateString(d.Get("name").(string), "sqs-credentials"),
		Permissions: &mnq.SqsPermissions{
			CanPublish: expandBoolPtr(d.Get("permissions.0.can_publish")),
			CanReceive: expandBoolPtr(d.Get("permissions.0.can_receive")),
			CanManage:  expandBoolPtr(d.Get("permissions.0.can_manage")),
		},
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(newRegionalIDString(region, credentials.ID))

	_ = d.Set("access_key", credentials.AccessKey)
	_ = d.Set("secret_key", credentials.SecretKey)

	return resourceScalewayMNQSQSCredentialsRead(ctx, d, meta)
}

func resourceScalewayMNQSQSCredentialsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := mnqSQSAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	credentials, err := api.GetSqsCredentials(&mnq.SqsAPIGetSqsCredentialsRequest{
		Region:           region,
		SqsCredentialsID: id,
	}, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", credentials.Name)
	_ = d.Set("region", credentials.Region)
	_ = d.Set("project_id", credentials.ProjectID)

	if credentials.Permissions != nil {
		_ = d.Set("permissions", []map[string]any{{
			"can_publish": credentials.Permissions.CanPublish,
			"can_receive": credentials.Permissions.CanReceive,
			"can_manage":  credentials.Permissions.CanManage,
		}})
	}

	return nil
}

func resourceScalewayMNQSQSCredentialsUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := mnqSQSAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &mnq.SqsAPIUpdateSqsCredentialsRequest{
		Region:           region,
		SqsCredentialsID: id,
	}

	if d.HasChange("name") {
		req.Name = expandUpdatedStringPtr(d.Get("name"))
	}

	if d.HasChange("permissions.0") {
		req.Permissions = &mnq.SqsPermissions{}

		if d.HasChange("permissions.0.can_publish") {
			req.Permissions.CanPublish = expandBoolPtr(d.Get("permissions.0.can_publish"))
		}

		if d.HasChange("permissions.0.can_receive") {
			req.Permissions.CanReceive = expandBoolPtr(d.Get("permissions.0.can_receive"))
		}

		if d.HasChange("permissions.0.can_manage") {
			req.Permissions.CanManage = expandBoolPtr(d.Get("permissions.0.can_manage"))
		}
	}

	if _, err := api.UpdateSqsCredentials(req, scw.WithContext(ctx)); err != nil {
		return diag.FromErr(err)
	}

	return resourceScalewayMNQSQSCredentialsRead(ctx, d, meta)
}

func resourceScalewayMNQSQSCredentialsDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := mnqSQSAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = api.DeleteSqsCredentials(&mnq.SqsAPIDeleteSqsCredentialsRequest{
		Region:           region,
		SqsCredentialsID: id,
	}, scw.WithContext(ctx))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
