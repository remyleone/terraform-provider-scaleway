package provider_test

import (
	"context"
	"fmt"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/meta"

	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	accountV3 "github.com/scaleway/scaleway-sdk-go/api/account/v3"
	iam "github.com/scaleway/scaleway-sdk-go/api/iam/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
	"github.com/stretchr/testify/require"
)

type FakeSideProjectTerminateFunc func() error

// createFakeSideProject creates a temporary project with a temporary IAM application and policy.
//
// The returned function is a cleanup function that should be called when to delete the project.
func createFakeSideProject(tt *tests.TestTools) (*accountV3.Project, *iam.APIKey, FakeSideProjectTerminateFunc, error) {
	terminateFunctions := []FakeSideProjectTerminateFunc{}
	terminate := func() error {
		for i := len(terminateFunctions) - 1; i >= 0; i-- {
			err := terminateFunctions[i]()
			if err != nil {
				return err
			}
		}

		return nil
	}

	projectName := sdkacctest.RandomWithPrefix("test-acc-scaleway-project")
	iamApplicationName := sdkacctest.RandomWithPrefix("test-acc-scaleway-iam-app")
	iamPolicyName := sdkacctest.RandomWithPrefix("test-acc-scaleway-iam-policy")

	projectAPI := accountV3.NewProjectAPI(tt.meta.GetScwClient())
	project, err := projectAPI.CreateProject(&accountV3.ProjectAPICreateProjectRequest{
		Name: projectName,
	})
	if err != nil {
		if err := terminate(); err != nil {
			return nil, nil, nil, err
		}

		return nil, nil, nil, err
	}
	terminateFunctions = append(terminateFunctions, func() error {
		return projectAPI.DeleteProject(&accountV3.ProjectAPIDeleteProjectRequest{
			ProjectID: project.ID,
		})
	})

	iamAPI := iam.NewAPI(tt.meta.GetScwClient())
	iamApplication, err := iamAPI.CreateApplication(&iam.CreateApplicationRequest{
		Name: iamApplicationName,
	})
	if err != nil {
		if err := terminate(); err != nil {
			return nil, nil, nil, err
		}

		return nil, nil, nil, err
	}
	terminateFunctions = append(terminateFunctions, func() error {
		return iamAPI.DeleteApplication(&iam.DeleteApplicationRequest{
			ApplicationID: iamApplication.ID,
		})
	})

	iamPolicy, err := iamAPI.CreatePolicy(&iam.CreatePolicyRequest{
		Name:          iamPolicyName,
		ApplicationID: types.ExpandStringPtr(iamApplication.ID),
		Rules: []*iam.RuleSpecs{
			{
				ProjectIDs:         &[]string{project.ID},
				PermissionSetNames: &[]string{"ObjectStorageReadOnly", "ObjectStorageObjectsRead", "ObjectStorageBucketsRead"},
			},
		},
	})
	if err != nil {
		if err := terminate(); err != nil {
			return nil, nil, nil, err
		}

		return nil, nil, nil, err
	}
	terminateFunctions = append(terminateFunctions, func() error {
		return iamAPI.DeletePolicy(&iam.DeletePolicyRequest{
			PolicyID: iamPolicy.ID,
		})
	})

	iamAPIKey, err := iamAPI.CreateAPIKey(&iam.CreateAPIKeyRequest{
		ApplicationID:    types.ExpandStringPtr(iamApplication.ID),
		DefaultProjectID: &project.ID,
	})
	if err != nil {
		if err := terminate(); err != nil {
			return nil, nil, nil, err
		}

		return nil, nil, nil, err
	}
	terminateFunctions = append(terminateFunctions, func() error {
		return iamAPI.DeleteAPIKey(&iam.DeleteAPIKeyRequest{
			AccessKey: iamAPIKey.AccessKey,
		})
	})

	return project, iamAPIKey, terminate, nil
}

