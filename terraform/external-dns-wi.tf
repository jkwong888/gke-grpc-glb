# GKE node service account
resource "google_service_account" "external_dns_sa" {
  project       = module.service_project.project_id
  account_id    = format("%s-external-dns", var.gke_cluster_name)
  display_name  = format("%s external dns service account", var.gke_cluster_name)
}

resource "google_service_account_iam_member" "external_dns_wi" {
  service_account_id = google_service_account.external_dns_sa.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${module.service_project.project_id}.svc.id.goog[external-dns/external-dns]"
}

resource "google_project_iam_member" "external_dns_admin" {
    project = data.google_project.host_project.project_id
    role = "roles/dns.admin"
    member = "serviceAccount:${google_service_account.external_dns_sa.email}"
}