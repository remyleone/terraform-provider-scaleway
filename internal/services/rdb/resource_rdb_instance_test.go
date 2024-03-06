package rdb_test

import (
	"fmt"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/logging"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/scaleway/scaleway-sdk-go/api/rdb/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

const (
	mySQLEngineName      = "MySQL"
	postgreSQLEngineName = "PostgreSQL"
)

func init() {
	resource.AddTestSweepers("scaleway_rdb_instance", &resource.Sweeper{
		Name: "scaleway_rdb_instance",
		F:    testSweepRDBInstance,
	})
}

func testAccCheckScalewayRdbEngineGetLatestVersion(tt *tests.TestTools, engineName string) string {
	api := rdb.NewAPI(tt.meta.GetScwClient())
	engines, err := api.ListDatabaseEngines(&rdb.ListDatabaseEnginesRequest{})
	if err != nil {
		tt.T.Fatalf("Could not get latest engine version: %s", err)
	}

	latestEngineVersion := ""
	for _, engine := range engines.Engines {
		if engine.Name == engineName {
			if len(engine.Versions) > 0 {
				latestEngineVersion = engine.Versions[0].Name
				break
			}
		}
	}
	return latestEngineVersion
}

func testSweepRDBInstance(_ string) error {
	return tests.SweepRegions(scw.AllRegions, func(scwClient *scw.Client, region scw.Region) error {
		rdbAPI := rdb.NewAPI(scwClient)
		logging.L.Debugf("sweeper: destroying the rdb instance in (%s)", region)
		listInstances, err := rdbAPI.ListInstances(&rdb.ListInstancesRequest{
			Region: region,
		}, scw.WithAllPages())
		if err != nil {
			return fmt.Errorf("error listing rdb instances in (%s) in sweeper: %s", region, err)
		}

		for _, instance := range listInstances.Instances {
			_, err := rdbAPI.DeleteInstance(&rdb.DeleteInstanceRequest{
				Region:     region,
				InstanceID: instance.ID,
			})
			if err != nil {
				return fmt.Errorf("error deleting rdb instance in sweeper: %s", err)
			}
		}

		return nil
	})
}

func TestAccScalewayRdbInstance_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	latestEngineVersion := testAccCheckScalewayRdbEngineGetLatestVersion(tt, postgreSQLEngineName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayRdbInstanceDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource scaleway_rdb_instance main {
						name = "test-rdb-basic"
						node_type = "db-dev-s"
						engine = %q
						is_ha_cluster = false
						disable_backup = true
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
						tags = [ "terraform-test", "scaleway_rdb_instance", "minimal" ]
					}
				`, latestEngineVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRdbExists(tt, "scaleway_rdb_instance.main"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "name", "test-rdb-basic"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "node_type", "db-dev-s"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "engine", latestEngineVersion),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "is_ha_cluster", "false"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "disable_backup", "true"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "user_name", "my_initial_user"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "password", "thiZ_is_v&ry_s3cret"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "tags.0", "terraform-test"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "tags.1", "scaleway_rdb_instance"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "tags.2", "minimal"),
					resource.TestCheckResourceAttrSet("scaleway_rdb_instance.main", "endpoint_ip"),
					resource.TestCheckResourceAttrSet("scaleway_rdb_instance.main", "endpoint_port"),
					resource.TestCheckResourceAttrSet("scaleway_rdb_instance.main", "certificate"),
					resource.TestCheckResourceAttrSet("scaleway_rdb_instance.main", "load_balancer.0.ip"),
					resource.TestCheckResourceAttrSet("scaleway_rdb_instance.main", "load_balancer.0.endpoint_id"),
					resource.TestCheckResourceAttrSet("scaleway_rdb_instance.main", "load_balancer.0.port"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource scaleway_rdb_instance main {
						name = "test-rdb-basic"
						node_type = "db-dev-s"
						engine = %q
						is_ha_cluster = false
						disable_backup = true
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
						tags = [ "terraform-change-tag" ]
					}
				`, latestEngineVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRdbExists(tt, "scaleway_rdb_instance.main"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "name", "test-rdb-basic"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "node_type", "db-dev-s"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "engine", latestEngineVersion),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "is_ha_cluster", "false"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "disable_backup", "true"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "user_name", "my_initial_user"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "password", "thiZ_is_v&ry_s3cret"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "tags.0", "terraform-change-tag"),
					resource.TestCheckResourceAttrSet("scaleway_rdb_instance.main", "endpoint_ip"),
					resource.TestCheckResourceAttrSet("scaleway_rdb_instance.main", "endpoint_port"),
					resource.TestCheckResourceAttrSet("scaleway_rdb_instance.main", "certificate"),
					resource.TestCheckResourceAttrSet("scaleway_rdb_instance.main", "load_balancer.0.ip"),
					resource.TestCheckResourceAttrSet("scaleway_rdb_instance.main", "load_balancer.0.endpoint_id"),
					resource.TestCheckResourceAttrSet("scaleway_rdb_instance.main", "load_balancer.0.port"),
				),
			},
		},
	})
}