// createFakeIAMManager creates a temporary project with a temporary IAM application and policy manager.
//
// The returned function is a cleanup function that should be called when to delete the project.
func createFakeIAMManager(tt *tests.TestTools) (*accountV3.Project, *iam.APIKey, FakeSideProjectTerminateFunc, error) {
	terminateFunctions := []FakeSideProjectTerminateFunc{}
	terminate := func() error {
		for i := len(terminateFunctions) - 1; i >= 0; i-- {
			err := terminateFunctions[i]()
			if err != nil {
				return err
			}
		}

		return nil
	}

	projectName := sdkacctest.RandomWithPrefix("test-acc-scaleway-project")
	iamApplicationName := sdkacctest.RandomWithPrefix("test-acc-scaleway-iam-app")
	iamPolicyName := sdkacctest.RandomWithPrefix("test-acc-scaleway-iam-policy")

	projectAPI := accountV3.NewProjectAPI(tt.meta.GetScwClient())
	project, err := projectAPI.CreateProject(&accountV3.ProjectAPICreateProjectRequest{
		Name: projectName,
	})
	if err != nil {
		if err := terminate(); err != nil {
			return nil, nil, nil, err
		}

		return nil, nil, nil, err
	}
	terminateFunctions = append(terminateFunctions, func() error {
		return projectAPI.DeleteProject(&accountV3.ProjectAPIDeleteProjectRequest{
			ProjectID: project.ID,
		})
	})

	iamAPI := iam.NewAPI(tt.meta.GetScwClient())
	iamApplication, err := iamAPI.CreateApplication(&iam.CreateApplicationRequest{
		Name: iamApplicationName,
	})
	if err != nil {
		if err := terminate(); err != nil {
			return nil, nil, nil, err
		}

		return nil, nil, nil, err
	}
	terminateFunctions = append(terminateFunctions, func() error {
		return iamAPI.DeleteApplication(&iam.DeleteApplicationRequest{
			ApplicationID: iamApplication.ID,
		})
	})

	iamPolicy, err := iamAPI.CreatePolicy(&iam.CreatePolicyRequest{
		Name:          iamPolicyName,
		ApplicationID: types.ExpandStringPtr(iamApplication.ID),
		Rules: []*iam.RuleSpecs{
			{
				OrganizationID:     &project.OrganizationID,
				PermissionSetNames: &[]string{"IAMManager"},
			},
		},
	})
	if err != nil {
		if err := terminate(); err != nil {
			return nil, nil, nil, err
		}

		return nil, nil, nil, err
	}
	terminateFunctions = append(terminateFunctions, func() error {
		return iamAPI.DeletePolicy(&iam.DeletePolicyRequest{
			PolicyID: iamPolicy.ID,
		})
	})

	iamAPIKey, err := iamAPI.CreateAPIKey(&iam.CreateAPIKeyRequest{
		ApplicationID:    types.ExpandStringPtr(iamApplication.ID),
		DefaultProjectID: &project.ID,
	})
	if err != nil {
		if err := terminate(); err != nil {
			return nil, nil, nil, err
		}

		return nil, nil, nil, err
	}
	terminateFunctions = append(terminateFunctions, func() error {
		return iamAPI.DeleteAPIKey(&iam.DeleteAPIKeyRequest{
			AccessKey: iamAPIKey.AccessKey,
		})
	})

	return project, iamAPIKey, terminate, nil
}

// fakeSideProjectProviders creates a new provider alias "side" with a new metaConfig that will use the
// given project and API key as default profile configuration.
//
// This is useful to test resources that need to create resources in another project.
func fakeSideProjectProviders(ctx context.Context, tt *tests.TestTools, project *accountV3.Project, iamAPIKey *iam.APIKey) map[string]func() (*schema.Provider, error) {
	t := tt.T

	metaSide, err := buildMeta(ctx, &meta.metaConfig{
		terraformVersion:    "terraform-tests",
		httpClient:          tt.Meta.httpClient,
		forceProjectID:      project.ID,
		forceOrganizationID: project.OrganizationID,
		forceAccessKey:      iamAPIKey.AccessKey,
		forceSecretKey:      *iamAPIKey.SecretKey,
	})
	require.NoError(t, err)

	providers := map[string]func() (*schema.Provider, error){
		"side": func() (*schema.Provider, error) {
			return Provider(&ProviderConfig{Meta: metaSide})(), nil
		},
	}

	for k, v := range tt.ProviderFactories {
		providers[k] = v
	}

	return providers
}

