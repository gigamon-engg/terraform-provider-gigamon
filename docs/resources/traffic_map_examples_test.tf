
# This file contains all examples from the traffic_map.md documentation
# for validation purposes. It demonstrates all Phase 2 map conditions.

# Prerequisites: these would normally be created separately
# resource "gigamon_monitoring_session" "ms" { ... }
# resource "gigamon_application" "app_ats" { ... }
# resource "gigamon_application" "analysis_tool" { ... }

# ─── Example 1: GTP-U TEID Single Value ───────────────────────────────────────

resource "gigamon_traffic_map" "gtpu_teid_simple" {
  monitoring_session_id = "ms-123"
  name                  = "gtpu-teid-simple"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 10
      pass_rules = [
        {
          rule_id = 1
          gtp_teid = {
            teid_min           = "00000001"
            nested_level_count = 0
            subnet             = "none"
          }
        }
      ]
    }
  ]
}

# ─── Example 2: GTP-U TEID with Range ─────────────────────────────────────────

resource "gigamon_traffic_map" "gtpu_teid_range" {
  monitoring_session_id = "ms-123"
  name                  = "gtpu-teid-range"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 10
      pass_rules = [
        {
          rule_id = 1
          gtp_teid = {
            teid_min           = "00001000"
            teid_max           = "00001FFF"
            nested_level_count = 0
            subnet             = "all"
          }
        }
      ]
    }
  ]
}

# ─── Example 3: Host Name Prefix ──────────────────────────────────────────────

resource "gigamon_traffic_map" "host_name_simple" {
  monitoring_session_id = "ms-123"
  name                  = "host-name-simple"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 11
      pass_rules = [
        {
          rule_id = 1
          host_name = {
            src_host_prefix = "api.example.com"
          }
        }
      ]
    }
  ]
}

# ─── Example 4: IPv6 Flow Label Single Value ──────────────────────────────────

resource "gigamon_traffic_map" "ipv6_flow_label_simple" {
  monitoring_session_id = "ms-123"
  name                  = "ipv6-flow-label-simple"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 12
      pass_rules = [
        {
          rule_id = 1
          ipv6_flow_label = {
            label_min = 291
            pos       = 0
            subnet    = "none"
          }
        }
      ]
    }
  ]
}

# ─── Example 5: IPv6 Flow Label with Range ────────────────────────────────────

resource "gigamon_traffic_map" "ipv6_flow_label_range" {
  monitoring_session_id = "ms-123"
  name                  = "ipv6-flow-label-range"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 12
      pass_rules = [
        {
          rule_id = 1
          ipv6_flow_label = {
            label_min = 256
            label_max = 511
            pos       = 0
            subnet    = "all"
          }
        }
      ]
    }
  ]
}

# ─── Example 6: IPv6 Next Header Single Value ─────────────────────────────────

resource "gigamon_traffic_map" "ipv6_next_header_simple" {
  monitoring_session_id = "ms-123"
  name                  = "ipv6-tcp-simple"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 13
      pass_rules = [
        {
          rule_id = 1
          ipv6_next_header = {
            header_min = 6
            pos        = 0
            subnet     = "none"
          }
        }
      ]
    }
  ]
}

# ─── Example 7: IPv6 Next Header with Range ───────────────────────────────────

resource "gigamon_traffic_map" "ipv6_next_header_range" {
  monitoring_session_id = "ms-123"
  name                  = "ipv6-proto-range"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 13
      pass_rules = [
        {
          rule_id = 1
          ipv6_next_header = {
            header_min = 6
            header_max = 17
            pos        = 0
            subnet     = "all"
          }
        }
      ]
    }
  ]
}

# ─── Example 8: MPLS Label Single Value ───────────────────────────────────────

resource "gigamon_traffic_map" "mpls_label_simple" {
  monitoring_session_id = "ms-123"
  name                  = "mpls-label-simple"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 14
      pass_rules = [
        {
          rule_id = 1
          mpls_label = {
            value_min = 100
            pos       = 0
            subnet    = "none"
          }
        }
      ]
    }
  ]
}

# ─── Example 9: MPLS Label with Range ─────────────────────────────────────────

resource "gigamon_traffic_map" "mpls_label_range" {
  monitoring_session_id = "ms-123"
  name                  = "mpls-label-range"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 14
      pass_rules = [
        {
          rule_id = 1
          mpls_label = {
            value_min = 100
            value_max = 200
            pos       = 0
            subnet    = "all"
          }
        }
      ]
    }
  ]
}

