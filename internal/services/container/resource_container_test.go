package container_test

import (
	"fmt"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/logging"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/container"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests/checks"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	containerSDK "github.com/scaleway/scaleway-sdk-go/api/container/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func init() {
	resource.AddTestSweepers("scaleway_container", &resource.Sweeper{
		Name: "scaleway_container",
		F:    testSweepContainer,
	})
}

func testSweepContainer(_ string) error {
	return tests.SweepRegions(scw.AllRegions, func(scwClient *scw.Client, region scw.Region) error {
		containerAPI := containerSDK.NewAPI(scwClient)
		logging.L.Debugf("sweeper: destroying the containerSDK in (%s)", region)
		listNamespaces, err := containerAPI.ListContainers(
			&containerSDK.ListContainersRequest{
				Region: region,
			}, scw.WithAllPages())
		if err != nil {
			return fmt.Errorf("error listing containers in (%s) in sweeper: %s", region, err)
		}

		for _, cont := range listNamespaces.Containers {
			_, err := containerAPI.DeleteContainer(&containerSDK.DeleteContainerRequest{
				ContainerID: cont.ID,
				Region:      region,
			})
			if err != nil {
				logging.L.Debugf("sweeper: error (%s)", err)

				return fmt.Errorf("error deleting containerSDK in sweeper: %s", err)
			}
		}

		return nil
	})
}

func TestAccScalewayContainer_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayContainerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_container_namespace main {
					}

					resource scaleway_container main {
						namespace_id = scaleway_container_namespace.main.id
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayContainerExists(tt, "scaleway_container.main"),
					tests.TestCheckResourceAttrUUID("scaleway_container_namespace.main", "id"),
					tests.TestCheckResourceAttrUUID("scaleway_container.main", "id"),
					resource.TestCheckResourceAttrSet("scaleway_container.main", "name"),
					resource.TestCheckResourceAttrSet("scaleway_container.main", "registry_image"),
					resource.TestCheckResourceAttrSet("scaleway_container.main", "domain_name"),
					resource.TestCheckResourceAttrSet("scaleway_container.main", "max_concurrency"),
					resource.TestCheckResourceAttrSet("scaleway_container.main", "domain_name"),
					resource.TestCheckResourceAttrSet("scaleway_container.main", "protocol"),
					resource.TestCheckResourceAttrSet("scaleway_container.main", "cpu_limit"),
					resource.TestCheckResourceAttrSet("scaleway_container.main", "timeout"),
					resource.TestCheckResourceAttrSet("scaleway_container.main", "memory_limit"),
					resource.TestCheckResourceAttrSet("scaleway_container.main", "max_scale"),
					resource.TestCheckResourceAttrSet("scaleway_container.main", "min_scale"),
					resource.TestCheckResourceAttrSet("scaleway_container.main", "privacy"),
				),
			},
			{
				Config: `
					resource scaleway_container_namespace main {
					}

					resource scaleway_container main {
						name = "my-containerSDK-tf"
						namespace_id = scaleway_container_namespace.main.id
						port = 8080
						cpu_limit = 70
						memory_limit = 128
						min_scale = 0
						max_scale = 20
						timeout = 300
						deploy = false
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayContainerExists(tt, "scaleway_container.main"),
					tests.TestCheckResourceAttrUUID("scaleway_container.main", "id"),
					resource.TestCheckResourceAttr("scaleway_container.main", "name", "my-containerSDK-tf"),
					resource.TestCheckResourceAttr("scaleway_container.main", "port", "8080"),
					resource.TestCheckResourceAttr("scaleway_container.main", "cpu_limit", "70"),
					resource.TestCheckResourceAttr("scaleway_container.main", "memory_limit", "128"),
					resource.TestCheckResourceAttr("scaleway_container.main", "min_scale", "0"),
					resource.TestCheckResourceAttr("scaleway_container.main", "max_scale", "20"),
					resource.TestCheckResourceAttr("scaleway_container.main", "timeout", "300"),
					resource.TestCheckResourceAttr("scaleway_container.main", "max_concurrency", "50"),
					resource.TestCheckResourceAttr("scaleway_container.main", "deploy", "false"),
					resource.TestCheckResourceAttr("scaleway_container.main", "privacy", containerSDK.ContainerPrivacyPublic.String()),
					resource.TestCheckResourceAttr("scaleway_container.main", "protocol", containerSDK.ContainerProtocolHTTP1.String()),
				),
			},
			{
				Config: `
					resource scaleway_container_namespace main {
					}

					resource "scaleway_container" main {
						name 			= "my-containerSDK-tf"
						namespace_id	= scaleway_container_namespace.main.id
						port         	= 5000
						min_scale    	= 1
						max_scale    	= 2
						max_concurrency = 80
						memory_limit 	= 256
						cpu_limit		= 280
						deploy       	= false
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayContainerExists(tt, "scaleway_container.main"),
					tests.TestCheckResourceAttrUUID("scaleway_container.main", "id"),
					resource.TestCheckResourceAttr("scaleway_container.main", "name", "my-containerSDK-tf"),
					resource.TestCheckResourceAttr("scaleway_container.main", "port", "5000"),
					resource.TestCheckResourceAttr("scaleway_container.main", "cpu_limit", "280"),
					resource.TestCheckResourceAttr("scaleway_container.main", "memory_limit", "256"),
					resource.TestCheckResourceAttr("scaleway_container.main", "min_scale", "1"),
					resource.TestCheckResourceAttr("scaleway_container.main", "max_scale", "2"),
					resource.TestCheckResourceAttr("scaleway_container.main", "timeout", "300"),
					resource.TestCheckResourceAttr("scaleway_container.main", "max_concurrency", "80"),
					resource.TestCheckResourceAttr("scaleway_container.main", "deploy", "false"),
					resource.TestCheckResourceAttr("scaleway_container.main", "protocol", containerSDK.ContainerProtocolHTTP1.String()),
				),
			},
		},
	})
}