func TestAccScalewayProvider_SSHKeys(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	SSHKeyName := "TestAccScalewayProvider_SSHKeys"
	SSHKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEEYrzDOZmhItdKaDAEqJQ4ORS2GyBMtBozYsK5kiXXX opensource@scaleway.com"

	ctx := context.Background()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { tests.TestAccPreCheck(t) },
		ProviderFactories: func() map[string]func() (*schema.Provider, error) {
			metaProd, err := buildMeta(ctx, &meta.metaConfig{
				terraformVersion: "terraform-tests",
				httpClient:       tt.Meta.httpClient,
			})
			require.NoError(t, err)

			metaDev, err := buildMeta(ctx, &meta.metaConfig{
				terraformVersion: "terraform-tests",
				httpClient:       tt.Meta.httpClient,
			})
			require.NoError(t, err)

			return map[string]func() (*schema.Provider, error){
				"prod": func() (*schema.Provider, error) {
					return Provider(&ProviderConfig{Meta: metaProd})(), nil
				},
				"dev": func() (*schema.Provider, error) {
					return Provider(&ProviderConfig{Meta: metaDev})(), nil
				},
			}
		}(),
		CheckDestroy: testAccCheckScalewayIamSSHKeyDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_account_ssh_key" "prod" {
						provider   = "prod"
						name 	   = "%[1]s"
						public_key = "%[2]s"
					}

					resource "scaleway_account_ssh_key" "dev" {
						provider   = "dev"
						name 	   = "%[1]s"
						public_key = "%[2]s"
					}
				`, SSHKeyName, SSHKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayIamSSHKeyExists(tt, "scaleway_account_ssh_key.prod"),
					testAccCheckScalewayIamSSHKeyExists(tt, "scaleway_account_ssh_key.dev"),
				),
			},
		},
	})
}

func TestAccScalewayProvider_InstanceIPZones(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	ctx := context.Background()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { tests.TestAccPreCheck(t) },
		ProviderFactories: func() map[string]func() (*schema.Provider, error) {
			metaProd, err := buildMeta(ctx, &meta.metaConfig{
				terraformVersion: "terraform-tests",
				forceZone:        scw.ZoneFrPar2,
				httpClient:       tt.Meta.httpClient,
			})
			require.NoError(t, err)

			metaDev, err := buildMeta(ctx, &meta.metaConfig{
				terraformVersion: "terraform-tests",
				forceZone:        scw.ZoneFrPar1,
				httpClient:       tt.Meta.httpClient,
			})
			require.NoError(t, err)

			return map[string]func() (*schema.Provider, error){
				"prod": func() (*schema.Provider, error) {
					return Provider(&ProviderConfig{Meta: metaProd})(), nil
				},
				"dev": func() (*schema.Provider, error) {
					return Provider(&ProviderConfig{Meta: metaDev})(), nil
				},
			}
		}(),
		CheckDestroy: testAccCheckScalewayIamSSHKeyDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_instance_ip dev {
					  provider = "dev"
					}

					resource scaleway_instance_ip prod {
					  provider = "prod"
					}
`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceIPExists(tt, "scaleway_instance_ip.prod"),
					testAccCheckScalewayInstanceIPExists(tt, "scaleway_instance_ip.dev"),
					resource.TestCheckResourceAttr("scaleway_instance_ip.prod", "zone", "fr-par-2"),
					resource.TestCheckResourceAttr("scaleway_instance_ip.dev", "zone", "fr-par-1"),
				),
			},
		},
	})
}
