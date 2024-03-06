package function

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	meta2 "github.com/scaleway/terraform-provider-scaleway/v2/internal/meta"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/project"
)

func expandFunctionTriggerMnqSqsCreationConfig(i interface{}) *function.CreateTriggerRequestMnqSqsClientConfig {
	m := i.(map[string]interface{})

	mnqNamespaceID := locality.ExpandID(m["namespace_id"].(string))

	req := &function.CreateTriggerRequestMnqSqsClientConfig{
		Queue:        m["queue"].(string),
		MnqProjectID: m["project_id"].(string),
		MnqRegion:    m["region"].(string),
	}

	if mnqNamespaceID != "" {
		req.MnqNamespaceID = &mnqNamespaceID
	}

	return req
}

func expandFunctionTriggerMnqNatsCreationConfig(i interface{}) *function.CreateTriggerRequestMnqNatsClientConfig {
	m := i.(map[string]interface{})

	return &function.CreateTriggerRequestMnqNatsClientConfig{
		Subject:          locality.ExpandID(m["subject"]),
		MnqProjectID:     m["project_id"].(string),
		MnqRegion:        m["region"].(string),
		MnqNatsAccountID: locality.ExpandID(m["account_id"]),
	}
}

func completeFunctionTriggerMnqCreationConfig(i interface{}, d *schema.ResourceData, meta interface{}, region scw.Region) error {
	m := i.(map[string]interface{})

	if sqsRegion, exists := m["region"]; !exists || sqsRegion == "" {
		m["region"] = region.String()
	}

	if projectID, exists := m["project_id"]; !exists || projectID == "" {
		projectID, _, err := project.ExtractProjectID(d, meta.(*meta2.Meta))
		if err == nil {
			m["project_id"] = projectID
		}
	}

	return nil
}
