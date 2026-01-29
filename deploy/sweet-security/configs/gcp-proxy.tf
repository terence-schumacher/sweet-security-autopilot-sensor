terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# Define variables for GCP resources
variable "project_id" {
  description = "GCP Project ID"
  type        = string
}

variable "region" {
  description = "GCP Region"
  type        = string
  default     = "us-central1"
}

variable "zone" {
  description = "GCP Zone"
  type        = string
  default     = "us-central1-a"
}

variable "network_name" {
  description = "VPC Network Name"
  type        = string
}

variable "subnet_name" {
  description = "Subnet Name"
  type        = string
}

variable "machine_image" {
  description = "GCE Machine Image (e.g., ubuntu-os-cloud/ubuntu-2204-lts-arm64)"
  type        = string
  default     = "ubuntu-os-cloud/ubuntu-2204-lts-arm64"
}

# Create a Service Account for the proxy instance
resource "google_service_account" "sweet_proxy" {
  account_id   = "sweet-proxy"
  display_name = "Sweet Security Proxy Service Account"
  project      = var.project_id
}

# Grant necessary permissions to the service account (minimal permissions for proxy)
resource "google_project_iam_member" "sweet_proxy_logging" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.sweet_proxy.email}"
}

# Create Firewall Rules for the proxy instance
resource "google_compute_firewall" "sweet_proxy_ingress_tcp" {
  name    = "sweet-proxy-ingress-tcp"
  network = var.network_name
  project = var.project_id

  allow {
    protocol = "tcp"
    ports    = ["443"]
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["sweet-proxy"]
}

resource "google_compute_firewall" "sweet_proxy_ingress_udp" {
  name    = "sweet-proxy-ingress-udp"
  network = var.network_name
  project = var.project_id

  allow {
    protocol = "udp"
    ports    = ["443"]
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["sweet-proxy"]
}

resource "google_compute_firewall" "sweet_proxy_egress" {
  name      = "sweet-proxy-egress"
  network   = var.network_name
  project   = var.project_id
  direction = "EGRESS"

  allow {
    protocol = "all"
  }

  destination_ranges = ["18.220.208.31/32"]
  target_tags        = ["sweet-proxy"]
}

# Create the GCE instance for the proxy
resource "google_compute_instance" "sweet_proxy" {
  name         = "sweet-proxy"
  machine_type = "e2-standard-2" # x86 instance (ARM not available in us-west1)
  zone         = var.zone
  project      = var.project_id

  boot_disk {
    initialize_params {
      image = var.machine_image
      size  = 20
      type  = "pd-balanced" # pd-standard not compatible with all machine types
    }
  }

  network_interface {
    network    = var.network_name
    subnetwork = var.subnet_name

    # Optional: Reserve a static internal IP
    # access_config {
    #   // Ephemeral public IP (if needed)
    # }
  }

  service_account {
    email  = google_service_account.sweet_proxy.email
    scopes = ["cloud-platform"]
  }

  metadata_startup_script = <<-EOF
#!/usr/bin/env bash
set -e

# Sweet IP
TARGET_IP=18.220.208.31

# Get the local IP address
LOCAL_IP=$(hostname -I | awk '{print $1}')
TARGET_PORT=443
LOCAL_PORT=443

# Enable IPv4 forwarding
sysctl -w net.ipv4.ip_forward=1 > /dev/null
echo "net.ipv4.ip_forward=1" >> /etc/sysctl.conf

# Clear existing NAT rules
iptables -t nat -F
iptables -t nat -X

# Set up DNAT: Traffic arriving at LOCAL_IP:LOCAL_PORT will be forwarded to TARGET_IP:TARGET_PORT
iptables -t nat -A PREROUTING -p tcp -d $LOCAL_IP --dport $LOCAL_PORT -j DNAT --to-destination $TARGET_IP:$TARGET_PORT
iptables -t nat -A PREROUTING -p udp -d $LOCAL_IP --dport $LOCAL_PORT -j DNAT --to-destination $TARGET_IP:$TARGET_PORT

# Set up MASQUERADE so that return traffic passes back through this host
iptables -t nat -A POSTROUTING -p tcp -d $TARGET_IP --dport $TARGET_PORT -j MASQUERADE
iptables -t nat -A POSTROUTING -p udp -d $TARGET_IP --dport $TARGET_PORT -j MASQUERADE

# Ensure FORWARD chain rules allow the forwarded traffic
iptables -A FORWARD -p tcp -d $TARGET_IP --dport $TARGET_PORT -j ACCEPT
iptables -A FORWARD -p tcp -s $TARGET_IP --sport $TARGET_PORT -j ACCEPT
iptables -A FORWARD -p udp -d $TARGET_IP --dport $TARGET_PORT -j ACCEPT
iptables -A FORWARD -p udp -s $TARGET_IP --sport $TARGET_PORT -j ACCEPT

# Save rules (Ubuntu/Debian)
# Wait a moment for network to be fully up
sleep 5
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install -y iptables-persistent
iptables-save > /etc/iptables/rules.v4
  EOF

  tags = ["sweet-proxy"]

  labels = {
    name = "sweet-proxy"
  }
}

# Create a Cloud DNS Private Zone
resource "google_dns_managed_zone" "sweet_security" {
  name        = "sweet-security-zone"
  dns_name    = "sweet.security."
  description = "Private DNS zone for Sweet Security"
  project     = var.project_id

  visibility = "private"

  private_visibility_config {
    networks {
      network_url = "projects/${var.project_id}/global/networks/${var.network_name}"
    }
  }
}

# Add a wildcard DNS A record for the proxy
resource "google_dns_record_set" "sweet_security_wildcard" {
  name         = "*.sweet.security."
  managed_zone = google_dns_managed_zone.sweet_security.name
  type         = "A"
  ttl          = 300
  project      = var.project_id

  rrdatas = [google_compute_instance.sweet_proxy.network_interface[0].network_ip]
}

# Outputs
output "proxy_instance_name" {
  description = "Name of the proxy instance"
  value       = google_compute_instance.sweet_proxy.name
}

output "proxy_internal_ip" {
  description = "Internal IP address of the proxy instance"
  value       = google_compute_instance.sweet_proxy.network_interface[0].network_ip
}

output "dns_zone_name" {
  description = "Name of the DNS zone"
  value       = google_dns_managed_zone.sweet_security.name
}

output "wildcard_dns_record" {
  description = "Wildcard DNS record"
  value       = "*.sweet.security -> ${google_compute_instance.sweet_proxy.network_interface[0].network_ip}"
}
