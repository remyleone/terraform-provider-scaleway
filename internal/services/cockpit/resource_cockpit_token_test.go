package cockpit_test

import (
	"fmt"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/cockpit"
	"strings"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	accountV3 "github.com/scaleway/scaleway-sdk-go/api/account/v3"
	cockpitSDK "github.com/scaleway/scaleway-sdk-go/api/cockpit/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func init() {
	resource.AddTestSweepers("scaleway_cockpit_token", &resource.Sweeper{
		Name: "scaleway_cockpit_token",
		F:    testSweepCockpitToken,
	})
}

func testSweepCockpitToken(_ string) error {
	return tests.Sweep(func(scwClient *scw.Client) error {
		accountAPI := accountV3.NewProjectAPI(scwClient)
		cockpitAPI := cockpitSDK.NewAPI(scwClient)

		listProjects, err := accountAPI.ListProjects(&accountV3.ProjectAPIListProjectsRequest{}, scw.WithAllPages())
		if err != nil {
			return fmt.Errorf("failed to list projects: %w", err)
		}

		for _, project := range listProjects.Projects {
			if !strings.HasPrefix(project.Name, "tf_tests") {
				continue
			}

			listTokens, err := cockpitAPI.ListTokens(&cockpitSDK.ListTokensRequest{
				ProjectID: project.ID,
			}, scw.WithAllPages())
			if err != nil {
				if http_errors.Is404Error(err) {
					return nil
				}

				return fmt.Errorf("failed to list tokens: %w", err)
			}

			for _, token := range listTokens.Tokens {
				err = cockpitAPI.DeleteToken(&cockpitSDK.DeleteTokenRequest{
					TokenID: token.ID,
				})
				if err != nil {
					if !http_errors.Is404Error(err) {
						return fmt.Errorf("failed to delete token: %w", err)
					}
				}
			}
		}

		return nil
	})
}

func TestAccScalewayCockpitToken_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	projectName := "tf_tests_cockpit_token_basic"
	tokenName := projectName

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayCockpitTokenDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_account_project" "project" {
						name = "%[1]s"
				  	}

					resource scaleway_cockpit main {
						project_id = scaleway_account_project.project.id
					}

					resource scaleway_cockpit_token main {
						project_id = scaleway_cockpit.main.project_id
						name = "%[2]s"
						scopes {
							query_metrics = true
							write_logs = false
						}
					}
				`, projectName, tokenName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayCockpitTokenExists(tt, "scaleway_cockpit_token.main"),
					resource.TestCheckResourceAttrPair("scaleway_cockpit_token.main", "project_id", "scaleway_cockpit.main", "project_id"),
					resource.TestCheckResourceAttrSet("scaleway_cockpit_token.main", "secret_key"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "name", tokenName),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.query_metrics", "true"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.write_metrics", "true"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.setup_metrics_rules", "false"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.query_logs", "false"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.write_logs", "false"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.setup_logs_rules", "false"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.query_traces", "false"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.write_traces", "false"),
				),
			},
		},
	})
}

func TestAccScalewayCockpitToken_NoScopes(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	projectName := "tf_tests_cockpit_token_no_scopes"
	tokenName := "tf_tests_cockpit_token_no_scopes"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayCockpitTokenDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_account_project" "project" {
						name = "%[1]s"
				  	}

					resource scaleway_cockpit main {
						project_id = scaleway_account_project.project.id
					}

					resource scaleway_cockpit_token main {
						project_id = scaleway_cockpit.main.project_id
						name = "%[2]s"
					}
				`, projectName, tokenName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayCockpitTokenExists(tt, "scaleway_cockpit_token.main"),
					resource.TestCheckResourceAttrPair("scaleway_cockpit_token.main", "project_id", "scaleway_cockpit.main", "project_id"),
					resource.TestCheckResourceAttrSet("scaleway_cockpit_token.main", "secret_key"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "name", tokenName),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.query_metrics", "false"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.write_metrics", "true"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.setup_metrics_rules", "false"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.query_logs", "false"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.write_logs", "true"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.setup_logs_rules", "false"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.query_traces", "false"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.write_traces", "false"),
				),
			},
		},
	})
}

func TestAccScalewayCockpitToken_Update(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	projectName := "tf_tests_cockpit_token_update"
	tokenName := projectName

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayCockpitTokenDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_account_project" "project" {
						name = "%[1]s"
				  	}

					resource scaleway_cockpit main {
						project_id = scaleway_account_project.project.id
					}

					resource scaleway_cockpit_token main {
						project_id = scaleway_cockpit.main.project_id
						name = "%[2]s"
						scopes {
							query_metrics = true
							write_logs = false
						}
					}
				`, projectName, tokenName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayCockpitTokenExists(tt, "scaleway_cockpit_token.main"),
					resource.TestCheckResourceAttrPair("scaleway_cockpit_token.main", "project_id", "scaleway_cockpit.main", "project_id"),
					resource.TestCheckResourceAttrSet("scaleway_cockpit_token.main", "secret_key"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "name", tokenName),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.query_metrics", "true"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.write_metrics", "true"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.setup_metrics_rules", "false"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.query_logs", "false"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.write_logs", "false"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.setup_logs_rules", "false"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.query_traces", "false"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.write_traces", "false"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "scaleway_account_project" "project" {
						name = "%[1]s"
					}

					resource scaleway_cockpit main {
						project_id = scaleway_account_project.project.id
					}

					resource scaleway_cockpit_token main {
						project_id = scaleway_cockpit.main.project_id
						name = "%[2]s"
						scopes {
							query_metrics = true
							write_logs = true
						}
					}
				`, projectName, tokenName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayCockpitTokenExists(tt, "scaleway_cockpit_token.main"),
					resource.TestCheckResourceAttrPair("scaleway_cockpit_token.main", "project_id", "scaleway_cockpit.main", "project_id"),
					resource.TestCheckResourceAttrSet("scaleway_cockpit_token.main", "secret_key"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "name", tokenName),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.query_metrics", "true"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.write_metrics", "true"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.setup_metrics_rules", "false"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.query_logs", "false"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.write_logs", "true"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.setup_logs_rules", "false"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.query_traces", "false"),
					resource.TestCheckResourceAttr("scaleway_cockpit_token.main", "scopes.0.write_traces", "false"),
				),
			},
		},
	})
}

func testAccCheckScalewayCockpitTokenExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource cockpit token not found: %s", n)
		}

		api, err := cockpit.NewAPI(tt.GetMeta())
		if err != nil {
			return err
		}

		_, err = api.GetToken(&cockpitSDK.GetTokenRequest{
			TokenID: rs.Primary.ID,
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckScalewayCockpitTokenDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_cockpit_token" {
				continue
			}

			api, err := cockpit.NewAPI(tt.GetMeta())
			if err != nil {
				return err
			}

			err = api.DeleteToken(&cockpitSDK.DeleteTokenRequest{
				TokenID: rs.Primary.ID,
			})
			if err == nil {
				return fmt.Errorf("cockpit token (%s) still exists", rs.Primary.ID)
			}

			// Currently the API returns a 403 error when we try to delete a token that does not exist
			if !http_errors.Is404Error(err) && !http_errors.Is403Error(err) {
				return err
			}
		}

		return nil
	}
}
