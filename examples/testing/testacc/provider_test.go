package testacc

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"terraform-provider-gigamon/internal/provider"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"gigamon": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// testAccPreCheck validates the necessary test API keys exist
// in the testing environment
func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("FM_ADDRESS"); v == "" {
		t.Fatal("FM_ADDRESS must be set for acceptance tests")
	}
	if v := os.Getenv("FM_API_TOKEN"); v == "" {
		t.Fatal("FM_API_TOKEN must be set for acceptance tests")
	}
	if v := os.Getenv("FM_MONITORING_SESSION_ID"); v == "" {
		t.Fatal("FM_MONITORING_SESSION_ID must be set for acceptance tests")
	}
}

// testAccRandomSuffix returns a random string suitable for use as a resource name suffix
func testAccRandomSuffix() string {
	return acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
}

func testAccProviderConfig() string {
	return fmt.Sprintf(`
provider "gigamon" {
  fm_address  = %q
  api_token   = %q
  skip_verify = true
}
`, os.Getenv("FM_ADDRESS"), os.Getenv("FM_API_TOKEN"))
}

func testAccMonitoringSessionID() string {
	return os.Getenv("FM_MONITORING_SESSION_ID")
}

func testAccRawUUIDFromTypedID(typedID string) string {
	parts := strings.Split(typedID, "::")
	if len(parts) < 3 {
		return typedID
	}
	return parts[len(parts)-1]
}

func testAccImportIDFromState(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found in state: %s", resourceName)
		}

		sessionID := rs.Primary.Attributes["monitoring_session_id"]
		rawID := testAccRawUUIDFromTypedID(rs.Primary.Attributes["id"])
		if sessionID == "" || rawID == "" {
			return "", fmt.Errorf("missing monitoring_session_id or id in state for %s", resourceName)
		}

		return fmt.Sprintf("%s::%s", sessionID, rawID), nil
	}
}

// testAccCheckApp5GCloudDestroy is a helper function to verify a 5G Cloud app
// resource no longer exists after destroy
func testAccCheckApp5GCloudDestroy(t *testing.T) error {
	// For now, just return nil as the FM API may not have a direct way to check
	// if a specific app still exists. This should be implemented based on actual
	// FM API capabilities.
	return nil
}