func TestAccScalewayContainer_Env(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayContainerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_container_namespace main {
					}

					resource scaleway_container main {
						namespace_id = scaleway_container_namespace.main.id
						environment_variables = {
							"test" = "test"
						}
						secret_environment_variables = {
							"test_secret" = "test_secret"
						}
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayContainerExists(tt, "scaleway_container.main"),
					tests.TestCheckResourceAttrUUID("scaleway_container_namespace.main", "id"),
					tests.TestCheckResourceAttrUUID("scaleway_container.main", "id"),
					resource.TestCheckResourceAttr("scaleway_container.main", "environment_variables.test", "test"),
					resource.TestCheckResourceAttr("scaleway_container.main", "secret_environment_variables.test_secret", "test_secret"),
				),
			},
			{
				Config: `
					resource scaleway_container_namespace main {
					}

					resource scaleway_container main {
						namespace_id = scaleway_container_namespace.main.id
						environment_variables = {
							"foo" = "bar"
						}
						secret_environment_variables = {
							"foo_secret" = "bar_secret"
						}
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayContainerExists(tt, "scaleway_container.main"),
					tests.TestCheckResourceAttrUUID("scaleway_container_namespace.main", "id"),
					tests.TestCheckResourceAttrUUID("scaleway_container.main", "id"),
					resource.TestCheckResourceAttr("scaleway_container.main", "environment_variables.foo", "bar"),
					resource.TestCheckResourceAttr("scaleway_container.main", "secret_environment_variables.foo_secret", "bar_secret"),
				),
			},
			{
				Config: `
					resource scaleway_container_namespace main {
					}

					resource scaleway_container main {
						namespace_id = scaleway_container_namespace.main.id
						environment_variables = {}
						secret_environment_variables = {}
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayContainerExists(tt, "scaleway_container.main"),
					tests.TestCheckResourceAttrUUID("scaleway_container_namespace.main", "id"),
					tests.TestCheckResourceAttrUUID("scaleway_container.main", "id"),
					resource.TestCheckNoResourceAttr("scaleway_container.main", "environment_variables.%"),
					resource.TestCheckNoResourceAttr("scaleway_container.main", "secret_environment_variables.%"),
					resource.TestCheckNoResourceAttr("scaleway_container.main", "environment_variables.foo"),
					resource.TestCheckNoResourceAttr("scaleway_container.main", "secret_environment_variables.foo_secret"),
				),
			},
		},
	})
}

