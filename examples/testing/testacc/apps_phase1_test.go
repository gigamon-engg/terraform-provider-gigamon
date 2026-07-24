package testacc

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

type phase1AppCase struct {
	name            string
	resourceType    string
	basicBody       string
	updatedBody     string
	expectedBasic   map[string]string
	expectedUpdated map[string]string
}

func TestAccPhase1Apps_BasicUpdateImport(t *testing.T) {
	tests := []phase1AppCase{
		{
			name:         "5GEvp",
			resourceType: "gigamon_app_5gevp",
			basicBody: `
  monitoring_session_id = "${local.ms_id}"
  network_mode          = "standalone"

  traffic_optimization = {
    enabled = true
    level   = 6
  }

  performance_tuning = {
    cache_size   = 256
    buffer_depth = 1000
  }
`,
			updatedBody: `
  monitoring_session_id = "${local.ms_id}"
  network_mode          = "distributed"

  traffic_optimization = {
    enabled = false
    level   = 8
  }

  performance_tuning = {
    cache_size   = 512
    buffer_depth = 1500
  }
`,
			expectedBasic: map[string]string{
				"network_mode":                  "standalone",
				"traffic_optimization.enabled":  "true",
				"traffic_optimization.level":    "6",
				"performance_tuning.cache_size": "256",
			},
			expectedUpdated: map[string]string{
				"network_mode":                  "distributed",
				"traffic_optimization.enabled":  "false",
				"traffic_optimization.level":    "8",
				"performance_tuning.cache_size": "512",
			},
		},
		{
			name:         "5GSBI",
			resourceType: "gigamon_app_5gsbi",
			basicBody: `
  monitoring_session_id = "${local.ms_id}"
  sbi_mode              = "nrf"
  protocol_handlers     = ["http", "https"]

  authentication = {
    enabled   = true
    cert_path = "/tmp/cert.pem"
    key_path  = "/tmp/key.pem"
  }
`,
			updatedBody: `
  monitoring_session_id = "${local.ms_id}"
  sbi_mode              = "amf"
  protocol_handlers     = ["grpc"]

  authentication = {
    enabled = false
  }
`,
			expectedBasic: map[string]string{
				"sbi_mode":               "nrf",
				"protocol_handlers.#":    "2",
				"authentication.enabled": "true",
			},
			expectedUpdated: map[string]string{
				"sbi_mode":               "amf",
				"protocol_handlers.#":    "1",
				"authentication.enabled": "false",
			},
		},
		{
			name:         "SBIPoe",
			resourceType: "gigamon_app_sbipoe",
			basicBody: `
  monitoring_session_id = "${local.ms_id}"
  poe_mode              = "stateful"
  session_tracking      = true

  analysis_rules = [
    {
      rule_id  = "allow-1"
      priority = 10
      action   = "allow"
    }
  ]
`,
			updatedBody: `
  monitoring_session_id = "${local.ms_id}"
  poe_mode              = "stateless"
  session_tracking      = false

  analysis_rules = [
    {
      rule_id  = "deny-1"
      priority = 20
      action   = "deny"
    }
  ]
`,
			expectedBasic: map[string]string{
				"poe_mode":                "stateful",
				"session_tracking":        "true",
				"analysis_rules.#":        "1",
				"analysis_rules.0.action": "allow",
			},
			expectedUpdated: map[string]string{
				"poe_mode":                "stateless",
				"session_tracking":        "false",
				"analysis_rules.#":        "1",
				"analysis_rules.0.action": "deny",
			},
		},
		{
			name:         "SSLDecrypt",
			resourceType: "gigamon_app_ssl_decrypt",
			basicBody: `
  monitoring_session_id = "${local.ms_id}"
  decryption_mode       = "full"
  cipher_suite          = "intermediate"

  performance = {
    max_connections = 20000
    timeout         = 60
  }
`,
			updatedBody: `
  monitoring_session_id = "${local.ms_id}"
  decryption_mode       = "bypass"
  cipher_suite          = "modern"

  performance = {
    max_connections = 50000
    timeout         = 120
  }
`,
			expectedBasic: map[string]string{
				"decryption_mode":             "full",
				"cipher_suite":                "intermediate",
				"performance.max_connections": "20000",
			},
			expectedUpdated: map[string]string{
				"decryption_mode":             "bypass",
				"cipher_suite":                "modern",
				"performance.max_connections": "50000",
			},
		},
		{
			name:         "GVHTTP2",
			resourceType: "gigamon_app_gvhttp2",
			basicBody: `
  monitoring_session_id = "${local.ms_id}"
  http2_mode            = "enabled"

  protocol_config = {
    stream_multiplexing = true
    server_push         = false
  }

  compression = {
    enabled = true
    level   = 6
  }

  flow_control = {
    window_size         = 65535
    initial_window_size = 65535
  }
`,
			updatedBody: `
  monitoring_session_id = "${local.ms_id}"
  http2_mode            = "disabled"

  protocol_config = {
    stream_multiplexing = false
    server_push         = false
  }

  compression = {
    enabled = false
    level   = 3
  }

  flow_control = {
    window_size         = 131072
    initial_window_size = 131072
  }
`,
			expectedBasic: map[string]string{
				"http2_mode":                       "enabled",
				"compression.enabled":              "true",
				"flow_control.initial_window_size": "65535",
			},
			expectedUpdated: map[string]string{
				"http2_mode":                       "disabled",
				"compression.enabled":              "false",
				"flow_control.initial_window_size": "131072",
			},
		},
		{
			name:         "PCapNG",
			resourceType: "gigamon_app_pcapng",
			basicBody: `
  monitoring_session_id = "${local.ms_id}"
  capture_mode          = "continuous"

  packet_filter = {
    bpf_syntax  = "tcp"
    source_ip   = "10.0.0.1"
    dest_ip     = "10.0.0.2"
    vlan_filter = [100, 200]
  }

  output_config = {
    file_path     = "/tmp/cap-basic.pcapng"
    max_file_size = 200
    rotation      = true
    compression   = "gzip"
  }

  performance = {
    buffer_size    = 128
    thread_count   = 4
    packet_snaplen = 2048
  }
`,
			updatedBody: `
  monitoring_session_id = "${local.ms_id}"
  capture_mode          = "triggered"

  packet_filter = {
    bpf_syntax  = "udp"
    source_ip   = "10.1.0.1"
    dest_ip     = "10.1.0.2"
    vlan_filter = [300]
  }

  output_config = {
    file_path     = "/tmp/cap-updated.pcapng"
    max_file_size = 500
    rotation      = false
    compression   = "xz"
  }

  performance = {
    buffer_size    = 256
    thread_count   = 8
    packet_snaplen = 4096
  }
`,
			expectedBasic: map[string]string{
				"capture_mode":              "continuous",
				"output_config.compression": "gzip",
				"performance.thread_count":  "4",
			},
			expectedUpdated: map[string]string{
				"capture_mode":              "triggered",
				"output_config.compression": "xz",
				"performance.thread_count":  "8",
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resourceName := fmt.Sprintf("%s.test", tc.resourceType)

			checkAttrs := func(attrs map[string]string) resource.TestCheckFunc {
				checks := []resource.TestCheckFunc{
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "monitoring_session_id", testAccMonitoringSessionID()),
				}
				for k, v := range attrs {
					checks = append(checks, resource.TestCheckResourceAttr(resourceName, k, v))
				}
				return resource.ComposeAggregateTestCheckFunc(checks...)
			}

			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: testAccPhase1AppConfig(tc.resourceType, tc.basicBody),
						Check:  checkAttrs(tc.expectedBasic),
					},
					{
						ResourceName:      resourceName,
						ImportState:       true,
						ImportStateIdFunc: testAccImportIDFromState(resourceName),
						ImportStateVerify: true,
					},
					{
						Config: testAccPhase1AppConfig(tc.resourceType, tc.updatedBody),
						Check:  checkAttrs(tc.expectedUpdated),
					},
				},
			})
		})
	}
}

func TestAccPhase1Apps_RequiredMonitoringSessionID(t *testing.T) {
	resources := []string{
		"gigamon_app_5gevp",
		"gigamon_app_5gsbi",
		"gigamon_app_sbipoe",
		"gigamon_app_ssl_decrypt",
		"gigamon_app_gvhttp2",
		"gigamon_app_pcapng",
	}

	for _, rt := range resources {
		rt := rt
		t.Run(rt, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      testAccPhase1MissingSessionConfig(rt),
						ExpectError: regexp.MustCompile("Missing required attribute"),
					},
				},
			})
		})
	}
}

func testAccPhase1AppConfig(resourceType string, resourceBody string) string {
	return fmt.Sprintf(`
%s

locals {
  ms_id = %q
}

resource %q "test" {
%s
}
`, testAccProviderConfig(), testAccMonitoringSessionID(), resourceType, resourceBody)
}

func testAccPhase1MissingSessionConfig(resourceType string) string {
	return fmt.Sprintf(`
%s

resource %q "test" {
}
`, testAccProviderConfig(), resourceType)
}
