package scaleway

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/scaleway-sdk-go/validation"
)

// Provider config can be used to provide additional config when creating provider.
type ProviderConfig struct {
	// Meta can be used to override Meta that will be used by the provider.
	// This is useful for tests.
	Meta *Meta
}

// DefaultProviderConfig return default ProviderConfig struct
func DefaultProviderConfig() *ProviderConfig {
	return &ProviderConfig{}
}

// Provider returns a terraform.ResourceProvider.
func Provider(config *ProviderConfig) plugin.ProviderFunc {
	return func() *schema.Provider {
		p := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"access_key": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "The Scaleway access key.",
				},
				"secret_key": {
					Type:         schema.TypeString,
					Optional:     true, // To allow user to use deprecated `token`.
					Description:  "The Scaleway secret Key.",
					ValidateFunc: validationUUID(),
				},
				"project_id": {
					Type:         schema.TypeString,
					Optional:     true, // To allow user to use organization instead of project
					Description:  "The Scaleway project ID.",
					ValidateFunc: validationUUID(),
				},
				"region": {
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "The Scaleway default region to use for your resources.",
					ValidateFunc: validationRegion(),
				},
				"zone": {
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "The Scaleway default zone to use for your resources.",
					ValidateFunc: validationZone(),
				},
				"api_url": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "The Scaleway API URL to use.",
				},
			},

			ResourcesMap: map[string]*schema.Resource{
				"scaleway_account_ssh_key":               resourceScalewayAccountSSKKey(),
				"scaleway_baremetal_server":              resourceScalewayBaremetalServer(),
				"scaleway_instance_ip":                   resourceScalewayInstanceIP(),
				"scaleway_instance_ip_reverse_dns":       resourceScalewayInstanceIPReverseDNS(),
				"scaleway_instance_volume":               resourceScalewayInstanceVolume(),
				"scaleway_instance_security_group":       resourceScalewayInstanceSecurityGroup(),
				"scaleway_instance_security_group_rules": resourceScalewayInstanceSecurityGroupRules(),
				"scaleway_instance_server":               resourceScalewayInstanceServer(),
				"scaleway_instance_placement_group":      resourceScalewayInstancePlacementGroup(),
				"scaleway_instance_private_nic":          resourceScalewayInstancePrivateNIC(),
				"scaleway_k8s_cluster_beta":              resourceScalewayK8SClusterBeta(),
				"scaleway_k8s_pool_beta":                 resourceScalewayK8SPoolBeta(),
				"scaleway_lb_beta":                       resourceScalewayLbBeta(),
				"scaleway_lb_ip_beta":                    resourceScalewayLbIPBeta(),
				"scaleway_lb_backend_beta":               resourceScalewayLbBackendBeta(),
				"scaleway_lb_certificate_beta":           resourceScalewayLbCertificateBeta(),
				"scaleway_lb_frontend_beta":              resourceScalewayLbFrontendBeta(),
				"scaleway_registry_namespace_beta":       resourceScalewayRegistryNamespaceBeta(),
				"scaleway_rdb_instance_beta":             resourceScalewayRdbInstanceBeta(),
				"scaleway_rdb_user_beta":                 resourceScalewayRdbUserBeta(),
				"scaleway_object_bucket":                 resourceScalewayObjectBucket(),
				"scaleway_vpc_private_network":           resourceScalewayVPCPrivateNetwork(),
			},

			DataSourcesMap: map[string]*schema.Resource{
				"scaleway_account_ssh_key":         dataSourceScalewayAccountSSHKey(),
				"scaleway_instance_security_group": dataSourceScalewayInstanceSecurityGroup(),
				"scaleway_instance_server":         dataSourceScalewayInstanceServer(),
				"scaleway_instance_image":          dataSourceScalewayInstanceImage(),
				"scaleway_instance_volume":         dataSourceScalewayInstanceVolume(),
				"scaleway_baremetal_offer":         dataSourceScalewayBaremetalOffer(),
				"scaleway_lb_ip_beta":              dataSourceScalewayLbIPBeta(),
				"scaleway_rdb_instance":            dataSourceScalewayRDBInstance(),
				"scaleway_marketplace_image_beta":  dataSourceScalewayMarketplaceImageBeta(),
				"scaleway_registry_namespace_beta": dataSourceScalewayRegistryNamespaceBeta(),
				"scaleway_registry_image_beta":     dataSourceScalewayRegistryImageBeta(),
				"scaleway_vpc_private_network":     dataSourceScalewayVPCPrivateNetwork(),
			},
		}

		p.ConfigureContextFunc = func(ctx context.Context, data *schema.ResourceData) (interface{}, diag.Diagnostics) {
			terraformVersion := p.TerraformVersion

			// If we provide meta in config use it. This is useful for tests
			if config.Meta != nil {
				return config.Meta, nil
			}

			meta, err := buildMeta(&MetaConfig{
				providerSchema:   data,
				terraformVersion: terraformVersion,
			})
			if err != nil {
				return nil, diag.FromErr(err)
			}
			return meta, nil
		}

		return p
	}
}