func TestAccScalewayContainer_WithIMG(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	containerNamespace := "test-cr-ns-02"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayContainerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource scaleway_container_namespace main {
						name = "%s"
						description = "test containerSDK"
					}
				`, containerNamespace),
			},
			{
				Config: fmt.Sprintf(`
					resource scaleway_container_namespace main {
						name = "%s"
						description = "test containerSDK"
					}
				`, containerNamespace),
				Check: resource.ComposeTestCheckFunc(
					checks.CheckConfigContainerNamespace(tt, "scaleway_container_namespace.main"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource scaleway_container_namespace main {
						name = "%s"
						description = "test containerSDK"
					}

					resource scaleway_container main {
						name = "my-containerSDK-02"
						description = "environment variables test"
						namespace_id = scaleway_container_namespace.main.id
						registry_image = "${scaleway_container_namespace.main.registry_endpoint}/nginx:test"
						port = 80
						cpu_limit = 140
						memory_limit = 256
						min_scale = 3
						max_scale = 5
						timeout = 600
						max_concurrency = 80
						privacy = "private"
						protocol = "h2c"
						deploy = true

						environment_variables = {
							"foo" = "var"
						}
					}
				`, containerNamespace),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayContainerExists(tt, "scaleway_container.main"),
					tests.TestCheckResourceAttrUUID("scaleway_container.main", "id"),
					resource.TestCheckResourceAttrSet("scaleway_container.main", "registry_image"),
					resource.TestCheckResourceAttr("scaleway_container.main", "name", "my-containerSDK-02"),
					resource.TestCheckResourceAttr("scaleway_container.main", "port", "80"),
					resource.TestCheckResourceAttr("scaleway_container.main", "cpu_limit", "140"),
					resource.TestCheckResourceAttr("scaleway_container.main", "memory_limit", "256"),
					resource.TestCheckResourceAttr("scaleway_container.main", "min_scale", "3"),
					resource.TestCheckResourceAttr("scaleway_container.main", "max_scale", "5"),
					resource.TestCheckResourceAttr("scaleway_container.main", "timeout", "600"),
					resource.TestCheckResourceAttr("scaleway_container.main", "max_concurrency", "80"),
					resource.TestCheckResourceAttr("scaleway_container.main", "deploy", "true"),
					resource.TestCheckResourceAttr("scaleway_container.main", "privacy", containerSDK.ContainerPrivacyPrivate.String()),
					resource.TestCheckResourceAttr("scaleway_container.main", "protocol", containerSDK.ContainerProtocolH2c.String()),
				),
			},
		},
	})
}

func TestAccScalewayContainer_HTTPOption(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayContainerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_container_namespace main {}

					resource scaleway_container main {
						namespace_id = scaleway_container_namespace.main.id
						deploy = false
						http_option = "enabled"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayContainerExists(tt, "scaleway_container.main"),
					resource.TestCheckResourceAttr("scaleway_container.main", "http_option", containerSDK.ContainerHTTPOptionEnabled.String()),
				),
			},
			{
				Config: `
					resource scaleway_container_namespace main {}

					resource scaleway_container main {
						namespace_id = scaleway_container_namespace.main.id
						deploy = false
						http_option = "redirected"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayContainerExists(tt, "scaleway_container.main"),
					resource.TestCheckResourceAttr("scaleway_container.main", "http_option", containerSDK.ContainerHTTPOptionRedirected.String()),
				),
			},
			{
				Config: `
					resource scaleway_container_namespace main {}

					resource scaleway_container main {
						namespace_id = scaleway_container_namespace.main.id
						deploy = false
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayContainerExists(tt, "scaleway_container.main"),
					resource.TestCheckResourceAttr("scaleway_container.main", "http_option", containerSDK.ContainerHTTPOptionEnabled.String()),
				),
			},
		},
	})
}

func testAccCheckScalewayContainerExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource containerSDK not found: %s", n)
		}

		api, region, id, err := container.ContainerAPIWithRegionAndID(tt.GetMeta(), rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = api.GetContainer(&containerSDK.GetContainerRequest{
			ContainerID: id,
			Region:      region,
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckScalewayContainerDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_container_namespace" {
				continue
			}

			api, region, id, err := container.ContainerAPIWithRegionAndID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = api.DeleteContainer(&containerSDK.DeleteContainerRequest{
				ContainerID: id,
				Region:      region,
			})

			if err == nil {
				return fmt.Errorf("containerSDK (%s) still exists", rs.Primary.ID)
			}

			if !http_errors.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}
