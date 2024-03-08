package instance_test

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	instanceSDK "github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/logging"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/instance"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
	"github.com/stretchr/testify/require"
	"sort"
	"testing"
)

func init() {
	resource.AddTestSweepers("scaleway_instance_security_group", &resource.Sweeper{
		Name: "scaleway_instance_security_group",
		F:    testSweepComputeInstanceSecurityGroup,
	})
}

func TestAccScalewayInstanceSecurityGroup_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	ipnetZero, err := types.ExpandIPNet("0.0.0.0/0")
	require.NoError(t, err)
	ipnetOne, err := types.ExpandIPNet("1.1.1.1")
	require.NoError(t, err)
	ipnetTest, err := types.ExpandIPNet("8.8.8.8")
	require.NoError(t, err)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceSecurityGroupDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_security_group" "base" {
						name = "sg-name"
						inbound_default_policy = "drop"
						
						inbound_rule {
							action = "accept"
							port = 80
							ip_range = "0.0.0.0/0"
						}
			
						inbound_rule {
							action = "accept"
							port = 22
							ip = "1.1.1.1"
						}
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceSecurityGroupExists(tt, "scaleway_instance_security_group.base"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "name", "sg-name"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_default_policy", "drop"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "outbound_default_policy", "accept"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.#", "2"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.0.action", "accept"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.0.protocol", "TCP"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.0.port", "80"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.0.ip_range", "0.0.0.0/0"),
					testAccCheckScalewayInstanceSecurityGroupRuleMatch(tt, "scaleway_instance_security_group.base", 0, &instanceSDK.SecurityGroupRule{
						Direction:    instanceSDK.SecurityGroupRuleDirectionInbound,
						IPRange:      ipnetZero,
						DestPortFrom: scw.Uint32Ptr(80),
						DestPortTo:   nil,
						Protocol:     instanceSDK.SecurityGroupRuleProtocolTCP,
						Action:       instanceSDK.SecurityGroupRuleActionAccept,
					}),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.1.action", "accept"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.1.protocol", "TCP"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.1.port", "22"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.1.ip", "1.1.1.1"),
					testAccCheckScalewayInstanceSecurityGroupRuleMatch(tt, "scaleway_instance_security_group.base", 1, &instanceSDK.SecurityGroupRule{
						Direction:    instanceSDK.SecurityGroupRuleDirectionInbound,
						IPRange:      ipnetOne,
						DestPortFrom: scw.Uint32Ptr(22),
						DestPortTo:   nil,
						Protocol:     instanceSDK.SecurityGroupRuleProtocolTCP,
						Action:       instanceSDK.SecurityGroupRuleActionAccept,
					}),
				),
			},
			{
				Config: `
					resource "scaleway_instance_security_group" "base" {
						name = "sg-name"
						inbound_default_policy = "accept"
						tags = [ "test-terraform" ]

						inbound_rule {
							action = "drop"
							port = 80
							ip = "8.8.8.8"
						}
			
						inbound_rule {
							action = "accept"
							port = 80
							ip_range = "0.0.0.0/0"
						}
			
						inbound_rule {
							action = "accept"
							port = 22
							ip = "1.1.1.1"
						}	
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceSecurityGroupExists(tt, "scaleway_instance_security_group.base"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "name", "sg-name"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "tags.0", "test-terraform"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_default_policy", "accept"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "outbound_default_policy", "accept"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.#", "3"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.0.action", "drop"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.0.protocol", "TCP"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.0.port", "80"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.0.ip", "8.8.8.8"),
					testAccCheckScalewayInstanceSecurityGroupRuleMatch(tt, "scaleway_instance_security_group.base", 0, &instanceSDK.SecurityGroupRule{
						Direction:    instanceSDK.SecurityGroupRuleDirectionInbound,
						IPRange:      ipnetTest,
						DestPortFrom: scw.Uint32Ptr(80),
						DestPortTo:   nil,
						Protocol:     instanceSDK.SecurityGroupRuleProtocolTCP,
						Action:       instanceSDK.SecurityGroupRuleActionDrop,
					}),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.1.action", "accept"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.1.protocol", "TCP"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.1.port", "80"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.1.ip_range", "0.0.0.0/0"),
					testAccCheckScalewayInstanceSecurityGroupRuleMatch(tt, "scaleway_instance_security_group.base", 1, &instanceSDK.SecurityGroupRule{
						Direction:    instanceSDK.SecurityGroupRuleDirectionInbound,
						IPRange:      ipnetZero,
						DestPortFrom: scw.Uint32Ptr(80),
						DestPortTo:   nil,
						Protocol:     instanceSDK.SecurityGroupRuleProtocolTCP,
						Action:       instanceSDK.SecurityGroupRuleActionAccept,
					}),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.2.action", "accept"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.2.protocol", "TCP"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.2.port", "22"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.2.ip", "1.1.1.1"),
					testAccCheckScalewayInstanceSecurityGroupRuleMatch(tt, "scaleway_instance_security_group.base", 2, &instanceSDK.SecurityGroupRule{
						Direction:    instanceSDK.SecurityGroupRuleDirectionInbound,
						IPRange:      ipnetOne,
						DestPortFrom: scw.Uint32Ptr(22),
						DestPortTo:   nil,
						Protocol:     instanceSDK.SecurityGroupRuleProtocolTCP,
						Action:       instanceSDK.SecurityGroupRuleActionAccept,
					}),
				),
			},
			{
				Config: `
					resource "scaleway_instance_security_group" "base" {
						name = "sg-name"
						inbound_default_policy = "accept"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "tags.#", "0"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.#", "0"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceSecurityGroup_ICMP(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	ipnetZero, err := types.ExpandIPNet("0.0.0.0/0")
	require.NoError(t, err)
	ipnetTest, err := types.ExpandIPNet("8.8.8.8")
	require.NoError(t, err)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceSecurityGroupDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_security_group" "base" {
						inbound_rule {
							action = "accept"
							port = 80
							ip_range = "0.0.0.0/0"
						}
						tags = [ "test-terraform" ]
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.0.action", "accept"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.0.protocol", "TCP"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.0.port", "80"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.0.ip_range", "0.0.0.0/0"),
					testAccCheckScalewayInstanceSecurityGroupRuleMatch(tt, "scaleway_instance_security_group.base", 0, &instanceSDK.SecurityGroupRule{
						Direction:    instanceSDK.SecurityGroupRuleDirectionInbound,
						IPRange:      ipnetZero,
						DestPortFrom: scw.Uint32Ptr(80),
						DestPortTo:   nil,
						Protocol:     instanceSDK.SecurityGroupRuleProtocolTCP,
						Action:       instanceSDK.SecurityGroupRuleActionAccept,
					}),
				),
			},
			{
				Config: `
					resource "scaleway_instance_security_group" "base" {
						inbound_rule {
							action = "drop"
							protocol = "ICMP"
							ip = "8.8.8.8"
						}
						tags = [ "test-terraform" ]
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.0.action", "drop"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.0.protocol", "ICMP"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.0.port", "0"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.0.ip", "8.8.8.8"),
					testAccCheckScalewayInstanceSecurityGroupRuleMatch(tt, "scaleway_instance_security_group.base", 0, &instanceSDK.SecurityGroupRule{
						Direction:    instanceSDK.SecurityGroupRuleDirectionInbound,
						IPRange:      ipnetTest,
						DestPortFrom: nil,
						DestPortTo:   nil,
						Protocol:     instanceSDK.SecurityGroupRuleProtocolICMP,
						Action:       instanceSDK.SecurityGroupRuleActionDrop,
					}),
				),
			},
		},
	})
}

func TestAccScalewayInstanceSecurityGroup_ANY(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceSecurityGroupDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					locals {
						ips_to_ban = ["1.1.1.1", "2.2.2.2", "3.3.3.3"]
					}
					
					resource "scaleway_instance_security_group" "ban_ips" {
						tags = [ "test-terraform" ]
						inbound_default_policy = "accept"
					
						dynamic "inbound_rule" {
						for_each = local.ips_to_ban
					
						content {
						  action = "drop"
						  protocol  = "ANY"
						  ip = inbound_rule.value
						}
					  }
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("scaleway_instance_security_group.ban_ips", "inbound_rule.0.action", "drop"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.ban_ips", "inbound_rule.0.protocol", "ANY"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.ban_ips", "inbound_rule.0.ip", "1.1.1.1"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.ban_ips", "inbound_rule.1.action", "drop"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.ban_ips", "inbound_rule.1.protocol", "ANY"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.ban_ips", "inbound_rule.1.ip", "2.2.2.2"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.ban_ips", "inbound_rule.2.action", "drop"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.ban_ips", "inbound_rule.2.protocol", "ANY"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.ban_ips", "inbound_rule.2.ip", "3.3.3.3"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceSecurityGroup_WithNoPort(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	ipnetZero, err := types.ExpandIPNet("0.0.0.0/0")
	require.NoError(t, err)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceSecurityGroupDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_security_group" "base" {
						inbound_rule {
							action = "accept"
							ip_range = "0.0.0.0/0"
						}
						tags = [ "test-terraform" ]
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceSecurityGroupRuleMatch(tt, "scaleway_instance_security_group.base", 0, &instanceSDK.SecurityGroupRule{
						Direction:    instanceSDK.SecurityGroupRuleDirectionInbound,
						IPRange:      ipnetZero,
						DestPortFrom: nil,
						DestPortTo:   nil,
						Protocol:     instanceSDK.SecurityGroupRuleProtocolTCP,
						Action:       instanceSDK.SecurityGroupRuleActionAccept,
					}),
				),
			},
		},
	})
}

func TestAccScalewayInstanceSecurityGroup_RemovePort(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	ipnetZero, err := types.ExpandIPNet("0.0.0.0/0")
	require.NoError(t, err)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceSecurityGroupDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_security_group" "base" {
						tags = [ "test-terraform" ]
						inbound_rule {
							action = "accept"
							ip_range = "0.0.0.0/0"
							port = 22
						}
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceSecurityGroupRuleMatch(tt, "scaleway_instance_security_group.base", 0, &instanceSDK.SecurityGroupRule{
						Direction:    instanceSDK.SecurityGroupRuleDirectionInbound,
						IPRange:      ipnetZero,
						DestPortFrom: scw.Uint32Ptr(22),
						DestPortTo:   nil,
						Protocol:     instanceSDK.SecurityGroupRuleProtocolTCP,
						Action:       instanceSDK.SecurityGroupRuleActionAccept,
					}),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "tags.#", "1"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "tags.0", "test-terraform"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_security_group" "base" {
						inbound_rule {
							action = "accept"
							ip_range = "0.0.0.0/0"
						}
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceSecurityGroupRuleMatch(tt, "scaleway_instance_security_group.base", 0, &instanceSDK.SecurityGroupRule{
						Direction:    instanceSDK.SecurityGroupRuleDirectionInbound,
						IPRange:      ipnetZero,
						DestPortFrom: nil,
						DestPortTo:   nil,
						Protocol:     instanceSDK.SecurityGroupRuleProtocolTCP,
						Action:       instanceSDK.SecurityGroupRuleActionAccept,
					}),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "tags.#", "0"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceSecurityGroup_WithPortRange(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceSecurityGroupDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_security_group" "base" {
						tags = [ "test-terraform" ]
						inbound_rule {
							action = "accept"
							port_range = "1-1024"
							ip = "8.8.8.8"
						}
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.0.port_range", "1-1024"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_security_group" "base" {
						tags = [ "test-terraform" ]
						inbound_rule {
							action = "accept"
							port = "22"
							ip = "8.8.8.8"
						}
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.0.port", "22"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_security_group" "base" {
						tags = [ "test-terraform" ]
						inbound_rule {
							action = "accept"
							port_range = "1-1024"
							ip = "8.8.8.8"
						}
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "inbound_rule.0.port_range", "1-1024"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceSecurityGroup_Tags(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceSecurityGroupDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_security_group" "main" {
						tags = [ "foo", "bar" ]
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("scaleway_instance_security_group.main", "tags.0", "foo"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.main", "tags.1", "bar"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_security_group" "main" {
						tags = [ "foo", "buzz" ]
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("scaleway_instance_security_group.main", "tags.0", "foo"),
					resource.TestCheckResourceAttr("scaleway_instance_security_group.main", "tags.1", "buzz"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_security_group" "main" {
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("scaleway_instance_security_group.main", "tags.#", "0"),
				),
			},
		},
	})
}

func testAccCheckScalewayInstanceSecurityGroupRuleMatch(tt *tests.TestTools, name string, index int, expected *instanceSDK.SecurityGroupRule) resource.TestCheckFunc {
	return testAccCheckScalewayInstanceSecurityGroupRuleIs(tt, name, expected.Direction, index, func(actual *instanceSDK.SecurityGroupRule) error {
		if ok, _ := instance.SecurityGroupRuleEquals(expected, actual); !ok {
			return fmt.Errorf("security group does not match %v, %v", actual, expected)
		}
		return nil
	})
}

func testAccCheckScalewayInstanceSecurityGroupRuleIs(tt *tests.TestTools, name string, direction instanceSDK.SecurityGroupRuleDirection, index int, test func(rule *instanceSDK.SecurityGroupRule) error) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("not found: %s", name)
		}

		instanceAPI, zone, ID, err := instance.InstanceAPIWithZoneAndID(tt.GetMeta(), rs.Primary.ID)
		if err != nil {
			return err
		}

		resRules, err := instanceAPI.ListSecurityGroupRules(&instanceSDK.ListSecurityGroupRulesRequest{
			SecurityGroupID: ID,
			Zone:            zone,
		}, scw.WithAllPages())
		if err != nil {
			return err
		}
		sort.Slice(resRules.Rules, func(i, j int) bool {
			return resRules.Rules[i].Position < resRules.Rules[j].Position
		})
		apiRules := map[instanceSDK.SecurityGroupRuleDirection][]*instanceSDK.SecurityGroupRule{
			instanceSDK.SecurityGroupRuleDirectionInbound:  {},
			instanceSDK.SecurityGroupRuleDirectionOutbound: {},
		}

		for _, apiRule := range resRules.Rules {
			if apiRule.Editable == false {
				continue
			}
			apiRules[apiRule.Direction] = append(apiRules[apiRule.Direction], apiRule)
		}

		return test(apiRules[direction][index])
	}
}

func testAccCheckScalewayInstanceSecurityGroupExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		zone, ID, err := locality.ParseZonedID(rs.Primary.ID)
		if err != nil {
			return err
		}

		instanceAPI := instanceSDK.NewAPI(tt.GetMeta().GetScwClient())
		_, err = instanceAPI.GetSecurityGroup(&instanceSDK.GetSecurityGroupRequest{
			SecurityGroupID: ID,
			Zone:            zone,
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckScalewayInstanceSecurityGroupDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		instanceAPI := instanceSDK.NewAPI(tt.GetMeta().GetScwClient())
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_instance_security_group" {
				continue
			}

			zone, ID, err := locality.ParseZonedID(rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = instanceAPI.GetSecurityGroup(&instanceSDK.GetSecurityGroupRequest{
				Zone:            zone,
				SecurityGroupID: ID,
			})

			// If no error resource still exist
			if err == nil {
				return fmt.Errorf("security group (%s) still exists", rs.Primary.ID)
			}

			// Unexpected api error we return it
			if !http_errors.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}

func testSweepComputeInstanceSecurityGroup(_ string) error {
	return tests.SweepZones(scw.AllZones, func(scwClient *scw.Client, zone scw.Zone) error {
		instanceAPI := instanceSDK.NewAPI(scwClient)
		logging.L.Debugf("sweeper: destroying the security groups in (%s)", zone)

		listResp, err := instanceAPI.ListSecurityGroups(&instanceSDK.ListSecurityGroupsRequest{
			Zone: zone,
		}, scw.WithAllPages())
		if err != nil {
			logging.L.Warningf("error listing security groups in sweeper: %s", err)
			return nil
		}

		for _, securityGroup := range listResp.SecurityGroups {
			// Can't delete default security group.
			if securityGroup.ProjectDefault {
				continue
			}
			err = instanceAPI.DeleteSecurityGroup(&instanceSDK.DeleteSecurityGroupRequest{
				Zone:            zone,
				SecurityGroupID: securityGroup.ID,
			})
			if err != nil {
				return fmt.Errorf("error deleting security groups in sweeper: %s", err)
			}
		}

		return nil
	})
}

func TestAccScalewayInstanceSecurityGroup_EnableDefaultSecurity(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceSecurityGroupDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_security_group" "base" {
						tags = [ "test-terraform" ]
						enable_default_security = false
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "enable_default_security", "false"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_security_group" "base" {
						tags = [ "test-terraform" ]
						enable_default_security = true
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("scaleway_instance_security_group.base", "enable_default_security", "true"),
				),
			},
		},
	})
}
