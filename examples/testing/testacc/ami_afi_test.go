package testacc

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAmiResource_Basic(t *testing.T) {
	resourceName := "gigamon_app_ami.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAmiResourceConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "monitoring_session_id", testAccMonitoringSessionID()),
					resource.TestCheckResourceAttr(resourceName, "alias", "ami-test"),
					resource.TestCheckResourceAttr(resourceName, "app_metadata.flow_behavior", "bidir"),
					resource.TestCheckResourceAttr(resourceName, "app_metadata.multi_collect", "true"),
					resource.TestCheckResourceAttr(resourceName, "app_metadata.aggregate_mode", "false"),
					resource.TestCheckResourceAttr(resourceName, "app_metadata.timeout.0.idle", "300"),
					resource.TestCheckResourceAttr(resourceName, "app_metadata.exporters.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "app_metadata.exporters.0.aep_id", "2"),
					resource.TestCheckResourceAttr(resourceName, "app_metadata.exporters.0.name", "ami-exporter-2"),
					resource.TestCheckResourceAttr(resourceName, "app_metadata.exporters.0.exporter_config.type", "cef"),
					resource.TestCheckResourceAttr(resourceName, "app_metadata.exporters.0.exporter_config.cef.record_type", "segregated"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccImportIDFromState(resourceName),
				ImportStateVerify: true,
			},
			{
				Config: testAccAmiResourceConfigUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "alias", "ami-test-updated"),
					resource.TestCheckResourceAttr(resourceName, "description", "updated AMI acceptance coverage"),
					resource.TestCheckResourceAttr(resourceName, "app_metadata.flow_behavior", "bidirectional"),
					resource.TestCheckResourceAttr(resourceName, "app_metadata.multi_collect", "false"),
					resource.TestCheckResourceAttr(resourceName, "app_metadata.aggregate_mode", "true"),
					resource.TestCheckResourceAttr(resourceName, "app_metadata.timeout.0.idle", "600"),
					resource.TestCheckResourceAttr(resourceName, "app_metadata.exporters.0.exporter_config.cef.record_type", "aggregated"),
				),
			},
		},
	})
}

func TestAccAmiResource_RequiredFieldMissing(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccAmiResourceConfigMissingSession(),
				ExpectError: regexp.MustCompile("Missing required attribute"),
			},
		},
	})
}

func TestAccTrafficMapAfiResource_Basic(t *testing.T) {
	resourceName := "gigamon_traffic_map.afi"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAfiResourceConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "monitoring_session_id", testAccMonitoringSessionID()),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-afi-map-basic"),
					resource.TestCheckResourceAttr(resourceName, "rule_sets.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule_sets.0.pass_rules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule_sets.0.pass_rules.0.ip_version.ip_version", "v4"),
					resource.TestCheckResourceAttr(resourceName, "asf.asf_profile_config.timeout", "15"),
					resource.TestCheckResourceAttr(resourceName, "asf.asf_profile_config.packet_count", "30"),
					resource.TestCheckResourceAttr(resourceName, "asf.asf_profile_config.bidi", "true"),
					resource.TestCheckResourceAttr(resourceName, "asf.asf_profile_config.buffering.enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "asf.asf_profile_config.buffering.protocol", "tcpUdp"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccImportIDFromState(resourceName),
				ImportStateVerify: true,
			},
			{
				Config: testAccAfiResourceConfigUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-afi-map-updated"),
					resource.TestCheckResourceAttr(resourceName, "description", "updated AFI acceptance coverage"),
					resource.TestCheckResourceAttr(resourceName, "rule_sets.0.pass_rules.0.ip_version.ip_version", "v6"),
					resource.TestCheckResourceAttr(resourceName, "asf.asf_profile_config.timeout", "20"),
					resource.TestCheckResourceAttr(resourceName, "asf.asf_profile_config.packet_count", "40"),
					resource.TestCheckResourceAttr(resourceName, "asf.asf_profile_config.buffering.buffer_count_before_match", "10"),
				),
			},
		},
	})
}

