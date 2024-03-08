package account_test

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	iamSDK "github.com/scaleway/scaleway-sdk-go/api/iam/v1alpha1"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/iam"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
)

func testAccCheckScalewayIamSSHKeyDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_iam_ssh_key" {
				continue
			}

			iamAPI := iam.IAMAPI(tt.GetMeta())

			_, err := iamAPI.GetSSHKey(&iamSDK.GetSSHKeyRequest{
				SSHKeyID: rs.Primary.ID,
			})

			// If no error resource still exist
			if err == nil {
				return fmt.Errorf("SSH key (%s) still exists", rs.Primary.ID)
			}

			// Unexpected api error we return it
			if !http_errors.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}

func testAccCheckScalewayIamSSHKeyExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		iamAPI := iam.IAMAPI(tt.GetMeta())

		_, err := iamAPI.GetSSHKey(&iamSDK.GetSSHKeyRequest{
			SSHKeyID: rs.Primary.ID,
		})
		if err != nil {
			return err
		}

		return nil
	}
}
