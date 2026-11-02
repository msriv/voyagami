# Product analytics (VOY-223): the API publishes checked events to a topic, a
# BigQuery subscription writes them into analytics.events with no code, and
# nightly scheduled queries build small rollup tables for Looker Studio.

data "google_project" "this" {
  project_id = var.project_id
}

locals {
  dataset_id     = "${replace(var.prefix, "-", "_")}analytics"
  pubsub_agent   = "serviceAccount:service-${data.google_project.this.number}@gcp-sa-pubsub.iam.gserviceaccount.com"
  transfer_agent = "serviceAccount:service-${data.google_project.this.number}@gcp-sa-bigquerydatatransfer.iam.gserviceaccount.com"
  events         = "`${var.project_id}.${local.dataset_id}.events`"
}

resource "google_bigquery_dataset" "analytics" {
  project     = var.project_id
  dataset_id  = local.dataset_id
  location    = var.region
  description = "Product events and nightly rollups. No phone numbers, places or free text (checked by the API)."
}

# One row per event. Columns match what the API publishes (internal/analytics).
resource "google_bigquery_table" "events" {
  project    = var.project_id
  dataset_id = google_bigquery_dataset.analytics.dataset_id
  table_id   = "events"

  time_partitioning {
    type          = "DAY"
    field         = "eventTime"
    expiration_ms = var.raw_retention_days * 24 * 60 * 60 * 1000
  }
  clustering = ["name"]

  schema = jsonencode([
    { name = "eventId", type = "STRING", mode = "REQUIRED", description = "Random id per event, for de-duplication" },
    { name = "name", type = "STRING", mode = "REQUIRED" },
    { name = "eventTime", type = "TIMESTAMP", mode = "REQUIRED", description = "When it happened on the device" },
    { name = "receivedTime", type = "TIMESTAMP", mode = "NULLABLE" },
    { name = "uid", type = "STRING", mode = "NULLABLE", description = "Account id; null for events counted without consent" },
    { name = "sessionId", type = "STRING", mode = "NULLABLE" },
    { name = "platform", type = "STRING", mode = "NULLABLE" },
    { name = "osVersion", type = "STRING", mode = "NULLABLE" },
    { name = "appVersion", type = "STRING", mode = "NULLABLE" },
    { name = "appLanguage", type = "STRING", mode = "NULLABLE" },
    { name = "plan", type = "STRING", mode = "NULLABLE" },
    { name = "homeRegion", type = "STRING", mode = "NULLABLE", description = "ISO country code" },
    { name = "source", type = "STRING", mode = "REQUIRED", description = "app or server" },
    { name = "props", type = "STRING", mode = "NULLABLE", description = "Event properties as JSON text; read with JSON_VALUE" },
  ])
}

resource "google_pubsub_topic" "events" {
  project = var.project_id
  name    = "${var.prefix}analytics-events"
}

# Rows BigQuery refuses land here for a week, so a schema mistake loses nothing.
resource "google_pubsub_topic" "dead_letter" {
  project = var.project_id
  name    = "${var.prefix}analytics-events-dlq"
}

resource "google_pubsub_subscription" "dead_letter" {
  project                    = var.project_id
  name                       = "${var.prefix}analytics-events-dlq"
  topic                      = google_pubsub_topic.dead_letter.id
  message_retention_duration = "604800s"
}

resource "google_pubsub_topic_iam_member" "api_publish" {
  project = var.project_id
  topic   = google_pubsub_topic.events.name
  role    = "roles/pubsub.publisher"
  member  = "serviceAccount:${var.publisher_email}"
}

resource "google_bigquery_dataset_iam_member" "pubsub_writes" {
  project    = var.project_id
  dataset_id = google_bigquery_dataset.analytics.dataset_id
  role       = "roles/bigquery.dataEditor"
  member     = local.pubsub_agent
}

resource "google_pubsub_topic_iam_member" "pubsub_dead_letters" {
  project = var.project_id
  topic   = google_pubsub_topic.dead_letter.name
  role    = "roles/pubsub.publisher"
  member  = local.pubsub_agent
}

resource "google_pubsub_subscription" "to_bigquery" {
  project = var.project_id
  name    = "${var.prefix}analytics-events-to-bigquery"
  topic   = google_pubsub_topic.events.id

  bigquery_config {
    table               = "${var.project_id}.${local.dataset_id}.${google_bigquery_table.events.table_id}"
    use_table_schema    = true
    drop_unknown_fields = true
  }

  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.dead_letter.id
    max_delivery_attempts = 5
  }

  depends_on = [google_bigquery_dataset_iam_member.pubsub_writes]
}

resource "google_pubsub_subscription_iam_member" "pubsub_acks_dead_letters" {
  project      = var.project_id
  subscription = google_pubsub_subscription.to_bigquery.name
  role         = "roles/pubsub.subscriber"
  member       = local.pubsub_agent
}

