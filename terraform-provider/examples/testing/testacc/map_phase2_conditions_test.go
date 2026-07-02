package testacc

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTrafficMapPhase2Conditions_Basic(t *testing.T) {
	resourceName := "gigamon_traffic_map.phase2"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTrafficMapPhase2ConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "monitoring_session_id", testAccMonitoringSessionID()),
					resource.TestCheckResourceAttr(resourceName, "rule_sets.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule_sets.0.pass_rules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule_sets.0.pass_rules.0.gtp_teid.teid_min", "1000"),
					resource.TestCheckResourceAttr(resourceName, "rule_sets.0.pass_rules.0.port_destination.port_min", "80"),
					resource.TestCheckResourceAttr(resourceName, "rule_sets.0.pass_rules.0.vlan.vlan_min", "100"),
				),
			},
			{
				Config: testAccTrafficMapPhase2ConfigUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "rule_sets.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule_sets.0.pass_rules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule_sets.0.pass_rules.0.host_name.host_prefix", "internal"),
					resource.TestCheckResourceAttr(resourceName, "rule_sets.0.pass_rules.0.vxlan_id.vxlan_min", "5000"),
					resource.TestCheckResourceAttr(resourceName, "rule_sets.0.pass_rules.0.vntag_src_vif_id.vif_min", "10"),
				),
			},
		},
	})
}

func TestAccTrafficMapPhase2Conditions_Validation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccTrafficMapPhase2ConfigInvalidVlan(),
				ExpectError: regexp.MustCompile("Attribute vlan.vlan_min value must be between 0 and 4095"),
			},
		},
	})
}

func testAccTrafficMapPhase2ConfigBasic() string {
	return fmt.Sprintf(`
%s

resource "gigamon_traffic_map" "phase2" {
  monitoring_session_id = %q
  name                  = "tf-acc-phase2-map-basic"
  description           = "Phase2 basic condition coverage"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 2

      pass_rules = [
        {
          rule_id = 1

          gtp_teid = {
            teid_min    = 1000
            teid_max    = 2000
            teid_subset = "all"
          }

          port_destination = {
            port_min = 80
            port_max = 443
          }

          port_source = {
            port_min = 1024
            port_max = 65535
          }

          vlan = {
            vlan_min    = 100
            vlan_max    = 200
            vlan_subset = "all"
          }

          tcp_control = {
            flags = "SYN,ACK"
          }

          ipv6_flow_label = {
            label_min = 10
            label_max = 100
          }

          ipv6_next_header = {
            header_min    = 6
            header_max    = 17
            header_subset = "all"
          }

          mpls_label = {
            label_min = 16
            label_max = 104857
          }
        }
      ]
    }
  ]
}
`, testAccProviderConfig(), testAccMonitoringSessionID())
}

func testAccTrafficMapPhase2ConfigUpdated() string {
	return fmt.Sprintf(`
%s

resource "gigamon_traffic_map" "phase2" {
  monitoring_session_id = %q
  name                  = "tf-acc-phase2-map-updated"
  description           = "Phase2 updated condition coverage"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 2

      pass_rules = [
        {
          rule_id = 1

          host_name = {
            host_prefix = "internal"
          }

          vxlan_id = {
            vxlan_min    = 5000
            vxlan_max    = 6000
            vxlan_subset = "all"
          }

          vntag_dst_vif_id = {
            vif_min = 20
            vif_max = 40
          }

          vntag_src_vif_id = {
            vif_min = 10
            vif_max = 30
          }

          vntag_vif_list_id = {
            list_id_min = 1
            list_id_max = 10
          }
        }
      ]
    }
  ]
}
`, testAccProviderConfig(), testAccMonitoringSessionID())
}

func testAccTrafficMapPhase2ConfigInvalidVlan() string {
	return fmt.Sprintf(`
%s

resource "gigamon_traffic_map" "phase2" {
  monitoring_session_id = %q
  name                  = "tf-acc-phase2-map-invalid"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 2

      pass_rules = [
        {
          rule_id = 1
          vlan = {
            vlan_min = 5000
            vlan_max = 5001
          }
        }
      ]
    }
  ]
}
`, testAccProviderConfig(), testAccMonitoringSessionID())
}