func TestAccScalewayRdbInstance_WithCluster(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	latestEngineVersion := testAccCheckScalewayRdbEngineGetLatestVersion(tt, postgreSQLEngineName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayRdbInstanceDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource scaleway_rdb_instance main {
						name = "test-rdb-with-cluster"
						node_type = "db-dev-m"
						engine = %q
						is_ha_cluster = true
						disable_backup = false
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s8cret"
						tags = [ "terraform-test", "scaleway_rdb_instance", "minimal" ]
					}
				`, latestEngineVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRdbExists(tt, "scaleway_rdb_instance.main"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "name", "test-rdb-with-cluster"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "node_type", "db-dev-m"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "engine", latestEngineVersion),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "is_ha_cluster", "true"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "disable_backup", "false"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "user_name", "my_initial_user"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "password", "thiZ_is_v&ry_s8cret"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "tags.0", "terraform-test"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "tags.1", "scaleway_rdb_instance"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "tags.2", "minimal"),
				),
			},
		},
	})
}

func TestAccScalewayRdbInstance_Settings(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	latestEngineVersion := testAccCheckScalewayRdbEngineGetLatestVersion(tt, postgreSQLEngineName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayRdbInstanceDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource scaleway_rdb_instance main {
						name = "test-rdb-instance-settings"
						node_type = "db-dev-s"
						disable_backup = true
						engine = %q
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
						settings = {
							work_mem = "4"
							max_connections = "200"
							effective_cache_size = "1300"
							maintenance_work_mem = "150"
							max_parallel_workers = "2"
							max_parallel_workers_per_gather = "2"
						}
					}
				`, latestEngineVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRdbExists(tt, "scaleway_rdb_instance.main"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "settings.work_mem", "4"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "settings.max_connections", "200"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "settings.effective_cache_size", "1300"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "settings.maintenance_work_mem", "150"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "settings.max_parallel_workers", "2"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "settings.max_parallel_workers_per_gather", "2"),
				),
			},
		},
	})
}

func TestAccScalewayRdbInstance_InitSettings(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	latestEngineVersion := testAccCheckScalewayRdbEngineGetLatestVersion(tt, mySQLEngineName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayRdbInstanceDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource scaleway_rdb_instance main {
						name = "test-rdb-init-settings"
						node_type = "db-dev-s"
						disable_backup = true
						engine = %q
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
						init_settings = {
							"lower_case_table_names" = 1
						}
						settings = {
							"max_connections" = "350"
						}
					}
				`, latestEngineVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRdbExists(tt, "scaleway_rdb_instance.main"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "init_settings.lower_case_table_names", "1"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "settings.max_connections", "350"),
				),
			},
		},
	})
}

func TestAccScalewayRdbInstance_Capitalize(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	latestEngineVersion := testAccCheckScalewayRdbEngineGetLatestVersion(tt, postgreSQLEngineName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayRdbInstanceDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource scaleway_rdb_instance main {
						name = "test-rdb-instance-capitalize"
						node_type = "DB-DEV-S"
						engine = %q
						is_ha_cluster = false
						disable_backup = true
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
					}
				`, latestEngineVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRdbExists(tt, "scaleway_rdb_instance.main"),
				),
			},
		},
	})
}

