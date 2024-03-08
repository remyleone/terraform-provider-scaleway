package marketplace_test

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
)

func CheckInstanceImageExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		zone, ID, err := locality.ParseZonedID(rs.Primary.ID)
		if err != nil {
			return err
		}
		instanceAPI := instance.NewAPI(tt.GetMeta().GetScwClient())
		_, err = instanceAPI.GetImage(&instance.GetImageRequest{
			ImageID: ID,
			Zone:    zone,
		})
		if err != nil {
			return err
		}
		return nil
	}
}
