data "google_dns_managed_zone" "env_dns_zone" {
  provider = google-beta
  name = "gcp-jkwong-info"
  project   = data.google_project.host_project.project_id
}

resource "google_dns_record_set" "hellogrpc-dev" {
  provider      = google-beta
  managed_zone  = data.google_dns_managed_zone.env_dns_zone.name
  project       = data.google_project.host_project.project_id
  name          = "hellogrpc-dev.gcp.jkwong.info."
  type          = "A"
  rrdatas       = [
    google_compute_global_address.hellogrpc-dev.address
  ]
  ttl          = 300
}

resource "google_compute_global_address" "hellogrpc-dev" {
  name      = "hellogrpc-dev"
  project   = module.service_project.project_id
}

resource "google_compute_global_forwarding_rule" "hellogrpc-dev-https" {
  name        = "hellogrpc-dev-https"
  target      = google_compute_target_https_proxy.hellogrpc-dev.id
  port_range  = "443"
  ip_address  = google_compute_global_address.hellogrpc-dev.id
  load_balancing_scheme = "EXTERNAL_MANAGED"
  project     = module.service_project.project_id
}

resource "google_compute_managed_ssl_certificate" "hellogrpc-dev" {
  name      = "hellogrpc-dev"
  project   = module.service_project.project_id

  managed {
    domains = ["hellogrpc-dev.gcp.jkwong.info."]
  }
}

resource "google_compute_target_https_proxy" "hellogrpc-dev" {
  name              = "hellogrpc-dev"
  url_map           = google_compute_url_map.hellogrpc-dev.id
  ssl_certificates  = [google_compute_managed_ssl_certificate.hellogrpc-dev.id]
  project           = module.service_project.project_id
}

resource "google_compute_url_map" "hellogrpc-dev" {
  name            = "hellogrpc-dev"
  description     = "hellogrpc-dev"
  default_service = google_compute_backend_service.hellogrpc-dev-a.id
  project         = module.service_project.project_id

  host_rule {
    hosts        = ["hellogrpc-dev.gcp.jkwong.info"]
    path_matcher = "allpaths"
  }

  path_matcher {
    name            = "allpaths"
    default_service = google_compute_backend_service.default.id

    route_rules {
      priority = 1000
      service = google_compute_backend_service.hellogrpc-dev-a.id
      match_rules {
        prefix_match = "/"
        ignore_case = true
        header_matches {
          header_name = "X-Tenant-Id"
          prefix_match = "0"
        }
      }
    }

    route_rules {
      priority = 1001
      service = google_compute_backend_service.hellogrpc-dev-a.id
      match_rules {
        prefix_match = "/"
        ignore_case = true
        header_matches {
          header_name = "X-Tenant-Id"
          prefix_match = "1"
        }
      }
    }

    route_rules {
      priority = 1002
      service = google_compute_backend_service.hellogrpc-dev-a.id
      match_rules {
        prefix_match = "/"
        ignore_case = true
        header_matches {
          header_name = "X-Tenant-Id"
          prefix_match = "2"
        }
      }
    }

    route_rules {
      priority = 1003
      service = google_compute_backend_service.hellogrpc-dev-a.id
      match_rules {
        prefix_match = "/"
        ignore_case = true
        header_matches {
          header_name = "X-Tenant-Id"
          prefix_match = "3"
        }
      }
    }

    route_rules {
      priority = 1004
      service = google_compute_backend_service.hellogrpc-dev-b.id
      match_rules {
        prefix_match = "/"
        ignore_case = true
        header_matches {
          header_name = "X-Tenant-Id"
          prefix_match = "4"
        }
      }
    }

    route_rules {
      priority = 1005
      service = google_compute_backend_service.hellogrpc-dev-b.id
      match_rules {
        prefix_match = "/"
        ignore_case = true
        header_matches {
          header_name = "X-Tenant-Id"
          prefix_match = "5"
        }
      }
    }

    route_rules {
      priority = 1006
      service = google_compute_backend_service.hellogrpc-dev-b.id
      match_rules {
        prefix_match = "/"
        ignore_case = true
        header_matches {
          header_name = "X-Tenant-Id"
          prefix_match = "6"
        }
      }
    }

    route_rules {
      priority = 1007
      service = google_compute_backend_service.hellogrpc-dev-b.id
      match_rules {
        prefix_match = "/"
        ignore_case = true
        header_matches {
          header_name = "X-Tenant-Id"
          prefix_match = "7"
        }
      }
    }

    route_rules {
      priority = 1008
      service = google_compute_backend_service.hellogrpc-dev-c.id
      match_rules {
        prefix_match = "/"
        ignore_case = true
        header_matches {
          header_name = "X-Tenant-Id"
          prefix_match = "8"
        }
      }
    }

    route_rules {
      priority = 1009
      service = google_compute_backend_service.hellogrpc-dev-c.id
      match_rules {
        prefix_match = "/"
        ignore_case = true
        header_matches {
          header_name = "X-Tenant-Id"
          prefix_match = "9"
        }
      }
    }

    route_rules {
      priority = 1010
      service = google_compute_backend_service.hellogrpc-dev-c.id
      match_rules {
        prefix_match = "/"
        ignore_case = true
        header_matches {
          header_name = "X-Tenant-Id"
          prefix_match = "a"
        }
      }
    }

    route_rules {
      priority = 1011
      service = google_compute_backend_service.hellogrpc-dev-c.id
      match_rules {
        prefix_match = "/"
        ignore_case = true
        header_matches {
          header_name = "X-Tenant-Id"
          prefix_match = "b"
        }
      }
    }

    route_rules {
      priority = 1012
      service = google_compute_backend_service.hellogrpc-dev-d.id
      match_rules {
        prefix_match = "/"
        ignore_case = true
        header_matches {
          header_name = "X-Tenant-Id"
          prefix_match = "c"
        }
      }
    }

    route_rules {
      priority = 1013
      service = google_compute_backend_service.hellogrpc-dev-d.id
      match_rules {
        prefix_match = "/"
        ignore_case = true
        header_matches {
          header_name = "X-Tenant-Id"
          prefix_match = "d"
        }
      }
    }

    route_rules {
      priority = 1014
      service = google_compute_backend_service.hellogrpc-dev-d.id
      match_rules {
        prefix_match = "/"
        ignore_case = true
        header_matches {
          header_name = "X-Tenant-Id"
          prefix_match = "e"
        }
      }
    }

    route_rules {
      priority = 1015
      service = google_compute_backend_service.hellogrpc-dev-d.id
      match_rules {
        prefix_match = "/"
        ignore_case = true
        header_matches {
          header_name = "X-Tenant-Id"
          prefix_match = "f"
        }
      }
    }

  }
}

