package registry_test

import (
	"fmt"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/logging"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/registry"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	registrySDK "github.com/scaleway/scaleway-sdk-go/api/registry/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func init() {
	resource.AddTestSweepers("scaleway_registry_namespace", &resource.Sweeper{
		Name: "scaleway_registry_namespace",
		F:    testSweepRegistryNamespace,
	})
}

func testSweepRegistryNamespace(_ string) error {
	return tests.SweepRegions([]scw.Region{scw.RegionFrPar, scw.RegionNlAms}, func(scwClient *scw.Client, region scw.Region) error {
		registryAPI := registrySDK.NewAPI(scwClient)
		logging.L.Debugf("sweeper: destroying the registry namespaces in (%s)", region)
		listNamespaces, err := registryAPI.ListNamespaces(
			&registrySDK.ListNamespacesRequest{Region: region}, scw.WithAllPages())
		if err != nil {
			return fmt.Errorf("error listing namespaces in (%s) in sweeper: %s", region, err)
		}

		for _, ns := range listNamespaces.Namespaces {
			_, err := registryAPI.DeleteNamespace(&registrySDK.DeleteNamespaceRequest{
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

func TestAccScalewayRegistryNamespace_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayRegistryNamespaceDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_registry_namespace cr01 {
						region = "pl-waw"
						name = "test-cr-ns-01"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRegistryNamespaceExists(tt, "scaleway_registry_namespace.cr01"),
					resource.TestCheckResourceAttr("scaleway_registry_namespace.cr01", "name", "test-cr-ns-01"),
					tests.TestCheckResourceAttrUUID("scaleway_registry_namespace.cr01", "id"),
				),
			},
			{
				Config: `
					resource scaleway_registry_namespace cr01 {
						region = "pl-waw"
						name = "test-cr-ns-01"
						description = "test registry namespace 01"
						is_public = true
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRegistryNamespaceExists(tt, "scaleway_registry_namespace.cr01"),
					resource.TestCheckResourceAttr("scaleway_registry_namespace.cr01", "description", "test registry namespace 01"),
					resource.TestCheckResourceAttr("scaleway_registry_namespace.cr01", "is_public", "true"),
					tests.TestCheckResourceAttrUUID("scaleway_registry_namespace.cr01", "id"),
				),
			},
		},
	})
}

func testAccCheckScalewayRegistryNamespaceExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		api, region, id, err := registry.RegistryAPIWithRegionAndID(tt.GetMeta(), rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = api.GetNamespace(&registrySDK.GetNamespaceRequest{
			NamespaceID: id,
			Region:      region,
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckScalewayRegistryNamespaceDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_registry_namespace" {
				continue
			}

			api, region, id, err := registry.RegistryAPIWithRegionAndID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = api.DeleteNamespace(&registrySDK.DeleteNamespaceRequest{
				NamespaceID: id,
				Region:      region,
			})

			if err == nil {
				return fmt.Errorf("namespace (%s) still exists", rs.Primary.ID)
			}

			if !http_errors.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}
