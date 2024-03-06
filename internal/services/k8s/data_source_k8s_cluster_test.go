package k8s

import (
	"fmt"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccScalewayDataSourceK8SCluster_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	clusterName := "tf-cluster"
	version := testAccScalewayK8SClusterGetLatestK8SVersion(tt)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckScalewayK8SPoolDestroy(tt, "scaleway_k8s_pool.default"),
			testAccCheckScalewayK8SClusterDestroy(tt),
			testAccCheckScalewayVPCPrivateNetworkDestroy(tt),
		),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_vpc_private_network" "main" {
						name = "test-data-source-cluster"
					}

					resource "scaleway_k8s_cluster" "main" {
					  	name 	= "%s"
						version = "%s"
						cni     = "cilium"
					  	tags    = [ "terraform-test", "data_scaleway_k8s_cluster", "basic" ]
						delete_additional_resources = true
						private_network_id = scaleway_vpc_private_network.main.id
					}

					resource "scaleway_k8s_pool" "default" {
						name = "default"
						cluster_id = "${scaleway_k8s_cluster.main.id}"
						node_type = "gp1_xs"
						autohealing = true
						autoscaling = true
						size = 1
					}
					
					data "scaleway_k8s_cluster" "prod" {
					  	name = "${scaleway_k8s_cluster.main.name}"
					}
					
					data "scaleway_k8s_cluster" "stg" {
					  	cluster_id = "${scaleway_k8s_cluster.main.id}"
					}`, clusterName, version),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayK8SClusterExists(tt, "data.scaleway_k8s_cluster.prod"),
					resource.TestCheckResourceAttr("data.scaleway_k8s_cluster.prod", "name", clusterName),
					testAccCheckScalewayK8SClusterExists(tt, "data.scaleway_k8s_cluster.stg"),
					resource.TestCheckResourceAttr("data.scaleway_k8s_cluster.stg", "name", clusterName),
				),
			},
		},
	})
}
