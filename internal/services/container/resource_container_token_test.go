package container_test

import (
	"fmt"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/container"
	"testing"
	"time"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	containerSDK "github.com/scaleway/scaleway-sdk-go/api/container/v1beta1"
)

func TestAccScalewayContainerToken_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	expiresAt := time.Now().Add(time.Hour * 24).Format(time.RFC3339)
	if !*tests.UpdateCassettes {
		expiresAt = "2023-01-05T14:12:46+01:00"
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayContainerTokenDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource scaleway_container_namespace main {
						name = "test-containerSDK-token-ns"
					}

					resource scaleway_container main {
						namespace_id = scaleway_container_namespace.main.id
					}

					resource scaleway_container_token namespace {
						namespace_id = scaleway_container_namespace.main.id
						expires_at = "%s"
					}

					resource scaleway_container_token containerSDK {
						container_id = scaleway_container.main.id
					}
				`, expiresAt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayContainerTokenExists(tt, "scaleway_container_token.namespace"),
					testAccCheckScalewayContainerTokenExists(tt, "scaleway_container_token.containerSDK"),
					tests.TestCheckResourceAttrUUID("scaleway_container_token.namespace", "id"),
					tests.TestCheckResourceAttrUUID("scaleway_container_token.containerSDK", "id"),
					resource.TestCheckResourceAttrSet("scaleway_container_token.namespace", "token"),
					resource.TestCheckResourceAttrSet("scaleway_container_token.containerSDK", "token"),
				),
			},
		},
	})
}

func testAccCheckScalewayContainerTokenExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		api, region, id, err := container.ContainerAPIWithRegionAndID(tt.GetMeta(), rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = api.GetToken(&containerSDK.GetTokenRequest{
			TokenID: id,
			Region:  region,
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckScalewayContainerTokenDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_container_token" {
				continue
			}

			api, region, id, err := container.ContainerAPIWithRegionAndID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = api.DeleteToken(&containerSDK.DeleteTokenRequest{
				TokenID: id,
				Region:  region,
			})

			if err == nil {
				return fmt.Errorf("containerSDK token (%s) still exists", rs.Primary.ID)
			}

			if !http_errors.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}
