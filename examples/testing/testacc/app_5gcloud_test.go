package testacc

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccApp5GCloudResource_Basic(t *testing.T) {
	resourceName := "gigamon_app_5gcloud.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccApp5GCloudResourceConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "monitoring_session_id", testAccMonitoringSessionID()),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "profile", "default"),
					resource.TestCheckResourceAttr(resourceName, "filter_config.protocol_filter", "udp"),
					resource.TestCheckResourceAttr(resourceName, "filter_config.port_range.min", "53"),
					resource.TestCheckResourceAttr(resourceName, "filter_config.port_range.max", "53"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccImportIDFromState(resourceName),
				ImportStateVerify: true,
			},
			{
				Config: testAccApp5GCloudResourceConfigUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "profile", "custom"),
					resource.TestCheckResourceAttr(resourceName, "filter_config.protocol_filter", "tcp"),
					resource.TestCheckResourceAttr(resourceName, "filter_config.port_range.min", "443"),
					resource.TestCheckResourceAttr(resourceName, "filter_config.port_range.max", "443"),
					resource.TestCheckResourceAttr(resourceName, "export_config.interval", "120"),
				),
			},
		},
	})
}

func TestAccApp5GCloudResource_RequiredFieldMissing(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccApp5GCloudResourceConfigMissingSession(),
				ExpectError: regexp.MustCompile("Missing required attribute"),
			},
		},
	})
}

func testAccApp5GCloudResourceConfigBasic() string {
	return fmt.Sprintf(`
%s

resource "gigamon_app_5gcloud" "test" {
  monitoring_session_id = %q
  enabled               = true
  profile               = "default"

  filter_config = {
    protocol_filter = "udp"
    port_range = {
      min = 53
      max = 53
    }
  }

  export_config = {
    format   = "netflow"
    interval = 60
  }
}
`, testAccProviderConfig(), testAccMonitoringSessionID())
}

func testAccApp5GCloudResourceConfigUpdated() string {
	return fmt.Sprintf(`
%s

resource "gigamon_app_5gcloud" "test" {
  monitoring_session_id = %q
  enabled               = false
  profile               = "custom"

  filter_config = {
    protocol_filter = "tcp"
    port_range = {
      min = 443
      max = 443
    }
  }

  export_config = {
    format   = "netflow"
    interval = 120
  }
}
`, testAccProviderConfig(), testAccMonitoringSessionID())
}

func testAccApp5GCloudResourceConfigMissingSession() string {
	return fmt.Sprintf(`
%s

resource "gigamon_app_5gcloud" "test" {
  enabled = true
}
`, testAccProviderConfig())
}
