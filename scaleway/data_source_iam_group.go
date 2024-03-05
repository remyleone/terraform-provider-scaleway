package scaleway

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	iam "github.com/scaleway/scaleway-sdk-go/api/iam/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/types"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/verify"
)

func dataSourceScalewayIamGroup() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasourceSchemaFromResourceSchema(resourceScalewayIamGroup().Schema)

	addOptionalFieldsToSchema(dsSchema, "name")

	dsSchema["name"].ConflictsWith = []string{"group_id"}
	dsSchema["group_id"] = &schema.Schema{
		Type:          schema.TypeString,
		Optional:      true,
		Description:   "The ID of the IAM group",
		ConflictsWith: []string{"name"},
		ValidateFunc:  verify.UUID(),
	}
	dsSchema["organization_id"] = &schema.Schema{
		Type:        schema.TypeString,
		Description: "The organization_id you want to attach the resource to",
		Optional:    true,
	}

	return &schema.Resource{
		ReadContext: dataSourceScalewayIamGroupRead,
		Schema:      dsSchema,
	}
}

func dataSourceScalewayIamGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := iamAPI(meta)

	groupID, groupIDExists := d.GetOk("group_id")
	if !groupIDExists {
		groupName := d.Get("name").(string)
		req := &iam.ListGroupsRequest{
			OrganizationID: flattenStringPtr(getOrganizationID(meta, d)).(string),
			Name:           types.ExpandStringPtr(groupName),
		}

		res, err := api.ListGroups(req, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}

		foundGroup, err := findExact(
			res.Groups,
			func(s *iam.Group) bool { return s.Name == groupName },
			groupName,
		)
		if err != nil {
			return diag.FromErr(err)
		}

		groupID = foundGroup.ID
	}

	d.SetId(groupID.(string))
	err := d.Set("group_id", groupID)
	if err != nil {
		return diag.FromErr(err)
	}

	diags := resourceScalewayIamGroupRead(ctx, d, meta)
	if diags != nil {
		return append(diags, diag.Errorf("failed to read iam group state")...)
	}

	if d.Id() == "" {
		return diag.Errorf("iam group (%s) not found", groupID)
	}

	return nil
}
