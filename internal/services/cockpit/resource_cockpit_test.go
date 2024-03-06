package cockpit_test

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	accountV3 "github.com/scaleway/scaleway-sdk-go/api/account/v3"
	cockpit "github.com/scaleway/scaleway-sdk-go/api/cockpit/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
	"strings"
	"testing"
)

func init() {
	resource.AddTestSweepers("scaleway_cockpit", &resource.Sweeper{
		Name: "scaleway_cockpit",
		F:    testSweepCockpit,
	})
}

func testSweepCockpit(_ string) error {
	return tests.Sweep(func(scwClient *scw.Client) error {
		accountAPI := accountV3.NewProjectAPI(scwClient)
		cockpitAPI := cockpit.NewAPI(scwClient)

		listProjects, err := accountAPI.ListProjects(&accountV3.ProjectAPIListProjectsRequest{}, scw.WithAllPages())
		if err != nil {
			return fmt.Errorf("failed to list projects: %w", err)
		}

		for _, project := range listProjects.Projects {
			if !strings.HasPrefix(project.Name, "tf_tests") {
				continue
			}

			_, err = cockpitAPI.WaitForCockpit(&cockpit.WaitForCockpitRequest{
				ProjectID: project.ID,
				Timeout:   scw.TimeDurationPtr(defaultCockpitTimeout),
			})
			if err != nil {
				if !http_errors.Is404Error(err) {
					return fmt.Errorf("failed to deactivate cockpit: %w", err)
				}
			}

			_, err = cockpitAPI.DeactivateCockpit(&cockpit.DeactivateCockpitRequest{
				ProjectID: project.ID,
			})
			if err != nil {
				if !http_errors.Is404Error(err) {
					return fmt.Errorf("failed to deactivate cockpit: %w", err)
				}
			}
		}

		return nil
	})
}

func TestAccScalewayCockpit_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayCockpitDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_account_project" "project" {
						name = "tf_tests_cockpit_project_basic"
				  	}

					resource scaleway_cockpit main {
						project_id = scaleway_account_project.project.id
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayCockpitExists(tt, "scaleway_cockpit.main"),
					resource.TestCheckResourceAttrSet("scaleway_cockpit.main", "plan_id"),
					resource.TestCheckResourceAttrSet("scaleway_cockpit.main", "endpoints.0.metrics_url"),
					resource.TestCheckResourceAttrSet("scaleway_cockpit.main", "endpoints.0.logs_url"),
					resource.TestCheckResourceAttrSet("scaleway_cockpit.main", "endpoints.0.alertmanager_url"),
					resource.TestCheckResourceAttrSet("scaleway_cockpit.main", "endpoints.0.grafana_url"),
					resource.TestCheckResourceAttrSet("scaleway_cockpit.main", "endpoints.0.traces_url"),
					resource.TestCheckResourceAttr("scaleway_cockpit.main", "push_url.0.push_logs_url", "https://logs.cockpit.fr-par.scw.cloud/loki/api/v1/push"),
					resource.TestCheckResourceAttr("scaleway_cockpit.main", "push_url.0.push_metrics_url", "https://metrics.cockpit.fr-par.scw.cloud/api/v1/push"),

					resource.TestCheckResourceAttrPair("scaleway_cockpit.main", "project_id", "scaleway_account_project.project", "id"),
				),
			},
			{
				Config: `
					resource "scaleway_account_project" "project" {
						name = "tf_tests_cockpit_project_basic"
				  	}

					data "scaleway_cockpit_plan" "premium" {
						name = "premium"
					}

					resource "scaleway_cockpit" "main" {
						project_id = scaleway_account_project.project.id
						plan       = data.scaleway_cockpit_plan.premium.id
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayCockpitExists(tt, "scaleway_cockpit.main"),
					resource.TestCheckResourceAttrSet("scaleway_cockpit.main", "plan_id"),
				),
			},
		},
	})
}

func TestAccScalewayCockpit_PremiumPlanByID(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayCockpitDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_account_project" "project" {
						name = "tf_tests_cockpit_project_premium"
				  	}

					data "scaleway_cockpit_plan" "premium" {
						name = "premium"
					}

					resource scaleway_cockpit main {
						project_id = scaleway_account_project.project.id
						plan       = data.scaleway_cockpit_plan.premium.id
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayCockpitExists(tt, "scaleway_cockpit.main"),
				),
			},
			{
				Config: `
					resource "scaleway_account_project" "project" {
						name = "tf_tests_cockpit_project_premium"
				  	}

					data "scaleway_cockpit_plan" "free" {
						name = "free"
					}

					resource scaleway_cockpit main {
						project_id = scaleway_account_project.project.id
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayCockpitExists(tt, "scaleway_cockpit.main"),
					resource.TestCheckResourceAttrPair("scaleway_cockpit.main", "plan_id", "data.scaleway_cockpit_plan.free", "id"),
				),
			},
		},
	})
}

func TestAccScalewayCockpit_PremiumPlanByName(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayCockpitDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_account_project" "project" {
						name = "tf_tests_cockpit_project_premium"
				  	}

					data "scaleway_cockpit_plan" "premium" {
						name = "premium"
					}

					resource "scaleway_cockpit" "main" {
						project_id = scaleway_account_project.project.id
						plan       = "premium"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayCockpitExists(tt, "scaleway_cockpit.main"),
					resource.TestCheckResourceAttrPair("scaleway_cockpit.main", "plan_id", "data.scaleway_cockpit_plan.premium", "id"),
				),
			},
		},
	})
}

func testAccCheckScalewayCockpitExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource cockpit not found: %s", n)
		}

		api, err := cockpitAPI(tt.Meta)
		if err != nil {
			return err
		}

		_, err = api.GetCockpit(&cockpit.GetCockpitRequest{
			ProjectID: rs.Primary.ID,
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckScalewayCockpitDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_cockpit" {
				continue
			}

			api, err := cockpitAPI(tt.Meta)
			if err != nil {
				return err
			}

			_, err = api.DeactivateCockpit(&cockpit.DeactivateCockpitRequest{
				ProjectID: rs.Primary.ID,
			})
			if err == nil {
				return fmt.Errorf("cockpit (%s) still exists", rs.Primary.ID)
			}

			if !http_errors.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}
