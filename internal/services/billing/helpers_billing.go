package billing

import (
	billing "github.com/scaleway/scaleway-sdk-go/api/billing/v2beta1"
	meta2 "github.com/scaleway/terraform-provider-scaleway/v2/internal/meta"
)

// billingAPI returns a new billing API.
func billingAPI(m interface{}) *billing.API {
	meta := m.(*meta2.Meta)
	return billing.NewAPI(meta.GetScwClient())
}