func TestAccScalewayRdbInstance_PrivateNetwork(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	latestEngineVersion := testAccCheckScalewayRdbEngineGetLatestVersion(tt, postgreSQLEngineName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayRdbInstanceDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_vpc_private_network pn01 {
						name = "my_private_network"
						region= "nl-ams"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("scaleway_vpc_private_network.pn01", "name", "my_private_network"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource scaleway_vpc_private_network pn01 {
						name = "my_private_network"
						region= "nl-ams"
					}

					resource scaleway_rdb_instance main {
						name = "test-rdb-private-network"
						node_type = "db-dev-s"
						engine = %q
						is_ha_cluster = false
						disable_backup = true
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
						region= "nl-ams"
						tags = [ "terraform-test", "scaleway_rdb_instance", "volume", "rdb_pn" ]
						volume_type = "bssd"
						volume_size_in_gb = 10
						private_network {
							ip_net = "192.168.1.42/24"
							pn_id = "${scaleway_vpc_private_network.pn01.id}"
						}
					}
				`, latestEngineVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRdbExists(tt, "scaleway_rdb_instance.main"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "private_network.#", "1"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "private_network.0.ip_net", "192.168.1.42/24"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource scaleway_vpc_private_network pn01 {
						name = "my_private_network_to_be_replaced"
						region= "nl-ams"
					}

					resource scaleway_vpc_private_network pn02 {
						name = "my_private_network"
						region= "nl-ams"
					}

					resource scaleway_rdb_instance main {
						name = "test-rdb-private-network"
						node_type = "db-dev-s"
						engine = %q
						is_ha_cluster = false
						disable_backup = true
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
						region= "nl-ams"
						tags = [ "terraform-test", "scaleway_rdb_instance", "volume", "rdb_pn" ]
						volume_type = "bssd"
						volume_size_in_gb = 10
						private_network {
							ip_net = "192.168.1.254/24"
							pn_id = "${scaleway_vpc_private_network.pn02.id}"
						}
					}
				`, latestEngineVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRdbExists(tt, "scaleway_rdb_instance.main"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "private_network.#", "1"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "private_network.0.ip_net", "192.168.1.254/24"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource scaleway_vpc_private_network pn01 {
						name = "my_private_network_to_be_replaced"
						region= "nl-ams"
					}

					resource scaleway_vpc_private_network pn02 {
						name = "my_private_network"
						region= "nl-ams"
					}

					resource scaleway_vpc_public_gateway_dhcp main {
						subnet = "192.168.1.0/24"
						zone = "nl-ams-1"
					}

					resource scaleway_vpc_public_gateway_ip main {
						zone = "nl-ams-1"
					}

					resource scaleway_vpc_public_gateway main {
						name = "foobar"
						type = "VPC-GW-S"
						zone = "nl-ams-1"
						ip_id = scaleway_vpc_public_gateway_ip.main.id
					}

					resource scaleway_vpc_public_gateway_pat_rule main {
						gateway_id = scaleway_vpc_public_gateway.main.id
						private_ip = scaleway_vpc_public_gateway_dhcp.main.address
						private_port = scaleway_rdb_instance.main.private_network.0.port
						public_port = 42
						protocol = "both"
						zone = "nl-ams-1"
						depends_on = [scaleway_vpc_gateway_network.main, scaleway_vpc_private_network.pn02]
					}

					resource scaleway_vpc_gateway_network main {
						gateway_id = scaleway_vpc_public_gateway.main.id
						private_network_id = scaleway_vpc_private_network.pn02.id
						dhcp_id = scaleway_vpc_public_gateway_dhcp.main.id
						cleanup_dhcp = true
						enable_masquerade = true
						zone = "nl-ams-1"
						depends_on = [scaleway_vpc_public_gateway_ip.main, scaleway_vpc_private_network.pn02]
					}

					resource scaleway_rdb_instance main {
						name = "test-rdb-private-network"
						node_type = "db-dev-s"
						engine = %q
						is_ha_cluster = false
						disable_backup = true
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
						region= "nl-ams"
						tags = [ "terraform-test", "scaleway_rdb_instance", "volume", "rdb_pn" ]
						volume_type = "bssd"
						volume_size_in_gb = 10
						private_network {
							ip_net = "192.168.1.254/24" #pool high
							pn_id = "${scaleway_vpc_private_network.pn02.id}"
						}
					}
				`, latestEngineVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRdbExists(tt, "scaleway_rdb_instance.main"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "private_network.0.ip_net", "192.168.1.254/24"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource scaleway_vpc_private_network pn02 {
						name = "my_private_network"
						region= "nl-ams"
					}

					resource scaleway_rdb_instance main {
						name = "test-rdb-private-network"
						node_type = "db-dev-s"
						engine = %q
						is_ha_cluster = false
						disable_backup = true
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
						region= "nl-ams"
						tags = [ "terraform-test", "scaleway_rdb_instance", "volume", "rdb_pn" ]
						volume_type = "bssd"
						volume_size_in_gb = 10
					}
				`, latestEngineVersion),
			},
		},
	})
}

