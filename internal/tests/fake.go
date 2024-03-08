package tests

import (
	"context"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	accountV3 "github.com/scaleway/scaleway-sdk-go/api/account/v3"
	iam "github.com/scaleway/scaleway-sdk-go/api/iam/v1alpha1"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/meta"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/provider"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
	"github.com/stretchr/testify/require"
)

// CreateFakeIAMManager creates a temporary project with a temporary IAM application and policy manager.
//
// The returned function is a cleanup function that should be called when to delete the project.
func CreateFakeIAMManager(tt *TestTools) (*accountV3.Project, *iam.APIKey, FakeSideProjectTerminateFunc, error) {
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

type FakeSideProjectTerminateFunc func() error

// FakeSideProjectProviders creates a new provider alias "side" with a new metaConfig that will use the
// given project and API key as default profile configuration.
//
// This is useful to test resources that need to create resources in another project.
func FakeSideProjectProviders(ctx context.Context, tt *TestTools, project *accountV3.Project, iamAPIKey *iam.APIKey) map[string]func() (*schema.Provider, error) {
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

// CreateFakeSideProject creates a temporary project with a temporary IAM application and policy.
//
// The returned function is a cleanup function that should be called when to delete the project.
func CreateFakeSideProject(tt *TestTools) (*accountV3.Project, *iam.APIKey, FakeSideProjectTerminateFunc, error) {
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
