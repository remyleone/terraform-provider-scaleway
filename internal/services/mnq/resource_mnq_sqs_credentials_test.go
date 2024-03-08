package mnq_test

import (
	"fmt"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/logging"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/mnq"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	mnqSDK "github.com/scaleway/scaleway-sdk-go/api/mnq/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func init() {
	resource.AddTestSweepers("scaleway_mnq_sqs_credentials", &resource.Sweeper{
		Name: "scaleway_mnq_sqs_credentials",
		F:    testSweepMNQSQSCredentials,
	})
}

func testSweepMNQSQSCredentials(_ string) error {
	return tests.SweepRegions((&mnqSDK.SqsAPI{}).Regions(), func(scwClient *scw.Client, region scw.Region) error {
		mnqAPI := mnqSDK.NewSqsAPI(scwClient)
		logging.L.Debugf("sweeper: destroying the mnqSDK sqs credentials in (%s)", region)
		listSqsCredentials, err := mnqAPI.ListSqsCredentials(
			&mnqSDK.SqsAPIListSqsCredentialsRequest{
				Region: region,
			}, scw.WithAllPages())
		if err != nil {
			return fmt.Errorf("error listing sqs credentials in (%s) in sweeper: %s", region, err)
		}

		for _, credentials := range listSqsCredentials.SqsCredentials {
			err := mnqAPI.DeleteSqsCredentials(&mnqSDK.SqsAPIDeleteSqsCredentialsRequest{
				SqsCredentialsID: credentials.ID,
				Region:           region,
			})
			if err != nil {
				logging.L.Debugf("sweeper: error (%s)", err)

				return fmt.Errorf("error deleting sqs credentials in sweeper: %s", err)
			}
		}

		return nil
	})
}

func TestAccScalewayMNQSQSCredentials_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayMNQSQSCredentialsDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_account_project main {
						name = "tf_tests_mnq_sqs_credentials_basic"
					}

					resource scaleway_mnq_sqs main {
						project_id = scaleway_account_project.main.id
					}

					resource scaleway_mnq_sqs_credentials main {
						project_id = scaleway_mnq_sqs.main.project_id
						name = "test-mnqSDK-sqs-credentials-basic"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayMNQSQSCredentialsExists(tt, "scaleway_mnq_sqs_credentials.main"),
					tests.TestCheckResourceAttrUUID("scaleway_mnq_sqs_credentials.main", "id"),
					resource.TestCheckResourceAttr("scaleway_mnq_sqs_credentials.main", "name", "test-mnqSDK-sqs-credentials-basic"),
					resource.TestCheckResourceAttrSet("scaleway_mnq_sqs_credentials.main", "access_key"),
					resource.TestCheckResourceAttrSet("scaleway_mnq_sqs_credentials.main", "secret_key"),
				),
			},
			{
				Config: `
					resource scaleway_account_project main {
						name = "tf_tests_mnq_sqs_credentials_basic"
					}

					resource scaleway_mnq_sqs main {
						project_id = scaleway_account_project.main.id
					}

					resource scaleway_mnq_sqs_credentials main {
						project_id = scaleway_mnq_sqs.main.project_id
						name = "test-mnqSDK-sqs-credentials-basic"
						permissions {
							can_manage = true
							can_receive = false
							can_publish = true
						}
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayMNQSQSCredentialsExists(tt, "scaleway_mnq_sqs_credentials.main"),
					resource.TestCheckResourceAttr("scaleway_mnq_sqs_credentials.main", "permissions.0.can_manage", "true"),
					resource.TestCheckResourceAttr("scaleway_mnq_sqs_credentials.main", "permissions.0.can_receive", "false"),
					resource.TestCheckResourceAttr("scaleway_mnq_sqs_credentials.main", "permissions.0.can_publish", "true"),
				),
			},
			{
				Config: `
					resource scaleway_account_project main {
						name = "tf_tests_mnq_sqs_credentials_basic"
					}

					resource scaleway_mnq_sqs main {
						project_id = scaleway_account_project.main.id
					}

					resource scaleway_mnq_sqs_credentials main {
						project_id = scaleway_mnq_sqs.main.project_id
						name = "test-mnqSDK-sqs-credentials-basic"
						permissions {
							can_manage = false
							can_receive = true
							can_publish = false
						}
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayMNQSQSCredentialsExists(tt, "scaleway_mnq_sqs_credentials.main"),
					resource.TestCheckResourceAttr("scaleway_mnq_sqs_credentials.main", "permissions.0.can_manage", "false"),
					resource.TestCheckResourceAttr("scaleway_mnq_sqs_credentials.main", "permissions.0.can_receive", "true"),
					resource.TestCheckResourceAttr("scaleway_mnq_sqs_credentials.main", "permissions.0.can_publish", "false"),
				),
			},
		},
	})
}

func testAccCheckScalewayMNQSQSCredentialsExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		api, region, id, err := mnq.MnqSQSAPIWithRegionAndID(tt.GetMeta(), rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = api.GetSqsCredentials(&mnqSDK.SqsAPIGetSqsCredentialsRequest{
			SqsCredentialsID: id,
			Region:           region,
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckScalewayMNQSQSCredentialsDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_mnq_sqs_credentials" {
				continue
			}

			api, region, id, err := mnq.MnqSQSAPIWithRegionAndID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			err = api.DeleteSqsCredentials(&mnqSDK.SqsAPIDeleteSqsCredentialsRequest{
				SqsCredentialsID: id,
				Region:           region,
			})

			if err == nil {
				return fmt.Errorf("mnqSDK sqs credentials (%s) still exists", rs.Primary.ID)
			}

			if !http_errors.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}
