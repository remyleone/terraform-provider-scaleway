package cockpit_test

import (
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccScalewayDataSourceCockpit_Basic(t *testing.T) {
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
						name = "tf_tests_datasource_cockpit_project_basic"
					}

					resource "scaleway_cockpit" "main" {
						project_id = scaleway_account_project.project.id
					}

					data "scaleway_cockpit" "selected" {
						project_id = scaleway_cockpit.main.project_id
					}

					data "scaleway_cockpit_plan" "free" {
						name = "free"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayCockpitExists(tt, "scaleway_cockpit.main"),
					testAccCheckScalewayCockpitExists(tt, "data.scaleway_cockpit.selected"),

					resource.TestCheckResourceAttrPair("data.scaleway_cockpit.selected", "plan_id", "data.scaleway_cockpit_plan.free", "id"),
					resource.TestCheckResourceAttrSet("data.scaleway_cockpit.selected", "endpoints.0.metrics_url"),
					resource.TestCheckResourceAttrSet("data.scaleway_cockpit.selected", "endpoints.0.metrics_url"),
					resource.TestCheckResourceAttrSet("data.scaleway_cockpit.selected", "endpoints.0.logs_url"),
					resource.TestCheckResourceAttrSet("data.scaleway_cockpit.selected", "endpoints.0.alertmanager_url"),
					resource.TestCheckResourceAttrSet("data.scaleway_cockpit.selected", "endpoints.0.grafana_url"),
					resource.TestCheckResourceAttrPair("data.scaleway_cockpit.selected", "project_id", "scaleway_account_project.project", "id"),
				),
			},
		},
	})
}
