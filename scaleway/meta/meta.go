package meta

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/transport"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/version"
	"net/http"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

type MetaConfig struct {
	ProviderSchema      *schema.ResourceData
	TerraformVersion    string
	forceZone           scw.Zone
	forceProjectID      string
	forceOrganizationID string
	forceAccessKey      string
	forceSecretKey      string
	httpClient          *http.Client
}

// Meta contains config and SDK clients used by resources.
//
// This meta value is passed into all resources.
type Meta struct {
	// scwClient is the Scaleway SDK client.
	scwClient *scw.Client
	// httpClient can be either a regular http.Client used to make real HTTP requests
	// or it can be a http.Client used to record and replay cassettes which is useful
	// to replay recorded interactions with APIs locally
	httpClient *http.Client
}

func (m Meta) GetScwClient() *scw.Client {
	return m.scwClient
}

func (m Meta) GetHTTPClient() *http.Client { return m.httpClient }

// BuildMeta creates the Meta object containing the SDK client.
func BuildMeta(ctx context.Context, config *MetaConfig) (*Meta, error) {
	////
	// Load Profile
	////
	profile, err := loadProfile(ctx, config.ProviderSchema)
	if err != nil {
		return nil, err
	}
	if config.forceZone != "" {
		region, err := config.forceZone.Region()
		if err != nil {
			return nil, err
		}
		profile.DefaultRegion = scw.StringPtr(region.String())
		profile.DefaultZone = scw.StringPtr(config.forceZone.String())
	}
	if config.forceProjectID != "" {
		profile.DefaultProjectID = scw.StringPtr(config.forceProjectID)
	}
	if config.forceOrganizationID != "" {
		profile.DefaultOrganizationID = scw.StringPtr(config.forceOrganizationID)
	}
	if config.forceAccessKey != "" {
		profile.AccessKey = scw.StringPtr(config.forceAccessKey)
	}
	if config.forceSecretKey != "" {
		profile.SecretKey = scw.StringPtr(config.forceSecretKey)
	}

	// TODO validated profile

	////
	// Create scaleway SDK client
	////
	opts := []scw.ClientOption{
		scw.WithUserAgent(customizeUserAgent(version.Version, config.TerraformVersion)),
		scw.WithProfile(profile),
	}

	httpClient := &http.Client{Transport: transport.NewRetryableTransport(http.DefaultTransport)}
	if config.httpClient != nil {
		httpClient = config.httpClient
	}
	opts = append(opts, scw.WithHTTPClient(httpClient))

	scwClient, err := scw.NewClient(opts...)
	if err != nil {
		return nil, err
	}

	return &Meta{
		scwClient:  scwClient,
		httpClient: httpClient,
	}, nil
}

//gocyclo:ignore
func loadProfile(ctx context.Context, d *schema.ResourceData) (*scw.Profile, error) {
	config, err := scw.LoadConfig()
	// If the config file do not exist, don't return an error as we may find config in ENV or flags.
	if _, isNotFoundError := err.(*scw.ConfigFileNotFoundError); isNotFoundError {
		config = &scw.Config{}
	} else if err != nil {
		return nil, err
	}

	// By default we set default zone and region to fr-par
	defaultZoneProfile := &scw.Profile{
		DefaultRegion: scw.StringPtr(scw.RegionFrPar.String()),
		DefaultZone:   scw.StringPtr(scw.ZoneFrPar1.String()),
	}

	activeProfile, err := config.GetActiveProfile()
	if err != nil {
		return nil, err
	}
	envProfile := scw.LoadEnvProfile()

	providerProfile := &scw.Profile{}
	if d != nil {
		if profileName, exist := d.GetOk("profile"); exist {
			profileFromConfig, err := config.GetProfile(profileName.(string))
			if err == nil {
				providerProfile = profileFromConfig
			}
		}
		if accessKey, exist := d.GetOk("access_key"); exist {
			providerProfile.AccessKey = scw.StringPtr(accessKey.(string))
		}
		if secretKey, exist := d.GetOk("secret_key"); exist {
			providerProfile.SecretKey = scw.StringPtr(secretKey.(string))
		}
		if projectID, exist := d.GetOk("project_id"); exist {
			providerProfile.DefaultProjectID = scw.StringPtr(projectID.(string))
		}
		if orgID, exist := d.GetOk("organization_id"); exist {
			providerProfile.DefaultOrganizationID = scw.StringPtr(orgID.(string))
		}
		if region, exist := d.GetOk("region"); exist {
			providerProfile.DefaultRegion = scw.StringPtr(region.(string))
		}
		if zone, exist := d.GetOk("zone"); exist {
			providerProfile.DefaultZone = scw.StringPtr(zone.(string))
		}
		if apiURL, exist := d.GetOk("api_url"); exist {
			providerProfile.APIURL = scw.StringPtr(apiURL.(string))
		}
	}

	profile := scw.MergeProfiles(defaultZoneProfile, activeProfile, providerProfile, envProfile)

	// If profile have a defaultZone but no defaultRegion we set the defaultRegion
	// to the one of the defaultZone
	if profile.DefaultZone != nil && *profile.DefaultZone != "" &&
		(profile.DefaultRegion == nil || *profile.DefaultRegion == "") {
		zone := scw.Zone(*profile.DefaultZone)
		tflog.Debug(ctx, fmt.Sprintf("guess region from %s zone", zone))
		region, err := zone.Region()
		if err == nil {
			profile.DefaultRegion = scw.StringPtr(region.String())
		} else {
			tflog.Debug(ctx, "cannot guess region: "+err.Error())
		}
	}
	return profile, nil
}

const appendUserAgentEnvVar = "TF_APPEND_USER_AGENT"

func customizeUserAgent(providerVersion string, terraformVersion string) string {
	userAgent := fmt.Sprintf("terraform-provider/%s terraform/%s", providerVersion, terraformVersion)

	if appendUserAgent := os.Getenv(appendUserAgentEnvVar); appendUserAgent != "" {
		userAgent += " " + appendUserAgent
	}

	return userAgent
}