func TestAccTrafficMapAfiResource_Validation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccAfiResourceConfigInvalidASF(),
				ExpectError: regexp.MustCompile("asf.asf_profile_config must be set when asf is provided"),
			},
		},
	})
}

func testAccAmiResourceConfigBasic() string {
	return fmt.Sprintf(`
%s

resource "gigamon_app_ami" "test" {
  monitoring_session_id = %q
  alias                 = "ami-test"

  app_metadata = {
    flow_behavior    = "bidir"
    multi_collect    = true
    aggregate_mode   = false
    observ_domain_id = 0

    timeout = {
      idle = 300
    }

    exporters = [
      {
        aep_id = 2
        name   = "ami-exporter-2"

        exporter_config = {
          type = "cef"

          cef = {
            active_timeout   = 60
            inactive_timeout = 15
            record_type      = "segregated"
          }
        }
      }
    ]
  }
}
`, testAccProviderConfig(), testAccMonitoringSessionID())
}

func testAccAmiResourceConfigUpdated() string {
	return fmt.Sprintf(`
%s

resource "gigamon_app_ami" "test" {
  monitoring_session_id = %q
  alias                 = "ami-test-updated"
  description           = "updated AMI acceptance coverage"

  app_metadata = {
    flow_behavior    = "bidirectional"
    multi_collect    = false
    aggregate_mode   = true
    observ_domain_id = 0

    timeout = {
      idle = 600
    }

    exporters = [
      {
        aep_id = 2
        name   = "ami-exporter-2"

        exporter_config = {
          type = "cef"

          cef = {
            active_timeout   = 120
            inactive_timeout = 30
            record_type      = "aggregated"
          }
        }
      }
    ]
  }
}
`, testAccProviderConfig(), testAccMonitoringSessionID())
}

func testAccAmiResourceConfigMissingSession() string {
	return fmt.Sprintf(`
%s

resource "gigamon_app_ami" "test" {
  alias = "ami-missing-session"
}
`, testAccProviderConfig())
}

func testAccAfiResourceConfigBasic() string {
	return fmt.Sprintf(`
%s

resource "gigamon_traffic_map" "afi" {
  monitoring_session_id = %q
  name                  = "tf-acc-afi-map-basic"
  description           = "AFI acceptance coverage"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 2

      pass_rules = [
        {
          rule_id = 1
          ip_version = {
            ip_version = "v4"
          }
        }
      ]
    }
  ]

  asf = {
    asf_profile_config = {
      session_fields = [
        {
          pos  = 2
          type = "fiveTuple"
        }
      ]

      timeout      = 15
      packet_count = 30
      bidi         = true

      buffering = {
        enabled                   = true
        protocol                  = "tcpUdp"
        buffer_count_before_match = 20
      }
    }
  }
}
`, testAccProviderConfig(), testAccMonitoringSessionID())
}

func testAccAfiResourceConfigUpdated() string {
	return fmt.Sprintf(`
%s

resource "gigamon_traffic_map" "afi" {
  monitoring_session_id = %q
  name                  = "tf-acc-afi-map-updated"
  description           = "updated AFI acceptance coverage"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 2

      pass_rules = [
        {
          rule_id = 1
          ip_version = {
            ip_version = "v6"
          }
        }
      ]
    }
  ]

  asf = {
    asf_profile_config = {
      session_fields = [
        {
          pos  = 2
          type = "fiveTuple"
        }
      ]

      timeout      = 20
      packet_count = 40
      bidi         = false

      buffering = {
        enabled                   = true
        protocol                  = "tcpUdp"
        buffer_count_before_match = 10
      }
    }
  }
}
`, testAccProviderConfig(), testAccMonitoringSessionID())
}

func testAccAfiResourceConfigInvalidASF() string {
	return fmt.Sprintf(`
%s

resource "gigamon_traffic_map" "afi" {
  monitoring_session_id = %q
  name                  = "tf-acc-afi-map-invalid"

  asf = {}
}
`, testAccProviderConfig(), testAccMonitoringSessionID())
}