package scaleway

import (
	"fmt"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	container "github.com/scaleway/scaleway-sdk-go/api/container/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func init() {
	resource.AddTestSweepers("scaleway_container_trigger", &resource.Sweeper{
		Name: "scaleway_container_trigger",
		F:    testSweepContainerTrigger,
	})
}

func testSweepContainerTrigger(_ string) error {
	return SweepRegions((&container.API{}).Regions(), func(scwClient *scw.Client, region scw.Region) error {
		containerAPI := container.NewAPI(scwClient)
		L.Debugf("sweeper: destroying the container triggers in (%s)", region)
		listTriggers, err := containerAPI.ListTriggers(
			&container.ListTriggersRequest{
				Region: region,
			}, scw.WithAllPages())
		if err != nil {
			return fmt.Errorf("error listing trigger in (%s) in sweeper: %s", region, err)
		}

		for _, trigger := range listTriggers.Triggers {
			_, err := containerAPI.DeleteTrigger(&container.DeleteTriggerRequest{
				TriggerID: trigger.ID,
				Region:    region,
			})
			if err != nil {
				L.Debugf("sweeper: error (%s)", err)

				return fmt.Errorf("error deleting trigger in sweeper: %s", err)
			}
		}

		return nil
	})
}

func TestAccScalewayContainerTrigger_SQS(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	basicConfig := `
					resource "scaleway_account_project" "project" {
						name = "tf_tests_container_trigger_sqs"
					}

					resource scaleway_container_namespace main {
						project_id = scaleway_account_project.project.id
					}

					resource scaleway_container main {
						namespace_id = scaleway_container_namespace.main.id
					}

					resource "scaleway_mnq_sqs" "main" {
						project_id = scaleway_account_project.project.id
					}

					resource "scaleway_mnq_sqs_credentials" "main" {
						project_id = scaleway_mnq_sqs.main.project_id
					
						permissions {
							can_publish = true
							can_receive = true
							can_manage  = true
						}
					}
					
					resource "scaleway_mnq_sqs_queue" "queue" {
						project_id = scaleway_mnq_sqs.main.project_id
						name = "TestQueue"
						access_key = scaleway_mnq_sqs_credentials.main.access_key
						secret_key = scaleway_mnq_sqs_credentials.main.secret_key
					}

					resource scaleway_container_trigger main {
						container_id = scaleway_container.main.id
						name = "test-container-trigger-sqs"
						sqs {
							queue = "TestQueue"
							project_id = scaleway_mnq_sqs.main.project_id
							region = scaleway_mnq_sqs.main.region
						}
					}
				`

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayContainerTriggerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: basicConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayContainerTriggerExists(tt, "scaleway_container_trigger.main"),
					testCheckResourceAttrUUID("scaleway_container_trigger.main", "id"),
					resource.TestCheckResourceAttr("scaleway_container_trigger.main", "name", "test-container-trigger-sqs"),
					testAccCheckScalewayContainerTriggerStatusReady(tt, "scaleway_container_trigger.main"),
				),
			},
			{
				Config:   basicConfig,
				PlanOnly: true,
			},
		},
	})
}

func TestAccScalewayContainerTrigger_Nats(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	basicConfig := `
					resource scaleway_container_namespace main {
					}

					resource scaleway_container main {
						namespace_id = scaleway_container_namespace.main.id
					}

					resource "scaleway_mnq_nats_account" "main" {}

					resource scaleway_container_trigger main {
						container_id = scaleway_container.main.id
						name = "test-container-trigger-nats"
						nats {
							subject = "TestSubject"
							account_id = scaleway_mnq_nats_account.main.id
							region = scaleway_mnq_nats_account.main.region
						}
					}
				`

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayContainerTriggerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: basicConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayContainerTriggerExists(tt, "scaleway_container_trigger.main"),
					testCheckResourceAttrUUID("scaleway_container_trigger.main", "id"),
					resource.TestCheckResourceAttr("scaleway_container_trigger.main", "name", "test-container-trigger-nats"),
					testAccCheckScalewayContainerTriggerStatusReady(tt, "scaleway_container_trigger.main"),
				),
			},
			{
				Config:   basicConfig,
				PlanOnly: true,
			},
		},
	})
}

func testAccCheckScalewayContainerTriggerExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		api, region, id, err := containerAPIWithRegionAndID(tt.Meta, rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = api.GetTrigger(&container.GetTriggerRequest{
			TriggerID: id,
			Region:    region,
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckScalewayContainerTriggerStatusReady(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		api, region, id, err := containerAPIWithRegionAndID(tt.Meta, rs.Primary.ID)
		if err != nil {
			return err
		}

		trigger, err := api.GetTrigger(&container.GetTriggerRequest{
			TriggerID: id,
			Region:    region,
		})
		if err != nil {
			return err
		}

		if trigger.Status != container.TriggerStatusReady {
			return fmt.Errorf("trigger status is %s, expected ready", trigger.Status)
		}

		return nil
	}
}

func testAccCheckScalewayContainerTriggerDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_container_trigger" {
				continue
			}

			api, region, id, err := containerAPIWithRegionAndID(tt.Meta, rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = api.DeleteTrigger(&container.DeleteTriggerRequest{
				TriggerID: id,
				Region:    region,
			})

			if err == nil {
				return fmt.Errorf("container trigger (%s) still exists", rs.Primary.ID)
			}

			if !http_errors.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}
