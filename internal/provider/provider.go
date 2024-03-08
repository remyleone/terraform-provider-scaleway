package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/az"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	meta2 "github.com/scaleway/terraform-provider-scaleway/v2/internal/meta"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/account"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/applesilicon"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/baremetal"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/billing"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/block"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/cockpit"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/container"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/documentdb"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/domain"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/fip"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/function"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/iam"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/instance"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/iot"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/ipam"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/jobs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/k8s"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/lb"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/marketplace"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/mnq"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/object"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/rdb"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/redis"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/registry"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/sdb"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/secret"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/tem"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/vpc"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/vpcgw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/webhosting"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/verify"
	"os"
)

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
					ValidateFunc: verify.UUID(),
				},
				"profile": {
					Type:        schema.TypeString,
					Optional:    true, // To allow user to use `access_key`, `secret_key`, `project_id`...
					Description: "The Scaleway profile to use.",
				},
				"project_id": {
					Type:         schema.TypeString,
					Optional:     true, // To allow user to use organization instead of project
					Description:  "The Scaleway project ID.",
					ValidateFunc: verify.UUID(),
				},
				"organization_id": {
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "The Scaleway organization ID.",
					ValidateFunc: verify.UUID(),
				},
				"region": locality.RegionalSchema(),
				"zone":   locality.ZonalSchema(),
				"api_url": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "The Scaleway API URL to use.",
				},
			},

			ResourcesMap: map[string]*schema.Resource{
				"scaleway_account_project":                     account.ResourceScalewayAccountProject(),
				"scaleway_account_ssh_key":                     account.ResourceScalewayAccountSSKKey(),
				"scaleway_apple_silicon_server":                applesilicon.ResourceScalewayAppleSiliconServer(),
				"scaleway_baremetal_server":                    baremetal.ResourceScalewayBaremetalServer(),
				"scaleway_block_volume":                        block.ResourceScalewayBlockVolume(),
				"scaleway_block_snapshot":                      block.ResourceScalewayBlockSnapshot(),
				"scaleway_cockpit":                             cockpit.ResourceScalewayCockpit(),
				"scaleway_cockpit_token":                       cockpit.ResourceScalewayCockpitToken(),
				"scaleway_cockpit_grafana_user":                cockpit.ResourceScalewayCockpitGrafanaUser(),
				"scaleway_container_namespace":                 container.ResourceScalewayContainerNamespace(),
				"scaleway_container_cron":                      container.ResourceScalewayContainerCron(),
				"scaleway_container_domain":                    container.ResourceScalewayContainerDomain(),
				"scaleway_container_trigger":                   container.ResourceScalewayContainerTrigger(),
				"scaleway_documentdb_instance":                 documentdb.ResourceScalewayDocumentDBInstance(),
				"scaleway_documentdb_database":                 documentdb.ResourceScalewayDocumentDBDatabase(),
				"scaleway_documentdb_private_network_endpoint": documentdb.ResourceScalewayDocumentDBInstancePrivateNetworkEndpoint(),
				"scaleway_documentdb_user":                     documentdb.ResourceScalewayDocumentDBUser(),
				"scaleway_documentdb_privilege":                documentdb.ResourceScalewayDocumentDBPrivilege(),
				"scaleway_documentdb_read_replica":             documentdb.ResourceScalewayDocumentDBReadReplica(),
				"scaleway_domain_record":                       domain.ResourceScalewayDomainRecord(),
				"scaleway_domain_zone":                         domain.ResourceScalewayDomainZone(),
				"scaleway_flexible_ip":                         fip.ResourceScalewayFlexibleIP(),
				"scaleway_flexible_ip_mac_address":             fip.ResourceScalewayFlexibleIPMACAddress(),
				"scaleway_function":                            function.ResourceScalewayFunction(),
				"scaleway_function_cron":                       function.ResourceScalewayFunctionCron(),
				"scaleway_function_domain":                     function.ResourceScalewayFunctionDomain(),
				"scaleway_function_namespace":                  function.ResourceScalewayFunctionNamespace(),
				"scaleway_function_token":                      function.ResourceScalewayFunctionToken(),
				"scaleway_function_trigger":                    function.ResourceScalewayFunctionTrigger(),
				"scaleway_iam_api_key":                         iam.ResourceScalewayIamAPIKey(),
				"scaleway_iam_application":                     iam.ResourceScalewayIamApplication(),
				"scaleway_iam_group":                           iam.ResourceScalewayIamGroup(),
				"scaleway_iam_group_membership":                iam.ResourceScalewayIamGroupMembership(),
				"scaleway_iam_policy":                          iam.ResourceScalewayIamPolicy(),
				"scaleway_iam_user":                            iam.ResourceScalewayIamUser(),
				"scaleway_instance_user_data":                  instance.ResourceScalewayInstanceUserData(),
				"scaleway_instance_image":                      instance.ResourceScalewayInstanceImage(),
				"scaleway_instance_ip":                         instance.ResourceScalewayInstanceIP(),
				"scaleway_instance_ip_reverse_dns":             instance.ResourceScalewayInstanceIPReverseDNS(),
				"scaleway_instance_volume":                     instance.ResourceScalewayInstanceVolume(),
				"scaleway_instance_security_group":             instance.ResourceScalewayInstanceSecurityGroup(),
				"scaleway_instance_security_group_rules":       instance.ResourceScalewayInstanceSecurityGroupRules(),
				"scaleway_instance_server":                     instance.ResourceScalewayInstanceServer(),
				"scaleway_instance_snapshot":                   instance.ResourceScalewayInstanceSnapshot(),
				"scaleway_iam_ssh_key":                         iam.ResourceScalewayIamSSKKey(),
				"scaleway_instance_placement_group":            instance.ResourceScalewayInstancePlacementGroup(),
				"scaleway_instance_private_nic":                instance.ResourceScalewayInstancePrivateNIC(),
				"scaleway_iot_hub":                             iot.ResourceScalewayIotHub(),
				"scaleway_iot_device":                          iot.ResourceScalewayIotDevice(),
				"scaleway_iot_route":                           iot.ResourceScalewayIotRoute(),
				"scaleway_iot_network":                         iot.ResourceScalewayIotNetwork(),
				"scaleway_ipam_ip":                             ipam.ResourceScalewayIPAMIP(),
				"scaleway_ipam_ip_reverse_dns":                 ipam.ResourceScalewayIPAMIPReverseDNS(),
				"scaleway_job_definition":                      jobs.ResourceDefinition(),
				"scaleway_k8s_cluster":                         k8s.ResourceScalewayK8SCluster(),
				"scaleway_k8s_pool":                            k8s.ResourceScalewayK8SPool(),
				"scaleway_lb":                                  lb.ResourceScalewayLb(),
				"scaleway_lb_acl":                              lb.ResourceScalewayLbACL(),
				"scaleway_lb_ip":                               lb.ResourceScalewayLbIP(),
				"scaleway_lb_backend":                          lb.ResourceScalewayLbBackend(),
				"scaleway_lb_certificate":                      lb.ResourceScalewayLbCertificate(),
				"scaleway_lb_frontend":                         lb.ResourceScalewayLbFrontend(),
				"scaleway_lb_route":                            lb.ResourceScalewayLbRoute(),
				"scaleway_registry_namespace":                  registry.ResourceScalewayRegistryNamespace(),
				"scaleway_tem_domain":                          tem.ResourceScalewayTemDomain(),
				"scaleway_container":                           container.ResourceScalewayContainer(),
				"scaleway_container_token":                     container.ResourceScalewayContainerToken(),
				"scaleway_rdb_acl":                             rdb.ResourceScalewayRdbACL(),
				"scaleway_rdb_database":                        rdb.ResourceScalewayRdbDatabase(),
				"scaleway_rdb_database_backup":                 rdb.ResourceScalewayRdbDatabaseBackup(),
				"scaleway_rdb_instance":                        rdb.ResourceScalewayRdbInstance(),
				"scaleway_rdb_privilege":                       rdb.ResourceScalewayRdbPrivilege(),
				"scaleway_rdb_user":                            rdb.ResourceScalewayRdbUser(),
				"scaleway_rdb_read_replica":                    rdb.ResourceScalewayRdbReadReplica(),
				"scaleway_redis_cluster":                       redis.ResourceScalewayRedisCluster(),
				"scaleway_sdb_sql_database":                    sdb.ResourceScalewaySDBSQLDatabase(),
				"scaleway_object":                              object.ResourceScalewayObject(),
				"scaleway_object_bucket":                       object.ResourceScalewayObjectBucket(),
				"scaleway_object_bucket_acl":                   object.ResourceScalewayObjectBucketACL(),
				"scaleway_object_bucket_lock_configuration":    object.ResourceObjectLockConfiguration(),
				"scaleway_object_bucket_policy":                object.ResourceScalewayObjectBucketPolicy(),
				"scaleway_object_bucket_website_configuration": object.ResourceBucketWebsiteConfiguration(),
				"scaleway_mnq_nats_account":                    mnq.ResourceScalewayMNQNatsAccount(),
				"scaleway_mnq_nats_credentials":                mnq.ResourceScalewayMNQNatsCredentials(),
				"scaleway_mnq_sns":                             mnq.ResourceScalewayMNQSNS(),
				"scaleway_mnq_sns_credentials":                 mnq.ResourceScalewayMNQSNSCredentials(),
				"scaleway_mnq_sns_topic":                       mnq.ResourceScalewayMNQSNSTopic(),
				"scaleway_mnq_sns_topic_subscription":          mnq.ResourceScalewayMNQSNSTopicSubscription(),
				"scaleway_mnq_sqs":                             mnq.ResourceScalewayMNQSQS(),
				"scaleway_mnq_sqs_queue":                       mnq.ResourceScalewayMNQSQSQueue(),
				"scaleway_mnq_sqs_credentials":                 mnq.ResourceScalewayMNQSQSCredentials(),
				"scaleway_secret":                              secret.ResourceScalewaySecret(),
				"scaleway_secret_version":                      secret.ResourceScalewaySecretVersion(),
				"scaleway_vpc":                                 vpc.ResourceScalewayVPC(),
				"scaleway_vpc_public_gateway":                  vpcgw.ResourceScalewayVPCPublicGateway(),
				"scaleway_vpc_gateway_network":                 vpcgw.ResourceScalewayVPCGatewayNetwork(),
				"scaleway_vpc_public_gateway_dhcp":             vpcgw.ResourceScalewayVPCPublicGatewayDHCP(),
				"scaleway_vpc_public_gateway_dhcp_reservation": vpcgw.ResourceScalewayVPCPublicGatewayDHCPReservation(),
				"scaleway_vpc_public_gateway_ip":               vpcgw.ResourceScalewayVPCPublicGatewayIP(),
				"scaleway_vpc_public_gateway_ip_reverse_dns":   vpcgw.ResourceScalewayVPCPublicGatewayIPReverseDNS(),
				"scaleway_vpc_public_gateway_pat_rule":         vpcgw.ResourceScalewayVPCPublicGatewayPATRule(),
				"scaleway_vpc_private_network":                 vpc.ResourceScalewayVPCPrivateNetwork(),
				"scaleway_webhosting":                          webhosting.ResourceScalewayWebhosting(),
			},

			DataSourcesMap: map[string]*schema.Resource{
				"scaleway_account_project":                     account.DataSourceScalewayAccountProject(),
				"scaleway_account_ssh_key":                     account.DataSourceScalewayAccountSSHKey(),
				"scaleway_availability_zones":                  az.DataSourceAvailabilityZones(),
				"scaleway_baremetal_offer":                     baremetal.DataSourceOffer(),
				"scaleway_baremetal_option":                    baremetal.DataSourceScalewayBaremetalOption(),
				"scaleway_baremetal_os":                        baremetal.DataSourceScalewayBaremetalOs(),
				"scaleway_baremetal_server":                    baremetal.DataSourceScalewayBaremetalServer(),
				"scaleway_billing_invoices":                    billing.DataSourceScalewayBillingInvoices(),
				"scaleway_billing_consumptions":                billing.DataSourceScalewayBillingConsumptions(),
				"scaleway_block_volume":                        block.DataSourceScalewayBlockVolume(),
				"scaleway_block_snapshot":                      block.DataSourceScalewayBlockSnapshot(),
				"scaleway_cockpit":                             cockpit.DataSourceScalewayCockpit(),
				"scaleway_cockpit_plan":                        cockpit.DataSourceScalewayCockpitPlan(),
				"scaleway_domain_record":                       domain.DataSourceScalewayDomainRecord(),
				"scaleway_domain_zone":                         domain.DataSourceScalewayDomainZone(),
				"scaleway_documentdb_instance":                 documentdb.DataSourceScalewayDocumentDBInstance(),
				"scaleway_documentdb_database":                 documentdb.DataSourceScalewayDocumentDBDatabase(),
				"scaleway_documentdb_load_balancer_endpoint":   documentdb.DataSourceScalewayDocumentDBEndpointLoadBalancer(),
				"scaleway_container_namespace":                 container.DataSourceScalewayContainerNamespace(),
				"scaleway_container":                           container.DataSourceScalewayContainer(),
				"scaleway_function":                            function.DataSourceScalewayFunction(),
				"scaleway_function_namespace":                  function.DataSourceScalewayFunctionNamespace(),
				"scaleway_iam_application":                     iam.DataSourceScalewayIamApplication(),
				"scaleway_flexible_ip":                         fip.DataSourceScalewayFlexibleIP(),
				"scaleway_flexible_ips":                        fip.DataSourceScalewayFlexibleIPs(),
				"scaleway_iam_group":                           iam.DataSourceScalewayIamGroup(),
				"scaleway_iam_ssh_key":                         iam.DataSourceScalewayIamSSHKey(),
				"scaleway_iam_user":                            iam.DataSourceScalewayIamUser(),
				"scaleway_instance_ip":                         instance.DataSourceScalewayInstanceIP(),
				"scaleway_instance_placement_group":            instance.DataSourceScalewayInstancePlacementGroup(),
				"scaleway_instance_private_nic":                instance.DataSourceScalewayInstancePrivateNIC(),
				"scaleway_instance_security_group":             instance.DataSourceSecurityGroup(),
				"scaleway_instance_server":                     instance.DataSourceScalewayInstanceServer(),
				"scaleway_instance_servers":                    instance.DataSourceScalewayInstanceServers(),
				"scaleway_instance_image":                      instance.DataSourceScalewayInstanceImage(),
				"scaleway_instance_volume":                     instance.DataSourceScalewayInstanceVolume(),
				"scaleway_instance_snapshot":                   instance.DataSourceScalewayInstanceSnapshot(),
				"scaleway_iot_hub":                             iot.DataSourceScalewayIotHub(),
				"scaleway_iot_device":                          iot.DataSourceScalewayIotDevice(),
				"scaleway_ipam_ip":                             ipam.DataSourceIP(),
				"scaleway_ipam_ips":                            ipam.DataSourceIPAMIPs(),
				"scaleway_k8s_cluster":                         k8s.DataSourceScalewayK8SCluster(),
				"scaleway_k8s_pool":                            k8s.DataSourceScalewayK8SPool(),
				"scaleway_k8s_version":                         k8s.DataSourceScalewayK8SVersion(),
				"scaleway_lb":                                  lb.DataSourceScalewayLb(),
				"scaleway_lbs":                                 lb.DataSourceScalewayLbs(),
				"scaleway_lb_acls":                             lb.DataSourceScalewayLbACLs(),
				"scaleway_lb_backend":                          lb.DataSourceScalewayLbBackend(),
				"scaleway_lb_backends":                         lb.DataSourceScalewayLbBackends(),
				"scaleway_lb_certificate":                      lb.DataSourceScalewayLbCertificate(),
				"scaleway_lb_frontend":                         lb.DataSourceScalewayLbFrontend(),
				"scaleway_lb_frontends":                        lb.DataSourceScalewayLbFrontends(),
				"scaleway_lb_ip":                               lb.DataSourceScalewayLbIP(),
				"scaleway_lb_ips":                              lb.DataSourceScalewayLbIPs(),
				"scaleway_lb_route":                            lb.DataSourceScalewayLbRoute(),
				"scaleway_lb_routes":                           lb.DataSourceScalewayLbRoutes(),
				"scaleway_marketplace_image":                   marketplace.DataSourceScalewayMarketplaceImage(),
				"scaleway_mnq_sqs":                             mnq.DataSourceScalewayMNQSQS(),
				"scaleway_object_bucket":                       object.DataSourceScalewayObjectBucket(),
				"scaleway_object_bucket_policy":                object.DataSourceScalewayObjectBucketPolicy(),
				"scaleway_rdb_acl":                             rdb.DataSourceScalewayRDBACL(),
				"scaleway_rdb_instance":                        rdb.DataSourceScalewayRDBInstance(),
				"scaleway_rdb_database":                        rdb.DataSourceScalewayRDBDatabase(),
				"scaleway_rdb_database_backup":                 rdb.DataSourceScalewayRDBDatabaseBackup(),
				"scaleway_rdb_privilege":                       rdb.DataSourceScalewayRDBPrivilege(),
				"scaleway_redis_cluster":                       redis.DataSourceScalewayRedisCluster(),
				"scaleway_registry_namespace":                  registry.DataSourceScalewayRegistryNamespace(),
				"scaleway_tem_domain":                          tem.DataSourceScalewayTemDomain(),
				"scaleway_secret":                              secret.DataSourceScalewaySecret(),
				"scaleway_secret_version":                      secret.DataSourceScalewaySecretVersion(),
				"scaleway_registry_image":                      registry.DataSourceScalewayRegistryImage(),
				"scaleway_vpc":                                 vpc.DataSourceScalewayVPC(),
				"scaleway_vpcs":                                vpc.DataSourceScalewayVPCs(),
				"scaleway_vpc_public_gateway":                  vpcgw.DataSourceScalewayVPCPublicGateway(),
				"scaleway_vpc_gateway_network":                 vpcgw.DataSourceScalewayVPCGatewayNetwork(),
				"scaleway_vpc_public_gateway_dhcp":             vpcgw.DataSourceScalewayVPCPublicGatewayDHCP(),
				"scaleway_vpc_public_gateway_dhcp_reservation": vpcgw.DataSourceScalewayVPCPublicGatewayDHCPReservation(),
				"scaleway_vpc_public_gateway_ip":               vpcgw.DataSourceScalewayVPCPublicGatewayIP(),
				"scaleway_vpc_private_network":                 vpc.DataSourceScalewayVPCPrivateNetwork(),
				"scaleway_vpc_public_gateway_pat_rule":         vpcgw.DataSourcePATRule(),
				"scaleway_webhosting":                          webhosting.DataSource(),
				"scaleway_webhosting_offer":                    webhosting.DataSourceOffer(),
			},
		}

		addBetaResources(p)

		p.ConfigureContextFunc = func(ctx context.Context, data *schema.ResourceData) (interface{}, diag.Diagnostics) {
			terraformVersion := p.TerraformVersion

			// If we provide meta in config use it. This is useful for tests
			if config.Meta != nil {
				return config.Meta, nil
			}

			meta, err := meta2.BuildMeta(ctx, &meta2.MetaConfig{
				ProviderSchema:   data,
				TerraformVersion: terraformVersion,
			})
			if err != nil {
				return nil, diag.FromErr(err)
			}
			return meta, nil
		}

		return p
	}
}

var terraformBetaEnabled = os.Getenv(scw.ScwEnableBeta) != ""

func addBetaResources(provider *schema.Provider) {
	if !terraformBetaEnabled {
		return
	}
	betaResources := map[string]*schema.Resource{}
	betaDataSources := map[string]*schema.Resource{}
	for resourceName, resource := range betaResources {
		provider.ResourcesMap[resourceName] = resource
	}
	for resourceName, resource := range betaDataSources {
		provider.DataSourcesMap[resourceName] = resource
	}
}

// ProviderConfig config can be used to provide additional config when creating provider.
type ProviderConfig struct {
	// Meta can be used to override Meta that will be used by the provider.
	// This is useful for tests.
	Meta *meta2.Meta
}

// DefaultProviderConfig return default ProviderConfig struct
func DefaultProviderConfig() *ProviderConfig {
	return &ProviderConfig{}
}
