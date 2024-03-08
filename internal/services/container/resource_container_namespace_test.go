package container_test

import (
	"fmt"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/logging"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/container"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/registry"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	containerSDK "github.com/scaleway/scaleway-sdk-go/api/container/v1beta1"
	registrySDK "github.com/scaleway/scaleway-sdk-go/api/registry/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func init() {
	resource.AddTestSweepers("scaleway_container_namespace", &resource.Sweeper{
		Name:         "scaleway_container_namespace",
		F:            testSweepContainerNamespace,
		Dependencies: []string{"scaleway_container"},
	})
}

func testSweepContainerNamespace(_ string) error {
	return tests.SweepRegions([]scw.Region{scw.RegionFrPar}, func(scwClient *scw.Client, region scw.Region) error {
		containerAPI := containerSDK.NewAPI(scwClient)
		logging.L.Debugf("sweeper: destroying the containerSDK namespaces in (%s)", region)
		listNamespaces, err := containerAPI.ListNamespaces(
			&containerSDK.ListNamespacesRequest{
				Region: region,
			}, scw.WithAllPages())
		if err != nil {
			return fmt.Errorf("error listing namespaces in (%s) in sweeper: %s", region, err)
		}

		for _, ns := range listNamespaces.Namespaces {
			_, err := containerAPI.DeleteNamespace(&containerSDK.DeleteNamespaceRequest{
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

func TestAccScalewayContainerNamespace_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayContainerNamespaceDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_container_namespace main {
						name = "test-cr-ns-01"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayContainerNamespaceExists(tt, "scaleway_container_namespace.main"),
					tests.TestCheckResourceAttrUUID("scaleway_container_namespace.main", "id"),
				),
			},
			{
				Config: `
					resource scaleway_container_namespace main {
						name = "test-cr-ns-01"
						description = "test containerSDK namespace 01"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayContainerNamespaceExists(tt, "scaleway_container_namespace.main"),
					resource.TestCheckResourceAttr("scaleway_container_namespace.main", "description", "test containerSDK namespace 01"),
					resource.TestCheckResourceAttr("scaleway_container_namespace.main", "name", "test-cr-ns-01"),
					tests.TestCheckResourceAttrUUID("scaleway_container_namespace.main", "id"),
				),
			},
			{
				Config: `
					resource scaleway_container_namespace main {
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
					testAccCheckScalewayContainerNamespaceExists(tt, "scaleway_container_namespace.main"),
					resource.TestCheckResourceAttr("scaleway_container_namespace.main", "description", ""),
					resource.TestCheckResourceAttr("scaleway_container_namespace.main", "name", "test-cr-ns-01"),
					resource.TestCheckResourceAttr("scaleway_container_namespace.main", "environment_variables.test", "test"),
					resource.TestCheckResourceAttr("scaleway_container_namespace.main", "secret_environment_variables.test_secret", "test_secret"),

					tests.TestCheckResourceAttrUUID("scaleway_container_namespace.main", "id"),
				),
			},
			{
				Config: `
					resource scaleway_container_namespace main {
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayContainerNamespaceExists(tt, "scaleway_container_namespace.main"),
					resource.TestCheckResourceAttrSet("scaleway_container_namespace.main", "name"),
					resource.TestCheckResourceAttrSet("scaleway_container_namespace.main", "registry_endpoint"),
					resource.TestCheckResourceAttrSet("scaleway_container_namespace.main", "registry_namespace_id"),
				),
			},
			{
				Config: `
					resource scaleway_container_namespace main {
						name = "tf-env-test"
						environment_variables = {
							"test" = "test"
						}
						secret_environment_variables = {
							"test_secret" = "test_secret"
						}
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayContainerNamespaceExists(tt, "scaleway_container_namespace.main"),
					resource.TestCheckResourceAttr("scaleway_container_namespace.main", "name", "tf-env-test"),
					resource.TestCheckResourceAttr("scaleway_container_namespace.main", "environment_variables.test", "test"),
					resource.TestCheckResourceAttr("scaleway_container_namespace.main", "secret_environment_variables.test_secret", "test_secret"),

					tests.TestCheckResourceAttrUUID("scaleway_container_namespace.main", "id"),
				),
			},
			{
				Config: `
					resource scaleway_container_namespace main {
						name = "tf-env-test"
						environment_variables = {
							"foo" = "bar"
						}
						secret_environment_variables = {
							"foo_secret" = "bar_secret"
						}
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayContainerNamespaceExists(tt, "scaleway_container_namespace.main"),
					resource.TestCheckResourceAttr("scaleway_container_namespace.main", "name", "tf-env-test"),
					resource.TestCheckResourceAttr("scaleway_container_namespace.main", "environment_variables.foo", "bar"),
					resource.TestCheckResourceAttr("scaleway_container_namespace.main", "secret_environment_variables.foo_secret", "bar_secret"),

					tests.TestCheckResourceAttrUUID("scaleway_container_namespace.main", "id"),
				),
			},
		},
	})
}

func TestAccScalewayContainerNamespace_DestroyRegistry(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckScalewayContainerNamespaceDestroy(tt),
			testAccCheckScalewayContainerRegistryDestroy(tt),
		),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_container_namespace main {
						region = "pl-waw"
						name = "test-cr-ns-01"
						destroy_registry = true
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayContainerNamespaceExists(tt, "scaleway_container_namespace.main"),
					tests.TestCheckResourceAttrUUID("scaleway_container_namespace.main", "id"),
				),
			},
		},
	})
}

func testAccCheckScalewayContainerNamespaceExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		api, region, id, err := container.ContainerAPIWithRegionAndID(tt.GetMeta(), rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = api.GetNamespace(&containerSDK.GetNamespaceRequest{
			NamespaceID: id,
			Region:      region,
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckScalewayContainerNamespaceDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_container_namespace" { //nolint:goconst
				continue
			}

			api, region, id, err := container.ContainerAPIWithRegionAndID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = api.DeleteNamespace(&containerSDK.DeleteNamespaceRequest{
				NamespaceID: id,
				Region:      region,
			})

			if err == nil {
				return fmt.Errorf("containerSDK namespace (%s) still exists", rs.Primary.ID)
			}

			if !http_errors.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}

func testAccCheckScalewayContainerRegistryDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_container_namespace" {
				continue
			}

			api, region, _, err := registry.RegistryAPIWithRegionAndID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = api.DeleteNamespace(&registrySDK.DeleteNamespaceRequest{
				NamespaceID: rs.Primary.Attributes["registry_namespace_id"],
				Region:      region,
			})

			if err == nil {
				return fmt.Errorf("registrySDK namespace (%s) still exists", rs.Primary.Attributes["registry_namespace_id"])
			}

			if !http_errors.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}
