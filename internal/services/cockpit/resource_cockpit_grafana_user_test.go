package cockpit_test

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	accountV3 "github.com/scaleway/scaleway-sdk-go/api/account/v3"
	cockpitSDK "github.com/scaleway/scaleway-sdk-go/api/cockpit/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/cockpit"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
	"regexp"
	"strings"
	"testing"
)

func init() {
	resource.AddTestSweepers("scaleway_cockpit_grafana_user", &resource.Sweeper{
		Name: "scaleway_cockpit_grafana_user",
		F:    testSweepCockpitGrafanaUser,
	})
}

func testSweepCockpitGrafanaUser(_ string) error {
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

			listGrafanaUsers, err := cockpitAPI.ListGrafanaUsers(&cockpitSDK.ListGrafanaUsersRequest{
				ProjectID: project.ID,
			}, scw.WithAllPages())
			if err != nil {
				if http_errors.Is404Error(err) {
					return nil
				}

				return fmt.Errorf("failed to list grafana users: %w", err)
			}

			for _, grafanaUser := range listGrafanaUsers.GrafanaUsers {
				err = cockpitAPI.DeleteGrafanaUser(&cockpitSDK.DeleteGrafanaUserRequest{
					ProjectID:     project.ID,
					GrafanaUserID: grafanaUser.ID,
				})
				if err != nil {
					if !http_errors.Is404Error(err) {
						return fmt.Errorf("failed to delete grafana user: %w", err)
					}
				}
			}
		}

		return nil
	})
}

func TestAccScalewayCockpitGrafanaUser_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	projectName := "tf_tests_cockpit_grafana_user_basic"
	grafanaTestUsername := "testuserbasic"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayCockpitGrafanaUserDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_account_project" "project" {
						name = "%[1]s"
					}

					resource scaleway_cockpit main {
						project_id = scaleway_account_project.project.id
					}

					resource scaleway_cockpit_grafana_user main {
						project_id = scaleway_cockpit.main.project_id
						login = "%[2]s"
						role = "editor"
					}
				`, projectName, grafanaTestUsername),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayCockpitGrafanaUserExists(tt, "scaleway_cockpit_grafana_user.main"),
					resource.TestCheckResourceAttrPair("scaleway_cockpit_grafana_user.main", "project_id", "scaleway_cockpit.main", "project_id"),
					resource.TestCheckResourceAttr("scaleway_cockpit_grafana_user.main", "login", grafanaTestUsername),
					resource.TestCheckResourceAttr("scaleway_cockpit_grafana_user.main", "role", "editor"),
					resource.TestCheckResourceAttrSet("scaleway_cockpit_grafana_user.main", "password"),
				),
			},
		},
	})
}

func TestAccScalewayCockpitGrafanaUser_Update(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	projectName := "tf_tests_cockpit_grafana_user_update"
	grafanaTestUsername := "testuserupdate"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayCockpitGrafanaUserDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_account_project" "project" {
						name = "%[1]s"
				  	}

					resource scaleway_cockpit main {
						project_id = scaleway_account_project.project.id
					}

					resource scaleway_cockpit_grafana_user main {
						project_id = scaleway_cockpit.main.project_id
						login = "%[2]s"
						role = "editor"
					}
				`, projectName, grafanaTestUsername),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayCockpitGrafanaUserExists(tt, "scaleway_cockpit_grafana_user.main"),
					resource.TestCheckResourceAttrPair("scaleway_cockpit_grafana_user.main", "project_id", "scaleway_cockpit.main", "project_id"),
					resource.TestCheckResourceAttr("scaleway_cockpit_grafana_user.main", "login", grafanaTestUsername),
					resource.TestCheckResourceAttr("scaleway_cockpit_grafana_user.main", "role", "editor"),
					resource.TestCheckResourceAttrSet("scaleway_cockpit_grafana_user.main", "password"),
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

					resource scaleway_cockpit_grafana_user main {
						project_id = scaleway_cockpit.main.project_id
						login = "%[2]s"
						role = "viewer"
					}
				`, projectName, grafanaTestUsername),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayCockpitGrafanaUserExists(tt, "scaleway_cockpit_grafana_user.main"),
					resource.TestCheckResourceAttrPair("scaleway_cockpit_grafana_user.main", "project_id", "scaleway_cockpit.main", "project_id"),
					resource.TestCheckResourceAttr("scaleway_cockpit_grafana_user.main", "login", grafanaTestUsername),
					resource.TestCheckResourceAttr("scaleway_cockpit_grafana_user.main", "role", "viewer"),
					resource.TestCheckResourceAttrSet("scaleway_cockpit_grafana_user.main", "password"),
				),
			},
		},
	})
}

func TestAccScalewayCockpitGrafanaUser_NonExistentCockpit(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	projectName := "tf_tests_cockpit_grafana_user_non_existent_cockpit"
	grafanaTestUsername := "testnonexistentuser"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayCockpitGrafanaUserDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_account_project" "project" {
						name = "%[1]s"
					}

					resource scaleway_cockpit_grafana_user main {
						project_id = scaleway_account_project.project.id
						login = "%[2]s"
						role = "editor"
					}
				`, projectName, grafanaTestUsername),
				ExpectError: regexp.MustCompile("not found"),
			},
		},
	})
}

func testAccCheckScalewayCockpitGrafanaUserExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource cockpit grafana user not found: %s", n)
		}

		api, projectID, grafanaUserID, err := cockpit.CockpitAPIGrafanaUserID(tt.GetMeta(), rs.Primary.ID)
		if err != nil {
			return err
		}

		res, err := api.ListGrafanaUsers(&cockpitSDK.ListGrafanaUsersRequest{
			ProjectID: projectID,
		}, scw.WithAllPages())
		if err != nil {
			return err
		}

		var grafanaUser *cockpitSDK.GrafanaUser
		for _, user := range res.GrafanaUsers {
			if user.ID == grafanaUserID {
				grafanaUser = user
				break
			}
		}

		if grafanaUser == nil {
			return fmt.Errorf("cockpit grafana user (%d) (project %s) not found", grafanaUserID, projectID)
		}

		return nil
	}
}

func testAccCheckScalewayCockpitGrafanaUserDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_cockpit_grafana_user" {
				continue
			}

			api, projectID, grafanaUserID, err := cockpit.CockpitAPIGrafanaUserID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			err = api.DeleteGrafanaUser(&cockpitSDK.DeleteGrafanaUserRequest{
				ProjectID:     projectID,
				GrafanaUserID: grafanaUserID,
			})
			if err == nil {
				return fmt.Errorf("cockpit grafana user (%s) still exists", rs.Primary.ID)
			}

			if !http_errors.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}
