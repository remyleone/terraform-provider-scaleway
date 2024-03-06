package cockpit_test

import (
	"regexp"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccScalewayDataSourceCockpitPlan_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "scaleway_cockpit_plan" "free" {
						name = "free"
					}

					data "scaleway_cockpit_plan" "premium" {
						name = "premium"
					}

					data "scaleway_cockpit_plan" "custom" {
						name = "custom"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.scaleway_cockpit_plan.free", "id"),
					resource.TestCheckResourceAttrSet("data.scaleway_cockpit_plan.premium", "id"),
					resource.TestCheckResourceAttrSet("data.scaleway_cockpit_plan.custom", "id"),
				),
			},
			{
				Config: `
					data "scaleway_cockpit_plan" "random" {
						name = "plan? there ain't no plan"
					}
				`,
				ExpectError: regexp.MustCompile("could not find plan"),
			},
		},
	})
}