func TestAccScalewayRdbInstance_PrivateNetworkUpdate(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	latestEngineVersion := testAccCheckScalewayRdbEngineGetLatestVersion(tt, postgreSQLEngineName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckScalewayRdbInstanceDestroy(tt),
			testAccCheckScalewayVPCPrivateNetworkDestroy(tt),
		),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource scaleway_vpc_private_network pn01 {
						name = "my_private_network"
					}

					resource scaleway_rdb_instance main {
						name = "test-rdb-private-network-update"
						node_type = "db-dev-s"
						engine = %q
						is_ha_cluster = false
						disable_backup = true
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
						tags = [ "terraform-test", "scaleway_rdb_instance", "volume", "rdb_pn" ]
						volume_type = "bssd"
						volume_size_in_gb = 10
						private_network {
							pn_id = "${scaleway_vpc_private_network.pn01.id}"
							enable_ipam = true
						}
					}
				`, latestEngineVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayVPCPrivateNetworkExists(tt, "scaleway_vpc_private_network.pn01"),
					testAccCheckScalewayRdbExists(tt, "scaleway_rdb_instance.main"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "private_network.#", "1"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "private_network.0.enable_ipam", "true"),
					resource.TestCheckResourceAttrPair("scaleway_rdb_instance.main", "private_network.0.pn_id", "scaleway_vpc_private_network.pn01", "id"),
				),
			},
			// Change PN but keep ipam config
			{
				Config: fmt.Sprintf(`
					resource scaleway_vpc_private_network pn01 {
						name = "my_private_network"
					}
					resource scaleway_vpc_private_network pn02 {
						name = "my_second_private_network"
					}
			
					resource scaleway_rdb_instance main {
						name = "test-rdb-private-network-update"
						node_type = "db-dev-s"
						engine = %q
						is_ha_cluster = false
						disable_backup = true
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
						tags = [ "terraform-test", "scaleway_rdb_instance", "volume", "rdb_pn" ]
						volume_type = "bssd"
						volume_size_in_gb = 10
						private_network {
							pn_id = "${scaleway_vpc_private_network.pn02.id}"
							enable_ipam = true
						}
					}
				`, latestEngineVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayVPCPrivateNetworkExists(tt, "scaleway_vpc_private_network.pn01"),
					testAccCheckScalewayVPCPrivateNetworkExists(tt, "scaleway_vpc_private_network.pn02"),
					testAccCheckScalewayRdbExists(tt, "scaleway_rdb_instance.main"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "private_network.#", "1"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "private_network.0.enable_ipam", "true"),
					resource.TestCheckResourceAttrPair("scaleway_rdb_instance.main", "private_network.0.pn_id", "scaleway_vpc_private_network.pn02", "id"),
				),
			},
			// Keep PN but change ipam config -> static
			{
				Config: fmt.Sprintf(`
					resource scaleway_vpc_private_network pn01 {
						name = "my_private_network"
					}
					resource scaleway_vpc_private_network pn02 {
						name = "my_second_private_network"
					}
			
					locals {
						ip_address  = cidrhost(scaleway_vpc_private_network.pn02.ipv4_subnet.0.subnet, 4)
						cidr_prefix = split("/", scaleway_vpc_private_network.pn02.ipv4_subnet.0.subnet)[1]
					}
			
					resource scaleway_rdb_instance main {
						name = "test-rdb-private-network-update"
						node_type = "db-dev-s"
						engine = %q
						is_ha_cluster = false
						disable_backup = true
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
						tags = [ "terraform-test", "scaleway_rdb_instance", "volume", "rdb_pn" ]
						volume_type = "bssd"
						volume_size_in_gb = 10`, latestEngineVersion) + `
						private_network {
							ip_net = format("%s/%s", local.ip_address, local.cidr_prefix)
							pn_id  = scaleway_vpc_private_network.pn02.id
						}
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayVPCPrivateNetworkExists(tt, "scaleway_vpc_private_network.pn02"),
					testAccCheckScalewayRdbExists(tt, "scaleway_rdb_instance.main"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "private_network.#", "1"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "private_network.0.enable_ipam", "false"),
					resource.TestCheckResourceAttrPair("scaleway_rdb_instance.main", "private_network.0.pn_id", "scaleway_vpc_private_network.pn02", "id"),
				),
			},
			// Change PN but keep static config
			{
				Config: fmt.Sprintf(`
					resource scaleway_vpc_private_network pn01 {
						name = "my_private_network"
					}
					resource scaleway_vpc_private_network pn02 {
						name = "my_second_private_network"
					}

					locals {
						ip_address  = cidrhost(scaleway_vpc_private_network.pn01.ipv4_subnet.0.subnet, 4)
						cidr_prefix = split("/", scaleway_vpc_private_network.pn01.ipv4_subnet.0.subnet)[1]
					}

					resource scaleway_rdb_instance main {
						name = "test-rdb-private-network-update"
						node_type = "db-dev-s"
						engine = %q
						is_ha_cluster = false
						disable_backup = true
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
						tags = [ "terraform-test", "scaleway_rdb_instance", "volume", "rdb_pn" ]
						volume_type = "bssd"
						volume_size_in_gb = 10`, latestEngineVersion) + `
						private_network {
							ip_net = format("%s/%s", local.ip_address, local.cidr_prefix)
							pn_id  = scaleway_vpc_private_network.pn01.id
						}
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayVPCPrivateNetworkExists(tt, "scaleway_vpc_private_network.pn01"),
					testAccCheckScalewayVPCPrivateNetworkExists(tt, "scaleway_vpc_private_network.pn02"),
					testAccCheckScalewayRdbExists(tt, "scaleway_rdb_instance.main"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "private_network.#", "1"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "private_network.0.enable_ipam", "false"),
					resource.TestCheckResourceAttrPair("scaleway_rdb_instance.main", "private_network.0.pn_id", "scaleway_vpc_private_network.pn01", "id"),
				),
			},
			// Keep PN but change static config -> ipam
			{
				Config: fmt.Sprintf(`
					resource scaleway_vpc_private_network pn01 {
						name = "my_private_network"
					}
			
					resource scaleway_rdb_instance main {
						name = "test-rdb-private-network-update"
						node_type = "db-dev-s"
						engine = %q
						is_ha_cluster = false
						disable_backup = true
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
						tags = [ "terraform-test", "scaleway_rdb_instance", "volume", "rdb_pn" ]
						volume_type = "bssd"
						volume_size_in_gb = 10`, latestEngineVersion) + `
						private_network {
							pn_id  = scaleway_vpc_private_network.pn01.id
							enable_ipam = true
						}
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayVPCPrivateNetworkExists(tt, "scaleway_vpc_private_network.pn01"),
					testAccCheckScalewayRdbExists(tt, "scaleway_rdb_instance.main"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "private_network.#", "1"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "private_network.0.enable_ipam", "true"),
					resource.TestCheckResourceAttrPair("scaleway_rdb_instance.main", "private_network.0.pn_id", "scaleway_vpc_private_network.pn01", "id"),
				),
			},
		},
	})
}