# ─── Example 10: Port Destination Single Value ────────────────────────────────

resource "gigamon_traffic_map" "port_destination_simple" {
  monitoring_session_id = "ms-123"
  name                  = "https-destination-simple"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 15
      pass_rules = [
        {
          rule_id = 1
          port_destination = {
            port_min = 443
            pos      = 0
            subnet   = "none"
          }
        }
      ]
    }
  ]
}

# ─── Example 11: Port Destination with Range ──────────────────────────────────

resource "gigamon_traffic_map" "port_destination_range" {
  monitoring_session_id = "ms-123"
  name                  = "port-range"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 15
      pass_rules = [
        {
          rule_id = 1
          port_destination = {
            port_min = 8000
            port_max = 8100
            pos      = 0
            subnet   = "all"
          }
        }
      ]
    }
  ]
}

# ─── Example 12: IPv6 TCP Traffic with Multiple Conditions ────────────────────

resource "gigamon_traffic_map" "ipv6_tcp_traffic" {
  monitoring_session_id = "ms-123"
  name                  = "ipv6-tcp-traffic-map"
  description           = "Match IPv6 TCP traffic on port 443 (HTTPS) with specific flow label"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 10

      pass_rules = [
        {
          rule_id = 1

          ip_version = {
            ip_version = "v6"
          }

          ipv6_next_header = {
            header_min = 6
            pos        = 0
          }

          port_destination = {
            port_min = 443
            pos      = 0
          }

          ipv6_flow_label = {
            label_min = 256
            label_max = 512
            pos       = 0
            subnet    = "all"
          }
        }
      ]
    }
  ]
}

# ─── Example 13: GTP-U Traffic Filtering ──────────────────────────────────────

resource "gigamon_traffic_map" "gtp_u_filter" {
  monitoring_session_id = "ms-123"
  name                  = "gtp-u-filter-map"
  description           = "Filter GTP-U traffic by TEID and forward to tool"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 15

      pass_rules = [
        {
          rule_id = 1

          gtp_teid = {
            teid_min           = "00000100"
            teid_max           = "000001FF"
            nested_level_count = 0
            subnet             = "all"
          }
        }
      ]
    }
  ]
}

# ─── Example 14: MPLS Traffic with Drop Rules ─────────────────────────────────

resource "gigamon_traffic_map" "mpls_drop_rules" {
  monitoring_session_id = "ms-123"
  name                  = "mpls-traffic-map"
  description           = "Route MPLS traffic; drop reserved labels"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 20

      pass_rules = [
        {
          rule_id = 1
          mpls_label = {
            value_min = 16
            value_max = 1048575
            pos       = 0
            subnet    = "none"
          }
        }
      ]

      drop_rules = [
        {
          rule_id = 1
          mpls_label = {
            value_min = 0
            value_max = 15
            pos       = 0
            subnet    = "none"
          }
        }
      ]
    }
  ]
}

# ─── Example 15: Host Name Routing with Multiple Prefix Matches ────────────────

resource "gigamon_traffic_map" "host_name_routing" {
  monitoring_session_id = "ms-123"
  name                  = "host-name-routing-map"
  description           = "Route traffic based on source host name"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 25

      pass_rules = [
        {
          rule_id = 1
          host_name = {
            src_host_prefix = "api-server"
          }
        },
        {
          rule_id = 2
          host_name = {
            src_host_prefix = "web-srv"
          }
        }
      ]
    }
  ]
}

# ─── Example 16: Multi-Condition Rule with IPv6, MPLS, and Port Matching ──────

resource "gigamon_traffic_map" "multi_condition_map" {
  monitoring_session_id = "ms-123"
  name                  = "multi-condition-map"
  description           = "Advanced traffic map with multiple Phase 2 conditions"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 30

      pass_rules = [
        {
          rule_id = 1

          ip_version = {
            ip_version = "v6"
          }

          mpls_label = {
            value_min = 100
            value_max = 200
            pos       = 0
            subnet    = "none"
          }

          port_destination = {
            port_min = 5000
            port_max = 6000
            pos      = 0
            subnet   = "all"
          }
        },

        {
          rule_id = 2
          gtp_teid = {
            teid_min = "00000001"
            subnet   = "none"
          }
        }
      ]

      drop_rules = [
        {
          rule_id = 1
          mpls_label = {
            value_min = 0
            value_max = 15
            pos       = 0
          }
        }
      ]
    }
  ]
}
