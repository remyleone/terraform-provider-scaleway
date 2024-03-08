package iam_test

import (
	"errors"
	"fmt"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/iam"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	iamSDK "github.com/scaleway/scaleway-sdk-go/api/iam/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func init() {
	resource.AddTestSweepers("scaleway_iam_user", &resource.Sweeper{
		Name: "scaleway_iam_user",
		F:    testSweepIamUser,
	})
}

func testSweepIamUser(_ string) error {
	return tests.Sweep(func(scwClient *scw.Client) error {
		api := iamSDK.NewAPI(scwClient)

		orgID, exists := scwClient.GetDefaultOrganizationID()
		if !exists {
			return errors.New("missing organizationID")
		}

		listUsers, err := api.ListUsers(&iamSDK.ListUsersRequest{
			OrganizationID: &orgID,
		})
		if err != nil {
			return fmt.Errorf("failed to list users: %w", err)
		}
		for _, user := range listUsers.Users {
			if !tests.IsTestResource(user.Email) {
				continue
			}
			err = api.DeleteUser(&iamSDK.DeleteUserRequest{
				UserID: user.ID,
			})
			if err != nil {
				return fmt.Errorf("failed to delete user: %w", err)
			}
		}
		return nil
	})
}

func TestAccScalewayIamUser_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayIamUserDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
						resource "scaleway_iam_user" "user_basic" {
							email = "foo@scaleway.com"
						}
					`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayIamUserExists(tt, "scaleway_iam_user.user_basic"),
					tests.TestCheckResourceAttrUUID("scaleway_iam_user.user_basic", "id"),
					resource.TestCheckResourceAttr("scaleway_iam_user.user_basic", "email", "foo@scaleway.com"),
				),
			},
		},
	})
}

func testAccCheckScalewayIamUserDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "scaleway_iam_user" {
				continue
			}

			iamAPI := iam.IAMAPI(tt.GetMeta())

			_, err := iamAPI.GetUser(&iamSDK.GetUserRequest{
				UserID: rs.Primary.ID,
			})

			// If no error resource still exist
			if err == nil {
				return fmt.Errorf("resource %s(%s) still exist", rs.Type, rs.Primary.ID)
			}

			// Unexpected api error we return it
			if !http_errors.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}
