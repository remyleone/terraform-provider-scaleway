package project

import (
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/meta"
)

// ExtractProjectID will try to guess the project id from the following:
//   - project_id field of the resource data
//   - default project id from config
func ExtractProjectID(d meta.TerraformResourceData, meta *meta.Meta) (projectID string, isDefault bool, err error) {
	rawProjectID, exist := d.GetOk("project_id")
	if exist {
		return rawProjectID.(string), false, nil
	}

	defaultProjectID, exist := meta.GetScwClient().GetDefaultProjectID()
	if exist {
		return defaultProjectID, true, nil
	}

	return "", false, ErrProjectIDNotFound
}
