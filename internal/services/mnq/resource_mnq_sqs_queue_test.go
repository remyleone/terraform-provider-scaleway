package mnq_test

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	accountSDK "github.com/scaleway/scaleway-sdk-go/api/account/v3"
	mnqSDK "github.com/scaleway/scaleway-sdk-go/api/mnq/v1beta1"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/meta"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/provider"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/account"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/mnq"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAccScalewayMNQSQSQueue_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayMNQSQSQueueDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_account_project main {
						name = "tf_tests_mnq_sqs_queue_basic"
					}

					resource scaleway_mnq_sqs main {
						project_id = scaleway_account_project.main.id
					}

					resource scaleway_mnq_sqs_credentials main {
						project_id = scaleway_mnq_sqs.main.project_id
						permissions {
							can_manage = true
						}
					}

					resource scaleway_mnq_sqs_queue main {
						project_id = scaleway_mnq_sqs.main.project_id
						name = "test-mnqSDK-sqs-queue-basic"
						sqs_endpoint = scaleway_mnq_sqs.main.endpoint
						access_key = scaleway_mnq_sqs_credentials.main.access_key
						secret_key = scaleway_mnq_sqs_credentials.main.secret_key
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayMNQSQSQueueExists(tt, "scaleway_mnq_sqs_queue.main"),
					tests.TestCheckResourceAttrUUID("scaleway_mnq_sqs_queue.main", "id"),
					resource.TestCheckResourceAttr("scaleway_mnq_sqs_queue.main", "name", "test-mnqSDK-sqs-queue-basic"),
				),
			},
			{
				Config: `
					resource scaleway_account_project main {
						name = "tf_tests_mnq_sqs_queue_basic"
					}

					resource scaleway_mnq_sqs main {
						project_id = scaleway_account_project.main.id
					}

					resource scaleway_mnq_sqs_credentials main {
						project_id = scaleway_mnq_sqs.main.project_id
						permissions {
							can_manage = true
						}
					}

					resource scaleway_mnq_sqs_queue main {
						project_id = scaleway_mnq_sqs.main.project_id
						name = "test-mnqSDK-sqs-queue-basic"
						sqs_endpoint = scaleway_mnq_sqs.main.endpoint
						access_key = scaleway_mnq_sqs_credentials.main.access_key
						secret_key = scaleway_mnq_sqs_credentials.main.secret_key

						message_max_age = 720
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayMNQSQSQueueExists(tt, "scaleway_mnq_sqs_queue.main"),
					tests.TestCheckResourceAttrUUID("scaleway_mnq_sqs_queue.main", "id"),
					resource.TestCheckResourceAttr("scaleway_mnq_sqs_queue.main", "message_max_age", "720"),
				),
			},
		},
	})
}

func TestAccScalewayMNQSQSQueue_DefaultProject(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	ctx := context.Background()

	accountAPI := accountSDK.NewProjectAPI(tt.GetMeta().GetScwClient())
	projectID := ""
	project, err := accountAPI.CreateProject(&accountSDK.ProjectAPICreateProjectRequest{
		Name: "tf_tests_mnq_sqs_queue_default_project",
	})
	require.NoError(t, err)

	projectID = project.ID

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { tests.TestAccPreCheck(t) },
		ProviderFactories: func() map[string]func() (*schema.Provider, error) {
			metaProd, err := meta.BuildMeta(ctx, &meta.MetaConfig{
				TerraformVersion: "terraform-tests",
				HttpClient:       tt.GetMeta().GetHTTPClient(),
			})
			require.NoError(t, err)

			return map[string]func() (*schema.Provider, error){
				"scaleway": func() (*schema.Provider, error) {
					return provider.Provider(&provider.ProviderConfig{Meta: metaProd})(), nil
				},
			}
		}(),
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckScalewayMNQSQSQueueDestroy(tt),
			func(_ *terraform.State) error {
				return accountAPI.DeleteProject(&accountSDK.ProjectAPIDeleteProjectRequest{
					ProjectID: projectID,
				})
			},
		),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource scaleway_mnq_sqs main {
						project_id = "%1s"
					}

					resource scaleway_mnq_sqs_credentials main {
						project_id = scaleway_mnq_sqs.main.project_id
						permissions {
							can_manage = true
						}
					}

					resource scaleway_mnq_sqs_queue main {
						project_id = scaleway_mnq_sqs.main.project_id
						name = "test-mnqSDK-sqs-queue-basic"
						access_key = scaleway_mnq_sqs_credentials.main.access_key
						secret_key = scaleway_mnq_sqs_credentials.main.secret_key
					}
				`, projectID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayMNQSQSQueueExists(tt, "scaleway_mnq_sqs_queue.main"),
					tests.TestCheckResourceAttrUUID("scaleway_mnq_sqs_queue.main", "id"),
					resource.TestCheckResourceAttr("scaleway_mnq_sqs_queue.main", "name", "test-mnqSDK-sqs-queue-basic"),
					resource.TestCheckResourceAttr("scaleway_mnq_sqs_queue.main", "project_id", projectID),
				),
			},
		},
	})
}

func testAccCheckScalewayMNQSQSQueueExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		region, _, queueName, err := mnq.DecomposeMNQID(rs.Primary.ID)
		if err != nil {
			return err
		}

		sqsClient, err := mnq.NewSQSClient(tt.GetMeta().GetHTTPClient(), region.String(), rs.Primary.Attributes["sqs_endpoint"], rs.Primary.Attributes["access_key"], rs.Primary.Attributes["secret_key"])
		if err != nil {
			return err
		}

		_, err = sqsClient.GetQueueUrl(&sqs.GetQueueUrlInput{
			QueueName: aws.String(queueName),
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckScalewayMNQSQSQueueDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_mnq_sqs_queue" {
				continue
			}

			region, projectID, queueName, err := mnq.DecomposeMNQID(rs.Primary.ID)
			if err != nil {
				return err
			}

			// Project may have been deleted, check for it first
			// Checking for Queue first may lead to an AccessDenied if project has been deleted
			accountAPI := account.AccountV3ProjectAPI(tt.GetMeta())
			_, err = accountAPI.GetProject(&accountSDK.ProjectAPIGetProjectRequest{
				ProjectID: projectID,
			})
			if err != nil {
				if http_errors.Is404Error(err) {
					return nil
				}

				return err
			}

			mnqAPI := mnqSDK.NewSqsAPI(tt.GetMeta().GetScwClient())
			sqsInfo, err := mnqAPI.GetSqsInfo(&mnqSDK.SqsAPIGetSqsInfoRequest{
				Region:    region,
				ProjectID: projectID,
			})
			if err != nil {
				return err
			}

			// SQS may be disabled for project, this means the queue does not exist
			if sqsInfo.Status == mnqSDK.SqsInfoStatusDisabled {
				return nil
			}

			sqsClient, err := mnq.NewSQSClient(tt.GetMeta().GetHTTPClient(), region.String(), rs.Primary.Attributes["sqs_endpoint"], rs.Primary.Attributes["access_key"], rs.Primary.Attributes["secret_key"])
			if err != nil {
				return err
			}

			_, err = sqsClient.GetQueueUrl(&sqs.GetQueueUrlInput{
				QueueName: aws.String(queueName),
			})
			if err != nil {
				if tfawserr.ErrCodeEquals(err, sqs.ErrCodeQueueDoesNotExist) || tfawserr.ErrCodeEquals(err, "AccessDeniedException") {
					return nil
				}

				return fmt.Errorf("failed to get queue url: %s", err)
			}

			if err == nil {
				return fmt.Errorf("mnqSDK sqs queue (%s) still exists", rs.Primary.ID)
			}

			if !http_errors.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}