// Meta contains config and SDK clients used by resources.
//
// This meta value is passed into all resources.
type Meta struct {
	// scwClient is the Scaleway SDK client.
	scwClient *scw.Client
}

type MetaConfig struct {
	providerSchema   *schema.ResourceData
	terraformVersion string
	forceZone        scw.Zone
	httpClient       *http.Client
}

// providerConfigure creates the Meta object containing the SDK client.
func buildMeta(config *MetaConfig) (*Meta, error) {
	////
	// Load Profile
	////
	profile, err := loadProfile(config.providerSchema)
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

	// TODO validated profile

	////
	// Create scaleway SDK client
	////
	opts := []scw.ClientOption{
		scw.WithUserAgent(fmt.Sprintf("terraform-provider/%s terraform/%s", version, config.terraformVersion)),
		scw.WithProfile(profile),
	}

	defaultHTTPClient := &http.Client{Transport: newRetryableTransport(http.DefaultTransport)}
	if config.httpClient != nil {
		opts = append(opts, scw.WithHTTPClient(config.httpClient))
	} else {
		opts = append(opts, scw.WithHTTPClient(defaultHTTPClient))
	}

	scwClient, err := scw.NewClient(opts...)
	if err != nil {
		return nil, err
	}

	return &Meta{
		scwClient: scwClient,
	}, nil
}

func loadProfile(d *schema.ResourceData) (*scw.Profile, error) {
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
		if accessKey, exist := d.GetOk("access_key"); exist {
			providerProfile.AccessKey = scw.StringPtr(accessKey.(string))
		}
		if secretKey, exist := d.GetOk("secret_key"); exist {
			providerProfile.SecretKey = scw.StringPtr(secretKey.(string))
		}
		if projectID, exist := d.GetOk("project_id"); exist {
			providerProfile.DefaultProjectID = scw.StringPtr(projectID.(string))
		}
		if region, exist := d.GetOk("region"); exist {
			providerProfile.DefaultRegion = scw.StringPtr(region.(string))
		}
		if zone, exist := d.GetOk("zone"); exist {
			providerProfile.DefaultZone = scw.StringPtr(zone.(string))
		}
	}

	profile := scw.MergeProfiles(defaultZoneProfile, activeProfile, providerProfile, envProfile)

	// If profile have a defaultZone but no defaultRegion we set the defaultRegion
	// to the one of the defaultZone
	if profile.DefaultZone != nil && *profile.DefaultZone != "" &&
		(profile.DefaultRegion == nil || *profile.DefaultRegion == "") {
		zone := *profile.DefaultZone
		l.Debugf("guess region from %s zone", zone)
		region := zone[:len(zone)-2]
		if validation.IsRegion(region) {
			profile.DefaultRegion = scw.StringPtr(region)
		} else {
			l.Debugf("invalid guessed region '%s'", region)
		}
	}
	return profile, nil
}