resource "google_compute_backend_service" "hellogrpc-dev-d" {
  name        = "hellogrpc-dev-d"
  port_name   = "http2"
  protocol    = "HTTP2"
  timeout_sec = 300

  load_balancing_scheme = "EXTERNAL_MANAGED"
  locality_lb_policy    = "LEAST_REQUEST"

  lifecycle {
    ignore_changes = [
      backend,
    ]
  }

  health_checks = [google_compute_health_check.http-health-check.id]
  project       = module.service_project.project_id

  log_config {
    enable = true
    sample_rate = 1
  }

}

resource "google_compute_backend_service" "hellogrpc-dev-c" {
  name        = "hellogrpc-dev-c"
  port_name   = "http2"
  protocol    = "HTTP2"
  timeout_sec = 300

  load_balancing_scheme = "EXTERNAL_MANAGED"
  locality_lb_policy    = "LEAST_REQUEST"

  lifecycle {
    ignore_changes = [
      backend,
    ]
  }

  health_checks = [google_compute_health_check.http-health-check.id]
  project       = module.service_project.project_id

  log_config {
    enable = true
    sample_rate = 1
  }

}

resource "google_compute_backend_service" "hellogrpc-dev-b" {
  name        = "hellogrpc-dev-b"
  port_name   = "http2"
  protocol    = "HTTP2"
  timeout_sec = 300

  load_balancing_scheme = "EXTERNAL_MANAGED"
  locality_lb_policy    = "LEAST_REQUEST"

  lifecycle {
    ignore_changes = [
      backend,
    ]
  }

  health_checks = [google_compute_health_check.http-health-check.id]
  project       = module.service_project.project_id

  log_config {
    enable = true
    sample_rate = 1
  }

}

resource "google_compute_backend_service" "hellogrpc-dev-a" {
  name        = "hellogrpc-dev-a"
  port_name   = "http2"
  protocol    = "HTTP2"
  timeout_sec = 300

  load_balancing_scheme = "EXTERNAL_MANAGED"
  locality_lb_policy    = "LEAST_REQUEST"

  lifecycle {
    ignore_changes = [
      backend,
    ]
  }

  health_checks = [google_compute_health_check.http-health-check.id]
  project       = module.service_project.project_id

  log_config {
    enable = true
    sample_rate = 1
  }

}

