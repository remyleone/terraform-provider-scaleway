package provider_test

import (
	"context"
	"fmt"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/provider"
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

// createFakeSideProject creates a temporary project with a temporary IAM application and policy.
//
// The returned function is a cleanup function that should be called when to delete the project.
func createFakeSideProject(tt *tests.TestTools) (*accountV3.Project, *iam.APIKey, tests.FakeSideProjectTerminateFunc, error) {
	terminateFunctions := []tests.FakeSideProjectTerminateFunc{}
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

	projectAPI := accountV3.NewProjectAPI(tt.GetMeta().GetScwClient())
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

	iamAPI := iam.NewAPI(tt.GetMeta().GetScwClient())
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

// fakeSideProjectProviders creates a new provider alias "side" with a new metaConfig that will use the
// given project and API key as default profile configuration.
//
// This is useful to test resources that need to create resources in another project.
func fakeSideProjectProviders(ctx context.Context, tt *tests.TestTools, project *accountV3.Project, iamAPIKey *iam.APIKey) map[string]func() (*schema.Provider, error) {
	t := tt.T

	metaSide, err := meta.BuildMeta(ctx, &meta.MetaConfig{
		TerraformVersion:    "terraform-tests",
		HttpClient:          tt.GetMeta().GetHTTPClient(),
		ForceProjectID:      project.ID,
		ForceOrganizationID: project.OrganizationID,
		ForceAccessKey:      iamAPIKey.AccessKey,
		ForceSecretKey:      *iamAPIKey.SecretKey,
	})
	require.NoError(t, err)

	providers := map[string]func() (*schema.Provider, error){
		"side": func() (*schema.Provider, error) {
			return provider.Provider(&provider.ProviderConfig{Meta: metaSide})(), nil
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
			metaProd, err := meta.BuildMeta(ctx, &meta.MetaConfig{
				TerraformVersion: "terraform-tests",
				HttpClient:       tt.GetMeta().GetHTTPClient(),
			})
			require.NoError(t, err)

			metaDev, err := meta.BuildMeta(ctx, &meta.MetaConfig{
				TerraformVersion: "terraform-tests",
				HttpClient:       tt.GetMeta().GetHTTPClient(),
			})
			require.NoError(t, err)

			return map[string]func() (*schema.Provider, error){
				"prod": func() (*schema.Provider, error) {
					return provider.Provider(&provider.ProviderConfig{Meta: metaProd})(), nil
				},
				"dev": func() (*schema.Provider, error) {
					return provider.Provider(&provider.ProviderConfig{Meta: metaDev})(), nil
				},
			}
		}(),
		CheckDestroy: CheckIamSSHKeyDestroy(tt),
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
					CheckIamSSHKeyExists(tt, "scaleway_account_ssh_key.prod"),
					CheckIamSSHKeyExists(tt, "scaleway_account_ssh_key.dev"),
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
			metaProd, err := meta.BuildMeta(ctx, &meta.MetaConfig{
				TerraformVersion: "terraform-tests",
				ForceZone:        scw.ZoneFrPar2,
				HttpClient:       tt.GetMeta().GetHTTPClient(),
			})
			require.NoError(t, err)

			metaDev, err := meta.BuildMeta(ctx, &meta.MetaConfig{
				TerraformVersion: "terraform-tests",
				ForceZone:        scw.ZoneFrPar1,
				HttpClient:       tt.GetMeta().GetHTTPClient(),
			})
			require.NoError(t, err)

			return map[string]func() (*schema.Provider, error){
				"prod": func() (*schema.Provider, error) {
					return provider.Provider(&provider.ProviderConfig{Meta: metaProd})(), nil
				},
				"dev": func() (*schema.Provider, error) {
					return provider.Provider(&provider.ProviderConfig{Meta: metaDev})(), nil
				},
			}
		}(),
		CheckDestroy: CheckIamSSHKeyDestroy(tt),
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
					CheckInstanceIPExists(tt, "scaleway_instance_ip.prod"),
					CheckInstanceIPExists(tt, "scaleway_instance_ip.dev"),
					resource.TestCheckResourceAttr("scaleway_instance_ip.prod", "zone", "fr-par-2"),
					resource.TestCheckResourceAttr("scaleway_instance_ip.dev", "zone", "fr-par-1"),
				),
			},
		},
	})
}
