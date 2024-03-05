package scaleway

import (
	"fmt"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	function "github.com/scaleway/scaleway-sdk-go/api/function/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func init() {
	resource.AddTestSweepers("scaleway_function", &resource.Sweeper{
		Name: "scaleway_function",
		F:    testSweepFunction,
	})
}

func testSweepFunction(_ string) error {
	return SweepRegions([]scw.Region{scw.RegionFrPar}, func(scwClient *scw.Client, region scw.Region) error {
		functionAPI := function.NewAPI(scwClient)
		L.Debugf("sweeper: destroying the function in (%s)", region)
		listFunctions, err := functionAPI.ListFunctions(
			&function.ListFunctionsRequest{
				Region: region,
			}, scw.WithAllPages())
		if err != nil {
			return fmt.Errorf("error listing functions in (%s) in sweeper: %s", region, err)
		}

		for _, f := range listFunctions.Functions {
			_, err := functionAPI.DeleteFunction(&function.DeleteFunctionRequest{
				FunctionID: f.ID,
				Region:     region,
			})
			if err != nil && !http_errors.Is404Error(err) {
				L.Debugf("sweeper: error (%s)", err)

				return fmt.Errorf("error deleting functions in sweeper: %s", err)
			}
		}

		return nil
	})
}

func TestAccScalewayFunction_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayFunctionDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_function_namespace main {}

					resource scaleway_function main {
						name = "foobar"
						namespace_id = scaleway_function_namespace.main.id
						runtime = "node14"
						privacy = "private"
						handler = "handler.handle"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFunctionExists(tt, "scaleway_function.main"),
					resource.TestCheckResourceAttr("scaleway_function.main", "name", "foobar"),
					resource.TestCheckResourceAttr("scaleway_function.main", "runtime", "node14"),
					resource.TestCheckResourceAttr("scaleway_function.main", "privacy", "private"),
					resource.TestCheckResourceAttr("scaleway_function.main", "handler", "handler.handle"),
					resource.TestCheckResourceAttrSet("scaleway_function.main", "namespace_id"),
					resource.TestCheckResourceAttrSet("scaleway_function.main", "region"),
				),
			},
		},
	})
}

func TestAccScalewayFunction_Timeout(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayFunctionDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_function_namespace main {}

					resource scaleway_function main {
						name = "foobar"
						namespace_id = scaleway_function_namespace.main.id
						runtime = "node14"
						privacy = "private"
						handler = "handler.handle"
						timeout = 10
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFunctionExists(tt, "scaleway_function.main"),
					resource.TestCheckResourceAttr("scaleway_function.main", "name", "foobar"),
					resource.TestCheckResourceAttr("scaleway_function.main", "runtime", "node14"),
					resource.TestCheckResourceAttr("scaleway_function.main", "privacy", "private"),
					resource.TestCheckResourceAttr("scaleway_function.main", "handler", "handler.handle"),
					resource.TestCheckResourceAttr("scaleway_function.main", "timeout", "10"),
				),
			},
		},
	})
}

func TestAccScalewayFunction_NoName(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayFunctionNamespaceDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_function_namespace main {}

					resource scaleway_function main {
						namespace_id = scaleway_function_namespace.main.id
						runtime = "node14"
						privacy = "private"
						handler = "handler.handle"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFunctionExists(tt, "scaleway_function.main"),
					resource.TestCheckResourceAttrSet("scaleway_function.main", "name"),
				),
			},
		},
	})
}