func TestAccScalewayRdbInstance_PrivateNetwork_DHCP(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	latestEngineVersion := testAccCheckScalewayRdbEngineGetLatestVersion(tt, postgreSQLEngineName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayRdbInstanceDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource scaleway_vpc_private_network pn02 {
						name = "my_private_network"
						region= "nl-ams"
					}

					resource scaleway_vpc_public_gateway_dhcp main {
						subnet = "192.168.1.0/24"
						zone = "nl-ams-1"
					}

					resource scaleway_vpc_public_gateway_ip main {
						zone = "nl-ams-1"
					}

					resource scaleway_vpc_public_gateway main {
						name = "foobar"
						type = "VPC-GW-S"
						zone = "nl-ams-1"
						ip_id = scaleway_vpc_public_gateway_ip.main.id
					}

					resource scaleway_vpc_public_gateway_pat_rule main {
						gateway_id = scaleway_vpc_public_gateway.main.id
						private_ip = scaleway_vpc_public_gateway_dhcp.main.address
						private_port = scaleway_rdb_instance.main.private_network.0.port
						public_port = 42
						protocol = "both"
						zone = "nl-ams-1"
						depends_on = [scaleway_vpc_gateway_network.main, scaleway_vpc_private_network.pn02]
					}

					resource scaleway_vpc_gateway_network main {
						gateway_id = scaleway_vpc_public_gateway.main.id
						private_network_id = scaleway_vpc_private_network.pn02.id
						dhcp_id = scaleway_vpc_public_gateway_dhcp.main.id
						cleanup_dhcp = true
						enable_masquerade = true
						zone = "nl-ams-1"
						depends_on = [scaleway_vpc_public_gateway_ip.main, scaleway_vpc_private_network.pn02]
					}

					resource scaleway_rdb_instance main {
						name = "test-rdb-private-network-dhcp"
						node_type = "db-dev-s"
						engine = %q
						is_ha_cluster = false
						disable_backup = true
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
						region= "nl-ams"
						tags = [ "terraform-test", "scaleway_rdb_instance", "volume", "rdb_pn" ]
						volume_type = "bssd"
						volume_size_in_gb = 10
						private_network {
							ip_net = "192.168.1.254/24" #pool high
							pn_id = "${scaleway_vpc_private_network.pn02.id}"
						}
					}
				`, latestEngineVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRdbExists(tt, "scaleway_rdb_instance.main"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "private_network.0.ip_net", "192.168.1.254/24"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource scaleway_vpc_private_network pn02 {
						name = "my_private_network"
						region= "nl-ams"
					}

					resource scaleway_rdb_instance main {
						name = "test-rdb-private-network-dhcp"
						node_type = "db-dev-s"
						engine = %q
						is_ha_cluster = false
						disable_backup = true
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
						region= "nl-ams"
						tags = [ "terraform-test", "scaleway_rdb_instance", "volume", "rdb_pn" ]
						volume_type = "bssd"
						volume_size_in_gb = 10
					}
				`, latestEngineVersion),
			},
		},
	})
}

