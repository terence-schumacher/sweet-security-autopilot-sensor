# GCP/GKE Autopilot Deployment Guide

This directory contains Terraform configurations and Kubernetes manifests for deploying the Sweet Security proxy on Google Cloud Platform (GCP) and GKE Autopilot.

## Files Overview

1. **`gcp-proxy.tf`** - Terraform configuration for deploying a GCE VM instance as a proxy
2. **`gke-autopilot.tf`** - Terraform configuration for creating a GKE Autopilot cluster with Cloud DNS
3. **`gke-autopilot-proxy.yaml`** - Kubernetes manifest for deploying the proxy as a pod in GKE Autopilot

## Option 1: GCE VM Instance (Direct AWS Equivalent)

This option creates a standalone GCE VM instance that acts as a proxy, similar to the original AWS EC2 instance.

### Prerequisites

- GCP Project with billing enabled
- Terraform >= 1.0
- `gcloud` CLI configured with appropriate permissions
- VPC network and subnet already created

### Variables

Create a `terraform.tfvars` file:

```hcl
project_id     = "your-gcp-project-id"
region         = "us-central1"
zone           = "us-central1-a"
network_name   = "your-vpc-network"
subnet_name    = "your-subnet-name"
machine_image  = "ubuntu-os-cloud/ubuntu-2204-lts-arm64"
```

### Deployment Steps

1. Initialize Terraform:
   ```bash
   terraform init
   ```

2. Review the plan:
   ```bash
   terraform plan
   ```

3. Apply the configuration:
   ```bash
   terraform apply
   ```

4. After deployment, update the Cloud DNS record with the instance's internal IP:
   ```bash
   # Get the internal IP
   gcloud compute instances describe sweet-proxy --zone=us-central1-a --format="get(networkInterfaces[0].networkIP)"
   
   # Update the DNS record in Terraform or manually via gcloud
   ```

### Outputs

- `proxy_instance_name` - Name of the proxy instance
- `proxy_internal_ip` - Internal IP address of the proxy
- `dns_zone_name` - Name of the DNS zone
- `wildcard_dns_record` - DNS record details

## Option 2: GKE Autopilot Deployment

This option deploys the proxy as a Kubernetes workload in a GKE Autopilot cluster, which is fully managed by Google.

### Prerequisites

- GCP Project with billing enabled
- Terraform >= 1.0
- `gcloud` CLI configured
- `kubectl` installed
- VPC network and subnet already created (or create them separately)

### Step 1: Create GKE Autopilot Cluster

1. Create a `terraform.tfvars` file:
   ```hcl
   project_id   = "your-gcp-project-id"
   region       = "us-central1"
   cluster_name = "sweet-security-cluster"
   network_name = "your-vpc-network"
   subnet_name  = "your-subnet-name"
   ```

2. Initialize and apply:
   ```bash
   terraform init
   terraform plan
   terraform apply
   ```

3. Configure kubectl:
   ```bash
   gcloud container clusters get-credentials sweet-security-cluster \
     --region us-central1 \
     --project your-gcp-project-id
   ```

### Step 2: Deploy the Proxy Pod

1. Deploy the Kubernetes manifest:
   ```bash
   kubectl apply -f gke-autopilot-proxy.yaml
   ```

2. Wait for the pod to be ready:
   ```bash
   kubectl wait --for=condition=ready pod -l app=sweet-proxy --timeout=300s
   ```

3. Get the service ClusterIP:
   ```bash
   kubectl get svc sweet-proxy -o jsonpath='{.spec.clusterIP}'
   ```

4. Update the Cloud DNS record with the ClusterIP:
   ```bash
   # Update the rrdatas in gke-autopilot.tf or use gcloud:
   gcloud dns record-sets update *.sweet.security. \
     --zone=sweet-security-zone \
     --rrdatas=10.x.x.x \
     --type=A \
     --project=your-gcp-project-id
   ```

### Important Notes for GKE Autopilot

- **Privileged Containers**: The proxy requires privileged access to modify iptables. GKE Autopilot allows privileged containers but with certain restrictions.
- **Resource Limits**: GKE Autopilot automatically manages resources, but you can specify requests/limits in the deployment.
- **Network Policies**: Consider adding NetworkPolicies if you need to restrict traffic to/from the proxy.

## Differences from AWS Version

### Infrastructure Changes

| AWS Resource | GCP Equivalent |
|-------------|----------------|
| IAM Role | Service Account |
| Security Group | Firewall Rules |
| EC2 Instance | GCE VM Instance |
| Route 53 Private Zone | Cloud DNS Private Zone |
| t4g.medium | t2a-standard-2 (ARM-based) |

### Key Differences

1. **Instance Type**: Uses `t2a-standard-2` (ARM-based) instead of `t4g.medium`
2. **Firewall Rules**: Separate ingress/egress rules instead of combined security group
3. **DNS**: Cloud DNS private zones work similarly to Route 53
4. **IAM**: Service accounts with IAM bindings instead of IAM roles
5. **User Data**: Uses `metadata_startup_script` instead of `user_data`

## Troubleshooting

### GCE VM Instance

1. **Check instance status**:
   ```bash
   gcloud compute instances describe sweet-proxy --zone=us-central1-a
   ```

2. **View startup script logs**:
   ```bash
   gcloud compute instances get-serial-port-output sweet-proxy --zone=us-central1-a
   ```

3. **SSH into instance**:
   ```bash
   gcloud compute ssh sweet-proxy --zone=us-central1-a
   ```

4. **Check iptables rules**:
   ```bash
   sudo iptables -t nat -L -n -v
   ```

### GKE Autopilot

1. **Check pod status**:
   ```bash
   kubectl get pods -l app=sweet-proxy
   kubectl describe pod -l app=sweet-proxy
   ```

2. **View pod logs**:
   ```bash
   kubectl logs -l app=sweet-proxy
   ```

3. **Check service**:
   ```bash
   kubectl get svc sweet-proxy
   kubectl describe svc sweet-proxy
   ```

4. **Test DNS resolution**:
   ```bash
   kubectl run -it --rm debug --image=busybox --restart=Never -- nslookup test.sweet.security
   ```

## Security Considerations

1. **Firewall Rules**: The current configuration allows ingress from `0.0.0.0/0`. Consider restricting this to your VPC CIDR ranges.
2. **Service Account Permissions**: The service account has minimal permissions. Adjust as needed.
3. **Private Cluster**: For GKE, consider enabling private endpoint for additional security.
4. **Network Policies**: Implement Kubernetes NetworkPolicies to restrict pod-to-pod communication.

## Cost Optimization

- **GCE VM**: Consider using preemptible instances for cost savings (not suitable for production)
- **GKE Autopilot**: Automatically scales and optimizes resources, but ensure you understand the pricing model
- **Instance Size**: Adjust machine types based on actual traffic requirements

## Next Steps

After deploying the proxy:

1. Verify DNS resolution works from your pods/services
2. Test connectivity to the target IP (18.220.208.31:443)
3. Monitor proxy instance/pod metrics
4. Set up alerts for proxy failures
5. Consider adding a load balancer if high availability is required