resource "google_compute_backend_service" "default" {
  name        = "default"
  port_name   = "http2"
  protocol    = "HTTP2"
  timeout_sec = 300

  load_balancing_scheme = "EXTERNAL_MANAGED"
  locality_lb_policy    = "LEAST_REQUEST"

  lifecycle {
    ignore_changes = [
      backend,
    ]
  }

  health_checks = [google_compute_health_check.http-health-check.id]
  project       = module.service_project.project_id

  log_config {
    enable = true
    sample_rate = 1
  }

}

resource "google_compute_health_check" "tcp-health-check" {
  name               = "tcp-health-check"
  timeout_sec        = 1
  check_interval_sec = 3
  project            = module.service_project.project_id

  tcp_health_check {
    port_specification = "USE_SERVING_PORT"
  }

  log_config {
    enable = true
  }

}

resource "google_compute_health_check" "http-health-check" {
  name                = "http-health-check"
  check_interval_sec  = 3
  timeout_sec         = 1
  project             = module.service_project.project_id

  http_health_check {
    port_specification  = "USE_SERVING_PORT"
    request_path        = "/healthz"
  }

  log_config {
    enable = true
  }
}

resource "google_compute_health_check" "grpc-health-check" {
  name               = "grpc-health-check"
  timeout_sec        = 1
  check_interval_sec = 3
  project            = module.service_project.project_id

  grpc_health_check {
    port_specification = "USE_SERVING_PORT"
  }

  log_config {
    enable = true
  }

}

/*

resource "google_dns_record_set" "istio-ingressgateway" {
  provider      = google-beta
  managed_zone  = data.google_dns_managed_zone.env_dns_zone.name
  project       = data.google_project.host_project.project_id
  name          = "istio-ingressgateway.gcp.jkwong.info."
  type          = "A"
  rrdatas       = [
    google_compute_global_address.istio-ingressgateway.address
  ]
  ttl          = 300
}

resource "google_compute_global_address" "istio-ingressgateway" {
  name      = "istio-ingressgateway"
  project   = module.service_project.project_id
}

resource "google_compute_global_forwarding_rule" "istio-ingressgateway-https" {
  name        = "istio-ingressgateway-https"
  target      = google_compute_target_https_proxy.istio-ingressgateway.id
  port_range  = "443"
  ip_address  = google_compute_global_address.istio-ingressgateway.id
  load_balancing_scheme = "EXTERNAL_MANAGED"
  project     = module.service_project.project_id
}

resource "google_compute_managed_ssl_certificate" "istio-ingressgateway" {
  name      = "istio-ingressgateway"
  project   = module.service_project.project_id

  managed {
    domains = ["istio-ingressgateway.gcp.jkwong.info."]
  }
}

resource "google_compute_target_https_proxy" "istio-ingressgateway" {
  name              = "istio-ingressgateway"
  url_map           = google_compute_url_map.istio-ingressgateway.id
  ssl_certificates  = [google_compute_managed_ssl_certificate.istio-ingressgateway.id]
  project           = module.service_project.project_id
}

resource "google_compute_url_map" "istio-ingressgateway" {
  name            = "istio-ingressgateway"
  description     = "istio-ingressgateway"
  default_service = google_compute_backend_service.istio-ingressgateway.self_link
  project         = module.service_project.project_id

  host_rule {
    hosts        = ["istio-ingressgateway.gcp.jkwong.info"]
    path_matcher = "allpaths"
  }

  path_matcher {
    name            = "allpaths"
    default_service = google_compute_backend_service.istio-ingressgateway.self_link

    path_rule {
      paths   = ["/*"]
      service = google_compute_backend_service.istio-ingressgateway.self_link
    }
  }

}

resource "google_compute_backend_service" "istio-ingressgateway" {
  name        = "istio-ingressgateway-dev"
  port_name   = "http2"
  protocol    = "HTTP2"
  timeout_sec = 300

  load_balancing_scheme = "EXTERNAL_MANAGED"
  locality_lb_policy    = "LEAST_REQUEST"

  lifecycle {
    ignore_changes = [
      backend,
    ]
  }

  log_config {
    enable = true
    sample_rate = 1
  }


  health_checks = [google_compute_health_check.istio-ingressgateway-check.id]
  project       = module.service_project.project_id
}

resource "google_compute_health_check" "istio-ingressgateway-check" {
  name                = "check-istio-ingressgateway"
  check_interval_sec  = 3
  timeout_sec         = 1
  project             = module.service_project.project_id

  http_health_check {
    port              = 15021
    request_path      = "/healthz/ready"
  }

  log_config {
    enable = true
  }
}


*/