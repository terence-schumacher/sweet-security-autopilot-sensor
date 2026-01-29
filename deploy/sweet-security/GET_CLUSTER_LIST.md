# How to Retrieve the List of 40 K8s Clusters

The Sweet Security dashboard shows "6/40" for GCP K8s clusters, meaning 6 clusters have sensors installed out of 40 total clusters. Here are several ways to get the complete list:

## Method 1: List from GCP (Recommended)

Use the provided script to list all GKE clusters directly from GCP:

```bash
cd deploy/sweet-security/scripts

# List all clusters in a project
./list-clusters.sh invisible-infra clusters.txt

# Or for multiple projects
for project in invisible-infra invisible-sre-sandbox invisible-stage; do
    ./list-clusters.sh "$project" "clusters-${project}.txt"
done
```

This will:
- Query GCP for all GKE clusters in the specified project(s)
- Format them as: `CLUSTER_NAME PROJECT_ID REGION`
- Save to `clusters.txt` for use with `deploy-batch.sh`

## Method 2: Export from Sweet Security Dashboard

### Option A: Manual Export
1. Log into Sweet Security dashboard
2. Navigate to **Sensor** tab
3. Click the **export/download** icon (usually in the top-right of the table)
4. Export as CSV or JSON
5. Parse the file to extract cluster names

### Option B: API Export (if available)
```bash
cd deploy/sweet-security/scripts

# Set your API credentials
export SWEET_API_KEY="your-api-key"
export SWEET_SECRET="your-secret"

# Try to export via API
./export-sweet-clusters.sh
```

## Method 3: Query GCP Directly

Use `gcloud` to list all clusters across all projects:

```bash
# For a specific project
gcloud container clusters list \
    --project=invisible-infra \
    --format="table(name,location,status)"

# For all projects you have access to
for project in $(gcloud projects list --format="value(projectId)"); do
    echo "=== Project: $project ==="
    gcloud container clusters list \
        --project="$project" \
        --format="table(name,location,status)" 2>/dev/null || echo "No access or no clusters"
    echo ""
done
```

## Method 4: Create clusters.txt Manually

If you know the cluster names, create the file manually:

```bash
cat > deploy/sweet-security/clusters.txt <<EOF
# Format: CLUSTER_NAME PROJECT_ID REGION

# invisible-infra project
us-central1-inv-pipelines-08761ab1-gke invisible-infra us-central1
us-central1-test-composer-a-eb03b270-gke invisible-infra us-central1
invisible-prod-cluster invisible-infra us-central1

# invisible-sre-sandbox project
sre-onboarding invisible-sre-sandbox us-west1
sre-771-staging invisible-sre-sandbox us-west1
sre-771-development invisible-sre-sandbox us-west1

# Add more clusters...
EOF
```

## Method 5: Compare GCP vs Sweet Security

To see which clusters are missing sensors:

```bash
# 1. Get all clusters from GCP
./scripts/list-clusters.sh invisible-infra all-clusters.txt

# 2. Get clusters with sensors from Sweet Security (manual export or API)
# Save to: clusters-with-sensors.txt

# 3. Compare
comm -23 <(sort all-clusters.txt) <(sort clusters-with-sensors.txt) > missing-sensors.txt
```

## Using the Cluster List

Once you have the cluster list, use it for batch deployment:

```bash
cd deploy/sweet-security

# Set credentials
export SWEET_API_KEY="your-api-key"
export SWEET_SECRET="your-secret"
export SWEET_CLUSTER_ID="your-cluster-id"

# Deploy to all clusters
./deploy-batch.sh clusters.txt
```

## Quick Command Reference

```bash
# List clusters from GCP (single project)
gcloud container clusters list --project=PROJECT_ID --format="table(name,location,status)"

# List clusters from GCP (all accessible projects)
for p in $(gcloud projects list --format="value(projectId)"); do
    gcloud container clusters list --project="$p" --format="table(name,location,status)" 2>/dev/null
done

# Use the helper script
./scripts/list-clusters.sh PROJECT_ID output.txt
```

## Notes

- The dashboard shows "6/40" meaning 6 clusters have sensors installed
- The remaining 34 clusters need deployment
- Use `list-clusters.sh` to get the complete list from GCP
- Format: `CLUSTER_NAME PROJECT_ID REGION` (one per line)
- Use `deploy-batch.sh` to deploy to all clusters at once
