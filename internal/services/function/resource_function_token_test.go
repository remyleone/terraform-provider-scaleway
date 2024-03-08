package function_test

import (
	"fmt"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/function"
	"testing"
	"time"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	functionSDK "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
)

func TestAccScalewayFunctionToken_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	expiresAt := time.Now().Add(time.Hour * 24).Format(time.RFC3339)
	if !*tests.UpdateCassettes {
		expiresAt = "2023-01-05T13:53:11+01:00"
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayFunctionTokenDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource scaleway_function_namespace main {
						name = "test-functionSDK-token-ns"
					}

					resource scaleway_function main {
						namespace_id = scaleway_function_namespace.main.id
						runtime = "node14"
						privacy = "private"
						handler = "handler.handle"
					}

					resource scaleway_function_token namespace {
						namespace_id = scaleway_function_namespace.main.id
						expires_at = "%s"
					}

					resource scaleway_function_token functionSDK {
						function_id = scaleway_function.main.id
					}
				`, expiresAt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFunctionTokenExists(tt, "scaleway_function_token.namespace"),
					testAccCheckScalewayFunctionTokenExists(tt, "scaleway_function_token.functionSDK"),
					tests.TestCheckResourceAttrUUID("scaleway_function_token.namespace", "id"),
					tests.TestCheckResourceAttrUUID("scaleway_function_token.functionSDK", "id"),
					resource.TestCheckResourceAttrSet("scaleway_function_token.namespace", "token"),
					resource.TestCheckResourceAttrSet("scaleway_function_token.functionSDK", "token"),
				),
			},
		},
	})
}

func testAccCheckScalewayFunctionTokenExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		api, region, id, err := function.FunctionAPIWithRegionAndID(tt.GetMeta(), rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = api.GetToken(&functionSDK.GetTokenRequest{
			TokenID: id,
			Region:  region,
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckScalewayFunctionTokenDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_function_token" {
				continue
			}

			api, region, id, err := function.FunctionAPIWithRegionAndID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = api.DeleteToken(&functionSDK.DeleteTokenRequest{
				TokenID: id,
				Region:  region,
			})

			if err == nil {
				return fmt.Errorf("functionSDK token (%s) still exists", rs.Primary.ID)
			}

			if !http_errors.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}
