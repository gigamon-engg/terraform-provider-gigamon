package testacc

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAppVizResource_Basic(t *testing.T) {
	resourceName := "gigamon_app_viz.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAppVizResourceConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "monitoring_session_id", testAccMonitoringSessionID()),
					resource.TestCheckResourceAttr(resourceName, "alias", "app_viz1"),
					resource.TestCheckResourceAttr(resourceName, "description", ""),
					resource.TestCheckResourceAttr(resourceName, "action", "true"),
					resource.TestCheckResourceAttr(resourceName, "mgmt_interface", "internal"),
					resource.TestCheckResourceAttr(resourceName, "exporter_config.monitor.timeout", "300"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccImportIDFromState(resourceName),
				ImportStateVerify: true,
			},
			{
				Config: testAccAppVizResourceConfigUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "alias", "app_viz2"),
					resource.TestCheckResourceAttr(resourceName, "description", "updated app viz config"),
					resource.TestCheckResourceAttr(resourceName, "action", "false"),
					resource.TestCheckResourceAttr(resourceName, "mgmt_interface", "external"),
					resource.TestCheckResourceAttr(resourceName, "exporter_config.monitor.timeout", "600"),
				),
			},
		},
	})
}

func TestAccAppVizResource_RequiredFieldMissing(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccAppVizResourceConfigMissingRequired(),
				ExpectError: regexp.MustCompile("Missing required attribute"),
			},
		},
	})
}

func testAccAppVizResourceConfigBasic() string {
	return fmt.Sprintf(`
%s

resource "gigamon_app_viz" "test" {
  monitoring_session_id = %q
  alias                 = "app_viz1"
  description           = ""
  action                = true
  mgmt_interface        = "internal"

  exporter_config = {
    monitor = {
      timeout = 300
    }
  }
}
`, testAccProviderConfig(), testAccMonitoringSessionID())
}

func testAccAppVizResourceConfigUpdated() string {
	return fmt.Sprintf(`
%s

resource "gigamon_app_viz" "test" {
  monitoring_session_id = %q
  alias                 = "app_viz2"
  description           = "updated app viz config"
  action                = false
  mgmt_interface        = "external"

  exporter_config = {
    monitor = {
      timeout = 600
    }
  }
}
`, testAccProviderConfig(), testAccMonitoringSessionID())
}

func testAccAppVizResourceConfigMissingRequired() string {
	return fmt.Sprintf(`
%s

resource "gigamon_app_viz" "test" {
  alias = "app_viz_missing_ms"
}
`, testAccProviderConfig())
}
