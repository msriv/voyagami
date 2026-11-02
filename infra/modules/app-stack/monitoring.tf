# Uptime check, alerts and a dashboard for this environment's API (VOY-227).
# Alerts go to var.alert_email; apply.sh passes the signed-in gcloud account, so
# no address is committed. Empty means checks and dashboard without emails.

locals {
  api_host     = trimprefix(module.api.url, "https://")
  service_name = "${var.prefix}voyagami-api"
  run_filter   = "resource.type=\"cloud_run_revision\" AND resource.label.service_name=\"${local.service_name}\""
  channels     = [for c in google_monitoring_notification_channel.email : c.id]
}

resource "google_monitoring_notification_channel" "email" {
  count = var.alert_email == "" ? 0 : 1

  project      = var.project_id
  display_name = "${var.prefix}alerts email"
  type         = "email"
  labels = {
    email_address = var.alert_email
  }
}

resource "google_monitoring_uptime_check_config" "api" {
  project      = var.project_id
  display_name = "${local.service_name} /health"
  timeout      = "10s"
  period       = "300s"

  http_check {
    path         = "/health" # not /healthz: Cloud Run answers that itself (VOY-210)
    port         = 443
    use_ssl      = true
    validate_ssl = true
  }

  monitored_resource {
    type = "uptime_url"
    labels = {
      project_id = var.project_id
      host       = local.api_host
    }
  }
}

resource "google_monitoring_alert_policy" "uptime" {
  project               = var.project_id
  display_name          = "${local.service_name} is down"
  combiner              = "OR"
  notification_channels = local.channels

  conditions {
    display_name = "/health failing"
    condition_threshold {
      filter          = "metric.type=\"monitoring.googleapis.com/uptime_check/check_passed\" AND metric.label.check_id=\"${google_monitoring_uptime_check_config.api.uptime_check_id}\" AND resource.type=\"uptime_url\""
      comparison      = "COMPARISON_GT"
      threshold_value = 1
      duration        = "600s"
      aggregations {
        alignment_period     = "1200s"
        per_series_aligner   = "ALIGN_NEXT_OLDER"
        cross_series_reducer = "REDUCE_COUNT_FALSE"
        group_by_fields      = ["resource.label.*"]
      }
    }
  }
}

resource "google_monitoring_alert_policy" "errors" {
  project               = var.project_id
  display_name          = "${local.service_name} 5xx errors"
  combiner              = "OR"
  notification_channels = local.channels

  conditions {
    display_name = "More than 5 server errors in 5 minutes"
    condition_threshold {
      filter          = "metric.type=\"run.googleapis.com/request_count\" AND metric.label.response_code_class=\"5xx\" AND ${local.run_filter}"
      comparison      = "COMPARISON_GT"
      threshold_value = 5
      duration        = "0s"
      aggregations {
        alignment_period     = "300s"
        per_series_aligner   = "ALIGN_DELTA"
        cross_series_reducer = "REDUCE_SUM"
      }
    }
  }
}

resource "google_monitoring_alert_policy" "latency" {
  project               = var.project_id
  display_name          = "${local.service_name} slow"
  combiner              = "OR"
  notification_channels = local.channels

  conditions {
    display_name = "p95 latency over 2 s for 10 minutes"
    condition_threshold {
      filter          = "metric.type=\"run.googleapis.com/request_latencies\" AND ${local.run_filter}"
      comparison      = "COMPARISON_GT"
      threshold_value = 2000
      duration        = "600s"
      aggregations {
        alignment_period     = "300s"
        per_series_aligner   = "ALIGN_PERCENTILE_95"
        cross_series_reducer = "REDUCE_MAX"
      }
    }
  }
}

resource "google_monitoring_dashboard" "api" {
  project = var.project_id
  dashboard_json = jsonencode({
    displayName = "${local.service_name}"
    mosaicLayout = {
      columns = 12
      tiles = [for i, w in [
        { title = "Requests by status class", metric = "run.googleapis.com/request_count", aligner = "ALIGN_RATE", reducer = "REDUCE_SUM", group = ["metric.label.response_code_class"] },
        { title = "Latency p95 (ms)", metric = "run.googleapis.com/request_latencies", aligner = "ALIGN_PERCENTILE_95", reducer = "REDUCE_MAX", group = [] },
        { title = "Instances", metric = "run.googleapis.com/container/instance_count", aligner = "ALIGN_MAX", reducer = "REDUCE_SUM", group = ["metric.label.state"] },
        { title = "CPU utilisation p95", metric = "run.googleapis.com/container/cpu/utilizations", aligner = "ALIGN_PERCENTILE_95", reducer = "REDUCE_MAX", group = [] },
        ] : {
        xPos   = (i % 2) * 6
        yPos   = floor(i / 2) * 4
        width  = 6
        height = 4
        widget = {
          title = w.title
          xyChart = {
            dataSets = [{
              plotType = "LINE"
              timeSeriesQuery = {
                timeSeriesFilter = {
                  filter = "metric.type=\"${w.metric}\" AND ${local.run_filter}"
                  aggregation = {
                    alignmentPeriod    = "60s"
                    perSeriesAligner   = w.aligner
                    crossSeriesReducer = w.reducer
                    groupByFields      = w.group
                  }
                }
              }
            }]
          }
        }
      }]
    }
  })
}
