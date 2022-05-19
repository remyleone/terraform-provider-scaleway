package scaleway

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	redis "github.com/scaleway/scaleway-sdk-go/api/redis/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func init() {
	resource.AddTestSweepers("scaleway_redis_cluster", &resource.Sweeper{
		Name: "scaleway_redis_cluster",
		F:    testSweepRedisCluster,
	})
}

func testSweepRedisCluster(_ string) error {
	return sweepZones(scw.AllZones, func(scwClient *scw.Client, zone scw.Zone) error {
		redisAPI := redis.NewAPI(scwClient)
		l.Debugf("sweeper: destroying the redis cluster in (%s)", zone)
		listClusters, err := redisAPI.ListClusters(&redis.ListClustersRequest{
			Zone: zone,
		}, scw.WithAllPages())
		if err != nil {
			return fmt.Errorf("error listing redis clusters in (%s) in sweeper: %w", zone, err)
		}

		for _, cluster := range listClusters.Clusters {
			_, err := redisAPI.DeleteCluster(&redis.DeleteClusterRequest{
				Zone:      zone,
				ClusterID: cluster.ID,
			})
			if err != nil {
				return fmt.Errorf("error deleting redis cluster in sweeper: %w", err)
			}
		}

		return nil
	})
}

func TestAccScalewayRedisCluster_Basic(t *testing.T) {
	tt := NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayRedisClusterDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_redis_cluster" "main" {
    					name = "test_redis_basic"
    					version = "6.2.6"
    					node_type = "MDB-BETA-M"
    					user_name = "my_initial_user"
    					password = "thiZ_is_v&ry_s3cret"
						tags = [ "test1" ]
						cluster_size = 1
						tls_enabled = "true"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRedisExists(tt, "scaleway_redis_cluster.main"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "name", "test_redis_basic"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "version", "6.2.6"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "node_type", "MDB-BETA-M"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "user_name", "my_initial_user"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "password", "thiZ_is_v&ry_s3cret"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "tags.0", "test1"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "cluster_size", "1"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "tls_enabled", "true"),
				),
			},
			{
				Config: `
					resource "scaleway_redis_cluster" "main" {
    					name = "test_redis_basic_edit"
    					version = "6.2.6"
    					node_type = "MDB-BETA-M"
    					user_name = "new_user"
    					password = "thiZ_is_A_n3w_passw0rd"
						tags = [ "test1", "other_tag" ]
						cluster_size = 1
						tls_enabled = "true"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRedisExists(tt, "scaleway_redis_cluster.main"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "name", "test_redis_basic_edit"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "version", "6.2.6"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "node_type", "MDB-BETA-M"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "user_name", "new_user"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "password", "thiZ_is_A_n3w_passw0rd"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "tags.0", "test1"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "tags.1", "other_tag"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "cluster_size", "1"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "tls_enabled", "true"),
				),
			},
		},
	})
}

func TestAccScalewayRedisCluster_Migrate(t *testing.T) {
	tt := NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayRedisClusterDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_redis_cluster" "main" {
    					name = "test_redis_basic"
    					version = "6.2.6"
    					node_type = "MDB-BETA-M"
    					user_name = "my_initial_user"
    					password = "thiZ_is_v&ry_s3cret"
						tags = [ "test1" ]
						cluster_size = 1
						tls_enabled = "true"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRedisExists(tt, "scaleway_redis_cluster.main"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "name", "test_redis_basic"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "version", "6.2.6"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "node_type", "MDB-BETA-M"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "user_name", "my_initial_user"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "password", "thiZ_is_v&ry_s3cret"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "tags.0", "test1"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "cluster_size", "1"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "tls_enabled", "true"),
				),
			},
			{
				Config: `
					resource "scaleway_redis_cluster" "main" {
    					name = "test_redis_basic"
    					version = "6.2.6"
    					node_type = "MDB-BETA-L"
    					user_name = "my_initial_user"
    					password = "thiZ_is_v&ry_s3cret"
						tags = [ "test1" ]
						cluster_size = 1
						tls_enabled = "true"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRedisExists(tt, "scaleway_redis_cluster.main"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "name", "test_redis_basic"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "version", "6.2.6"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "node_type", "MDB-BETA-L"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "user_name", "my_initial_user"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "password", "thiZ_is_v&ry_s3cret"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "tags.0", "test1"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "cluster_size", "1"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "tls_enabled", "true"),
				),
			},
		},
	})
}

