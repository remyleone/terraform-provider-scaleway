package function_test

import (
	"fmt"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/logging"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/function"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	functionSDK "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func init() {
	resource.AddTestSweepers("scaleway_function_namespace", &resource.Sweeper{
		Name: "scaleway_function_namespace",
		F:    testSweepFunctionNamespace,
	})
}

func testSweepFunctionNamespace(_ string) error {
	return tests.SweepRegions([]scw.Region{scw.RegionFrPar}, func(scwClient *scw.Client, region scw.Region) error {
		functionAPI := functionSDK.NewAPI(scwClient)
		logging.L.Debugf("sweeper: destroying the functionSDK namespaces in (%s)", region)
		listNamespaces, err := functionAPI.ListNamespaces(
			&functionSDK.ListNamespacesRequest{
				Region: region,
			}, scw.WithAllPages())
		if err != nil {
			return fmt.Errorf("error listing namespaces in (%s) in sweeper: %s", region, err)
		}

		for _, ns := range listNamespaces.Namespaces {
			_, err := functionAPI.DeleteNamespace(&functionSDK.DeleteNamespaceRequest{
				NamespaceID: ns.ID,
				Region:      region,
			})
			if err != nil {
				logging.L.Debugf("sweeper: error (%s)", err)

				return fmt.Errorf("error deleting namespace in sweeper: %s", err)
			}
		}

		return nil
	})
}

func TestAccScalewayFunctionNamespace_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayFunctionNamespaceDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_function_namespace main {
						name = "test-cr-ns-01"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFunctionNamespaceExists(tt, "scaleway_function_namespace.main"),
					tests.TestCheckResourceAttrUUID("scaleway_function_namespace.main", "id"),
				),
			},
			{
				Config: `
					resource scaleway_function_namespace main {
						name = "test-cr-ns-01"
						description = "test functionSDK namespace 01"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFunctionNamespaceExists(tt, "scaleway_function_namespace.main"),
					resource.TestCheckResourceAttr("scaleway_function_namespace.main", "description", "test functionSDK namespace 01"),
					resource.TestCheckResourceAttr("scaleway_function_namespace.main", "name", "test-cr-ns-01"),
					tests.TestCheckResourceAttrUUID("scaleway_function_namespace.main", "id"),
				),
			},
			{
				Config: `
					resource scaleway_function_namespace main {
						name = "test-cr-ns-01"
						environment_variables = {
							"test" = "test"
						}
						secret_environment_variables = {
							"test_secret" = "test_secret"
						}
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFunctionNamespaceExists(tt, "scaleway_function_namespace.main"),
					resource.TestCheckResourceAttr("scaleway_function_namespace.main", "description", ""),
					resource.TestCheckResourceAttr("scaleway_function_namespace.main", "name", "test-cr-ns-01"),
					resource.TestCheckResourceAttr("scaleway_function_namespace.main", "environment_variables.test", "test"),
					resource.TestCheckResourceAttr("scaleway_function_namespace.main", "secret_environment_variables.test_secret", "test_secret"),

					tests.TestCheckResourceAttrUUID("scaleway_function_namespace.main", "id"),
				),
			},
		},
	})
}

func TestAccScalewayFunctionNamespace_NoName(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayFunctionNamespaceDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_function_namespace main {
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFunctionNamespaceExists(tt, "scaleway_function_namespace.main"),
				),
			},
		},
	})
}

func TestAccScalewayFunctionNamespace_EnvironmentVariables(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayFunctionNamespaceDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_function_namespace main {
						name = "tf-env-test"
						environment_variables = {
							"test" = "test"
						}
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFunctionNamespaceExists(tt, "scaleway_function_namespace.main"),
					resource.TestCheckResourceAttr("scaleway_function_namespace.main", "name", "tf-env-test"),
					resource.TestCheckResourceAttr("scaleway_function_namespace.main", "environment_variables.test", "test"),

					tests.TestCheckResourceAttrUUID("scaleway_function_namespace.main", "id"),
				),
			},
			{
				Config: `
					resource scaleway_function_namespace main {
						name = "tf-env-test"
						environment_variables = {
							"foo" = "bar"
						}
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFunctionNamespaceExists(tt, "scaleway_function_namespace.main"),
					resource.TestCheckResourceAttr("scaleway_function_namespace.main", "name", "tf-env-test"),
					resource.TestCheckResourceAttr("scaleway_function_namespace.main", "environment_variables.foo", "bar"),

					tests.TestCheckResourceAttrUUID("scaleway_function_namespace.main", "id"),
				),
			},
		},
	})
}

func testAccCheckScalewayFunctionNamespaceExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		api, region, id, err := function.FunctionAPIWithRegionAndID(tt.GetMeta(), rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = api.GetNamespace(&functionSDK.GetNamespaceRequest{
			NamespaceID: id,
			Region:      region,
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckScalewayFunctionNamespaceDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_function_namespace" {
				continue
			}

			api, region, id, err := function.FunctionAPIWithRegionAndID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = api.DeleteNamespace(&functionSDK.DeleteNamespaceRequest{
				NamespaceID: id,
				Region:      region,
			})

			if err == nil {
				return fmt.Errorf("functionSDK namespace (%s) still exists", rs.Primary.ID)
			}

			if !http_errors.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}
