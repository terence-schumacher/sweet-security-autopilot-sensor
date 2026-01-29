#!/bin/bash
# List all GKE clusters from GCP and optionally compare with Sweet Security dashboard
# Usage: ./list-clusters.sh [PROJECT_ID] [OUTPUT_FILE]

set -e

PROJECT_ID=${1:-""}
OUTPUT_FILE=${2:-"clusters.txt"}

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}GKE Cluster Listing Script${NC}"
echo "================================"
echo ""

if [ -z "$PROJECT_ID" ]; then
    echo "Usage: $0 <PROJECT_ID> [OUTPUT_FILE]"
    echo ""
    echo "Example:"
    echo "  $0 invisible-infra clusters.txt"
    echo ""
    echo "This will list all GKE clusters in the specified project and save to clusters.txt"
    echo ""
    echo "Available projects:"
    gcloud projects list --format="table(projectId,name)" 2>/dev/null | head -20
    exit 1
fi

echo "Listing GKE clusters in project: $PROJECT_ID"
echo "Output file: $OUTPUT_FILE"
echo ""

# List all GKE clusters across all regions
echo "Fetching clusters..."
CLUSTERS=$(gcloud container clusters list \
    --project="$PROJECT_ID" \
    --format="table(name,location,status)" 2>/dev/null)

if [ -z "$CLUSTERS" ]; then
    echo "No clusters found or error accessing project."
    exit 1
fi

# Count clusters
TOTAL=$(echo "$CLUSTERS" | tail -n +2 | wc -l | tr -d ' ')
echo "Found $TOTAL clusters"
echo ""

# Save to file in format: CLUSTER_NAME PROJECT_ID REGION
echo "Saving to $OUTPUT_FILE..."
echo "# GKE Clusters for Sweet Security Deployment" > "$OUTPUT_FILE"
echo "# Format: CLUSTER_NAME PROJECT_ID REGION" >> "$OUTPUT_FILE"
echo "# Generated: $(date)" >> "$OUTPUT_FILE"
echo "" >> "$OUTPUT_FILE"

# Parse and format clusters
echo "$CLUSTERS" | tail -n +2 | while IFS= read -r line; do
    if [ -n "$line" ]; then
        CLUSTER_NAME=$(echo "$line" | awk '{print $1}')
        LOCATION=$(echo "$line" | awk '{print $2}')
        
        # Determine if location is a region or zone
        if [[ "$LOCATION" == *"-"* ]]; then
            # Extract region from zone (e.g., us-west1-a -> us-west1)
            REGION=$(echo "$LOCATION" | sed 's/-[a-z]$//')
        else
            REGION="$LOCATION"
        fi
        
        echo "$CLUSTER_NAME $PROJECT_ID $REGION" >> "$OUTPUT_FILE"
    fi
done

echo -e "${GREEN}✓${NC} Cluster list saved to $OUTPUT_FILE"
echo ""
echo "First 10 clusters:"
head -n 13 "$OUTPUT_FILE" | tail -n +5
echo ""
echo "Use this file with deploy-batch.sh:"
echo "  ./deploy-batch.sh $OUTPUT_FILE"