func TestAccScalewayRedisCluster_ACL(t *testing.T) {
	tt := NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayRedisClusterDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_redis_cluster" "main" {
    					name = "test_redis_acl"
    					version = "6.2.6"
    					node_type = "MDB-BETA-M"
    					user_name = "my_initial_user"
    					password = "thiZ_is_v&ry_s3cret"
						acl {
							ip = "0.0.0.0/0"
							description = "An acl description"
						}
						acl {
							ip = "192.168.10.0/24"
							description = "A second acl description"
						}
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRedisExists(tt, "scaleway_redis_cluster.main"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "name", "test_redis_acl"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "version", "6.2.6"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "node_type", "MDB-BETA-M"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "user_name", "my_initial_user"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "password", "thiZ_is_v&ry_s3cret"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "acl.0.ip", "0.0.0.0/0"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "acl.0.description", "An acl description"),
					resource.TestCheckResourceAttrSet("scaleway_redis_cluster.main", "acl.0.id"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "acl.1.ip", "192.168.10.0/24"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "acl.1.description", "A second acl description"),
					resource.TestCheckResourceAttrSet("scaleway_redis_cluster.main", "acl.1.id"),
				),
			},
			{
				Config: `
					resource "scaleway_redis_cluster" "main" {
    					name = "test_redis_acl"
    					version = "6.2.6"
    					node_type = "MDB-BETA-M"
    					user_name = "my_initial_user"
    					password = "thiZ_is_v&ry_s3cret"
						acl {
							ip = "192.168.11.0/24"
							description = "Another acl description"
						}
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRedisExists(tt, "scaleway_redis_cluster.main"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "name", "test_redis_acl"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "version", "6.2.6"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "node_type", "MDB-BETA-M"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "user_name", "my_initial_user"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "password", "thiZ_is_v&ry_s3cret"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "acl.0.ip", "192.168.11.0/24"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "acl.0.description", "Another acl description"),
					resource.TestCheckResourceAttrSet("scaleway_redis_cluster.main", "acl.0.id"),
				),
			},
		},
	})
}

func TestAccScalewayRedisCluster_Settings(t *testing.T) {
	tt := NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayRedisClusterDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_redis_cluster" "main" {
    					name = "test_redis_settings"
    					version = "6.2.6"
    					node_type = "MDB-BETA-M"
    					user_name = "my_initial_user"
    					password = "thiZ_is_v&ry_s3cret"
						settings = {
							"tcp-keepalive" = "150"
							"maxclients" = "5000"
						}
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRedisExists(tt, "scaleway_redis_cluster.main"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "name", "test_redis_settings"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "version", "6.2.6"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "node_type", "MDB-BETA-M"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "user_name", "my_initial_user"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "password", "thiZ_is_v&ry_s3cret"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "settings.tcp-keepalive", "150"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "settings.maxclients", "5000"),
				),
			},
			{
				Config: `
					resource "scaleway_redis_cluster" "main" {
    					name = "test_redis_settings"
    					version = "6.2.6"
    					node_type = "MDB-BETA-M"
    					user_name = "my_initial_user"
    					password = "thiZ_is_v&ry_s3cret"
						settings = {
							"maxclients" = "2000"
						}
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayRedisExists(tt, "scaleway_redis_cluster.main"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "name", "test_redis_settings"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "version", "6.2.6"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "node_type", "MDB-BETA-M"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "user_name", "my_initial_user"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "password", "thiZ_is_v&ry_s3cret"),
					resource.TestCheckResourceAttr("scaleway_redis_cluster.main", "settings.maxclients", "2000"),
				),
			},
		},
	})
}

func testAccCheckScalewayRedisClusterDestroy(tt *TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_redis_cluster" {
				continue
			}

			redisAPI, zone, ID, err := redisAPIWithZoneAndID(tt.Meta, rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = redisAPI.GetCluster(&redis.GetClusterRequest{
				ClusterID: ID,
				Zone:      zone,
			})

			if err == nil {
				return fmt.Errorf("cluster (%s) still exists", rs.Primary.ID)
			}

			if !is404Error(err) {
				return err
			}
		}

		return nil
	}
}

func testAccCheckScalewayRedisExists(tt *TestTools, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		redisAPI, zone, ID, err := redisAPIWithZoneAndID(tt.Meta, rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = redisAPI.GetCluster(&redis.GetClusterRequest{
			ClusterID: ID,
			Zone:      zone,
		})

		if err != nil {
			return err
		}

		return nil
	}
}