func TestAccScalewayFunction_EnvironmentVariables(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayFunctionDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_function_namespace main {}

					resource scaleway_function main {
						name = "foobar"
						namespace_id = scaleway_function_namespace.main.id
						runtime = "node14"
						privacy = "private"
						handler = "handler.handle"
						environment_variables = {
							"test" = "test"
						}

						secret_environment_variables = {
							"test_secret" = "test_secret"
						}
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFunctionExists(tt, "scaleway_function.main"),
					resource.TestCheckResourceAttr("scaleway_function.main", "environment_variables.test", "test"),
					resource.TestCheckResourceAttr("scaleway_function.main", "secret_environment_variables.test_secret", "test_secret"),
				),
			},
			{
				Config: `
					resource scaleway_function_namespace main {}

					resource scaleway_function main {
						name = "foobar"
						namespace_id = scaleway_function_namespace.main.id
						runtime = "node14"
						privacy = "private"
						handler = "handler.handle"
						environment_variables = {
							"foo" = "bar"
						}
						
						secret_environment_variables = {
							"foo_secret" = "bar_secret"
						}
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFunctionExists(tt, "scaleway_function.main"),
					resource.TestCheckResourceAttr("scaleway_function.main", "environment_variables.foo", "bar"),
					resource.TestCheckResourceAttr("scaleway_function.main", "secret_environment_variables.foo_secret", "bar_secret"),
				),
			},
		},
	})
}

func TestAccScalewayFunction_Upload(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayFunctionDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_function_namespace main {}

					resource scaleway_function main {
						name = "foobar"
						namespace_id = scaleway_function_namespace.main.id
						runtime = "go118"
						privacy = "private"
						handler = "Handle"
						zip_file = "testfixture/gofunction.zip"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFunctionExists(tt, "scaleway_function.main"),
				),
			},
		},
	})
}

func TestAccScalewayFunction_Deploy(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayFunctionDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_function_namespace main {}

					resource scaleway_function main {
						name = "foobar"
						namespace_id = scaleway_function_namespace.main.id
						runtime = "go118"
						privacy = "private"
						handler = "Handle"
						zip_file = "testfixture/gofunction.zip"
						deploy = true
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFunctionExists(tt, "scaleway_function.main"),
				),
			},
			{
				Config: `
					resource scaleway_function_namespace main {}

					resource scaleway_function main {
						name = "foobar"
						namespace_id = scaleway_function_namespace.main.id
						runtime = "go118"
						privacy = "private"
						handler = "Handle"
						zip_file = "testfixture/gofunction.zip"
						zip_hash = "value"
						deploy = true
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFunctionExists(tt, "scaleway_function.main"),
				),
			},
		},
	})
}

func TestAccScalewayFunction_HTTPOption(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayFunctionDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_function_namespace main {}

					resource scaleway_function main {
						name = "foobar"
						namespace_id = scaleway_function_namespace.main.id
						runtime = "node14"
						privacy = "private"
						handler = "handler.handle"
						http_option = "enabled"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFunctionExists(tt, "scaleway_function.main"),
					resource.TestCheckResourceAttr("scaleway_function.main", "http_option", function.FunctionHTTPOptionEnabled.String()),
				),
			},
			{
				Config: `
					resource scaleway_function_namespace main {}

					resource scaleway_function main {
						name = "foobar"
						namespace_id = scaleway_function_namespace.main.id
						runtime = "node14"
						privacy = "private"
						handler = "handler.handle"
						http_option = "redirected"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFunctionExists(tt, "scaleway_function.main"),
					resource.TestCheckResourceAttr("scaleway_function.main", "http_option", function.FunctionHTTPOptionRedirected.String()),
				),
			},
			{
				Config: `
					resource scaleway_function_namespace main {}

					resource scaleway_function main {
						name = "foobar"
						namespace_id = scaleway_function_namespace.main.id
						runtime = "node14"
						privacy = "private"
						handler = "handler.handle"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFunctionExists(tt, "scaleway_function.main"),
					resource.TestCheckResourceAttr("scaleway_function.main", "http_option", function.FunctionHTTPOptionEnabled.String()),
				),
			},
		},
	})
}

func testAccCheckScalewayFunctionExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		api, region, id, err := functionAPIWithRegionAndID(tt.Meta, rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = api.GetFunction(&function.GetFunctionRequest{
			FunctionID: id,
			Region:     region,
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckScalewayFunctionDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_function" {
				continue
			}

			api, region, id, err := functionAPIWithRegionAndID(tt.Meta, rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = api.DeleteFunction(&function.DeleteFunctionRequest{
				FunctionID: id,
				Region:     region,
			})

			if err == nil {
				return fmt.Errorf("function (%s) still exists", rs.Primary.ID)
			}

			if !http_errors.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}