func TestAccScalewayRdbInstance_BackupSchedule(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	latestEngineVersion := testAccCheckScalewayRdbEngineGetLatestVersion(tt, postgreSQLEngineName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayRdbInstanceDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource scaleway_rdb_instance main {
						name                      = "test-rdb-instance-backup-schedule"
						node_type                 = "db-dev-s"
						engine                    = %q
						is_ha_cluster             = false
						disable_backup            = false
                        backup_schedule_frequency = 24
                        backup_schedule_retention = 7
						backup_same_region        = true
						user_name                 = "my_initial_user"
						password                  = "thiZ_is_v&ry_s3cret"
						region                    = "nl-ams"
						tags                      = [ "terraform-test", "scaleway_rdb_instance", "volume" ]
					}
				`, latestEngineVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRdbExists(tt, "scaleway_rdb_instance.main"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "disable_backup", "false"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "backup_schedule_frequency", "24"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "backup_schedule_retention", "7"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "backup_same_region", "true"),
				),
			},
		},
	})
}

func TestAccScalewayRdbInstance_Volume(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	latestEngineVersion := testAccCheckScalewayRdbEngineGetLatestVersion(tt, postgreSQLEngineName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayRdbInstanceDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource scaleway_rdb_instance main {
						name = "test-rdb-instance-volume"
						node_type = "db-dev-s"
						engine = %q
						is_ha_cluster = false
						disable_backup = true
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
						region= "nl-ams"
						tags = [ "terraform-test", "scaleway_rdb_instance", "volume" ]
					}
				`, latestEngineVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRdbExists(tt, "scaleway_rdb_instance.main"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "volume_type", "lssd"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource scaleway_rdb_instance main {
						name = "test-rdb-instance-volume"
						node_type = "db-dev-s"
						engine = %q
						is_ha_cluster = false
						disable_backup = true
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
						region= "nl-ams"
						tags = [ "terraform-test", "scaleway_rdb_instance", "volume" ]
						volume_type = "bssd"
						volume_size_in_gb = 10
					}
				`, latestEngineVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRdbExists(tt, "scaleway_rdb_instance.main"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "volume_type", "bssd"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "volume_size_in_gb", "10"),
				),
			},
		},
	})
}

func TestAccScalewayRdbInstance_SBSVolume(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	latestEngineVersion := testAccCheckScalewayRdbEngineGetLatestVersion(tt, postgreSQLEngineName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayRdbInstanceDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource scaleway_rdb_instance main {
						name = "test-rdb-instance-volume"
						node_type = "db-play2-pico"
						engine = %q
						is_ha_cluster = false
						disable_backup = true
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
						region= "nl-ams"
						tags = [ "terraform-test", "scaleway_rdb_instance", "sdb-volume" ]
						volume_type = "sbs_5k"
						volume_size_in_gb = 10
					}
				`, latestEngineVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRdbExists(tt, "scaleway_rdb_instance.main"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "volume_type", "sbs_5k"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "volume_size_in_gb", "10"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource scaleway_rdb_instance main {
						name = "test-rdb-instance-volume"
						node_type = "db-play2-pico"
						engine = %q
						is_ha_cluster = false
						disable_backup = true
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
						region= "nl-ams"
						tags = [ "terraform-test", "scaleway_rdb_instance", "volume" ]
						volume_type = "sbs_5k"
						volume_size_in_gb = 20
					}
				`, latestEngineVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRdbExists(tt, "scaleway_rdb_instance.main"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "volume_type", "sbs_5k"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "volume_size_in_gb", "20"),
				),
			},
		},
	})
}

func TestAccScalewayRdbInstance_ChangeVolumeType(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	latestEngineVersion := testAccCheckScalewayRdbEngineGetLatestVersion(tt, postgreSQLEngineName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayRdbInstanceDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource scaleway_rdb_instance main {
						name = "test-rdb-instance-volume"
						node_type = "db-play2-pico"
						engine = %q
						is_ha_cluster = false
						disable_backup = true
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
						region= "nl-ams"
						tags = [ "terraform-test", "scaleway_rdb_instance", "sdb-volume" ]
						volume_type = "bssd"
						volume_size_in_gb = 10
					}
				`, latestEngineVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRdbExists(tt, "scaleway_rdb_instance.main"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "volume_type", "bssd"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "volume_size_in_gb", "10"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource scaleway_rdb_instance main {
						name = "test-rdb-instance-volume"
						node_type = "db-play2-pico"
						engine = %q
						is_ha_cluster = false
						disable_backup = true
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
						region= "nl-ams"
						tags = [ "terraform-test", "scaleway_rdb_instance", "volume" ]
						volume_type = "sbs_5k"
						volume_size_in_gb = 20
					}
				`, latestEngineVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRdbExists(tt, "scaleway_rdb_instance.main"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "volume_type", "sbs_5k"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.main", "volume_size_in_gb", "20"),
				),
			},
		},
	})
}

func TestAccScalewayRdbInstance_Endpoints(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	latestEngineVersion := testAccCheckScalewayRdbEngineGetLatestVersion(tt, postgreSQLEngineName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayRdbInstanceDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_vpc_private_network" "test_endpoints" {
						name = "test-rdb-endpoints"
					}

					resource "scaleway_rdb_instance" "test_endpoints" {
						name = "test-rdb-endpoints"
						node_type = "db-dev-s"
						engine = %q
						is_ha_cluster = false
						disable_backup = true
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
						tags = [ "terraform-test", "scaleway_rdb_instance", "test_endpoints" ]
						private_network {
							pn_id = scaleway_vpc_private_network.test_endpoints.id
							enable_ipam = true
						}
					}
				`, latestEngineVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRdbExists(tt, "scaleway_rdb_instance.test_endpoints"),
					testAccCheckScalewayVPCPrivateNetworkExists(tt, "scaleway_vpc_private_network.test_endpoints"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.test_endpoints", "private_network.#", "1"),
					resource.TestCheckResourceAttrPair("scaleway_rdb_instance.test_endpoints", "private_network.0.pn_id", "scaleway_vpc_private_network.test_endpoints", "id"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.test_endpoints", "private_network.0.enable_ipam", "true"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.test_endpoints", "load_balancer.#", "0"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.test_endpoints", "endpoint_ip", ""),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.test_endpoints", "endpoint_port", "0"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "scaleway_vpc_private_network" "test_endpoints" {
						name = "test-rdb-endpoints"
					}

					resource "scaleway_rdb_instance" "test_endpoints" {
						name = "test-rdb-endpoints"
						node_type = "db-dev-s"
						engine = %q
						is_ha_cluster = false
						disable_backup = true
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
						tags = [ "terraform-test", "scaleway_rdb_instance", "test_endpoints" ]
						private_network {
							pn_id = scaleway_vpc_private_network.test_endpoints.id
							enable_ipam = true
						}
						load_balancer {}
					}
				`, latestEngineVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRdbExists(tt, "scaleway_rdb_instance.test_endpoints"),
					testAccCheckScalewayVPCPrivateNetworkExists(tt, "scaleway_vpc_private_network.test_endpoints"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.test_endpoints", "load_balancer.#", "1"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.test_endpoints", "private_network.#", "1"),
					resource.TestCheckResourceAttrPair("scaleway_rdb_instance.test_endpoints", "private_network.0.pn_id", "scaleway_vpc_private_network.test_endpoints", "id"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.test_endpoints", "private_network.0.enable_ipam", "true"),
					resource.TestCheckResourceAttrSet("scaleway_rdb_instance.test_endpoints", "endpoint_ip"),
					resource.TestCheckResourceAttrSet("scaleway_rdb_instance.test_endpoints", "endpoint_port"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "scaleway_vpc_private_network" "test_endpoints" {
						name = "test-rdb-endpoints"
					}

					resource "scaleway_rdb_instance" "test_endpoints" {
						name = "test-rdb-endpoints"
						node_type = "db-dev-s"
						engine = %q
						is_ha_cluster = false
						disable_backup = true
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
						tags = [ "terraform-test", "scaleway_rdb_instance", "test_endpoints" ]
					}
				`, latestEngineVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRdbExists(tt, "scaleway_rdb_instance.test_endpoints"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.test_endpoints", "load_balancer.#", "1"),
					resource.TestCheckResourceAttr("scaleway_rdb_instance.test_endpoints", "private_network.#", "0"),
					resource.TestCheckResourceAttrSet("scaleway_rdb_instance.test_endpoints", "endpoint_ip"),
					resource.TestCheckResourceAttrSet("scaleway_rdb_instance.test_endpoints", "endpoint_port"),
				),
			},
		},
	})
}

func testAccCheckScalewayRdbExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		rdbAPI, region, ID, err := rdbAPIWithRegionAndID(tt.Meta, rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = rdbAPI.GetInstance(&rdb.GetInstanceRequest{
			InstanceID: ID,
			Region:     region,
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckScalewayRdbInstanceDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_rdb_instance" {
				continue
			}

			rdbAPI, region, ID, err := rdbAPIWithRegionAndID(tt.Meta, rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = rdbAPI.GetInstance(&rdb.GetInstanceRequest{
				InstanceID: ID,
				Region:     region,
			})

			// If no error resource still exist
			if err == nil {
				return fmt.Errorf("instance (%s) still exists", rs.Primary.ID)
			}

			// Unexpected api error we return it
			if !http_errors.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}