# Nightly rollups run as their own account.
resource "google_service_account" "rollups" {
  project      = var.project_id
  account_id   = "${var.prefix}analytics-rollups"
  display_name = "Analytics nightly rollups"
}

resource "google_project_iam_member" "rollups_jobs" {
  project = var.project_id
  role    = "roles/bigquery.jobUser"
  member  = "serviceAccount:${google_service_account.rollups.email}"
}

resource "google_bigquery_dataset_iam_member" "rollups_write" {
  project    = var.project_id
  dataset_id = google_bigquery_dataset.analytics.dataset_id
  role       = "roles/bigquery.dataEditor"
  member     = "serviceAccount:${google_service_account.rollups.email}"
}

resource "google_service_account_iam_member" "transfer_agent_mints" {
  service_account_id = google_service_account.rollups.name
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = local.transfer_agent
}

locals {
  activation_steps = ["app_installed", "sign_in", "trip_created", "invite_sent", "crew_joined", "expense_added", "trial_started", "paid"]

  # shortcut: each rollup rebuilds from all raw events every night. Fine while
  # events are small; switch to incremental (one day at a time) past a few GB.
  rollups = {
    daily_counts = <<-SQL
      SELECT DATE(eventTime) AS day, name, platform, COUNT(*) AS events, COUNT(DISTINCT uid) AS people
      FROM ${local.events}
      GROUP BY day, name, platform
    SQL

    activation_funnel = <<-SQL
      WITH steps AS (
        SELECT step, i + 1 AS step_number FROM UNNEST(${jsonencode(local.activation_steps)}) AS step WITH OFFSET i
      ), firsts AS (
        SELECT uid, name, MIN(eventTime) AS at
        FROM ${local.events}
        WHERE uid IS NOT NULL AND name IN (SELECT step FROM steps)
        GROUP BY uid, name
      ), cohorts AS (
        SELECT uid, DATE_TRUNC(DATE(MIN(at)), WEEK(MONDAY)) AS cohort_week FROM firsts GROUP BY uid
      )
      SELECT cohort_week, step_number, step, COUNT(DISTINCT uid) AS people
      FROM firsts
      JOIN cohorts USING (uid)
      JOIN steps ON steps.step = firsts.name
      GROUP BY cohort_week, step_number, step
    SQL

    paywall_daily = <<-SQL
      SELECT DATE(eventTime) AS day, JSON_VALUE(props, '$.moment') AS moment,
        COUNTIF(name = 'paywall_shown') AS shown,
        COUNTIF(name = 'paywall_dismissed') AS dismissed,
        COUNTIF(name = 'paywall_converted') AS converted
      FROM ${local.events}
      WHERE name IN ('paywall_shown', 'paywall_dismissed', 'paywall_converted')
      GROUP BY day, moment
    SQL

    weekly_retention = <<-SQL
      WITH active AS (
        SELECT DISTINCT uid, DATE_TRUNC(DATE(eventTime), WEEK(MONDAY)) AS week
        FROM ${local.events}
        WHERE name = 'app_open' AND uid IS NOT NULL
      ), cohorts AS (
        SELECT uid, MIN(week) AS cohort_week FROM active GROUP BY uid
      )
      SELECT cohort_week, DATE_DIFF(week, cohort_week, WEEK) AS weeks_after, COUNT(*) AS people
      FROM active JOIN cohorts USING (uid)
      GROUP BY cohort_week, weeks_after
    SQL
  }
}

resource "google_bigquery_data_transfer_config" "rollups" {
  for_each = local.rollups

  project                = var.project_id
  location               = var.region
  display_name           = "${var.prefix}analytics ${each.key}"
  data_source_id         = "scheduled_query"
  schedule               = "every day 01:00"
  destination_dataset_id = google_bigquery_dataset.analytics.dataset_id
  service_account_name   = google_service_account.rollups.email

  params = {
    query                           = each.value
    destination_table_name_template = each.key
    write_disposition               = "WRITE_TRUNCATE"
  }

  depends_on = [google_project_iam_member.rollups_jobs, google_bigquery_dataset_iam_member.rollups_write, google_service_account_iam_member.transfer_agent_mints]
}

# Account deletion publishes an account_purged event for the uid. This removes
# every row of those uids (the marker too). Rows younger than 90 minutes wait a
# night, because BigQuery may still be committing them.
resource "google_bigquery_data_transfer_config" "purge" {
  project              = var.project_id
  location             = var.region
  display_name         = "${var.prefix}analytics purge deleted accounts"
  data_source_id       = "scheduled_query"
  schedule             = "every day 00:30"
  service_account_name = google_service_account.rollups.email

  params = {
    query = <<-SQL
      DELETE FROM ${local.events}
      WHERE uid IN (SELECT uid FROM ${local.events} WHERE name = 'account_purged' AND uid IS NOT NULL)
        AND receivedTime < TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 90 MINUTE)
    SQL
  }

  depends_on = [google_project_iam_member.rollups_jobs, google_bigquery_dataset_iam_member.rollups_write, google_service_account_iam_member.transfer_agent_mints]
}
