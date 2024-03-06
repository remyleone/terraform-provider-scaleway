package account

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/iam"
)

func ResourceScalewayAccountSSKKey() *schema.Resource {
	return iam.ResourceScalewayIamSSKKey()
}
