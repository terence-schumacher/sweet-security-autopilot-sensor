#!/bin/bash

# Automation Commands Based on Bash History Analysis
# Common workflows extracted from your command patterns

# =============================================================================
# GIT AUTOMATION
# =============================================================================

# Quick commit and push (used 15+ times in history)
function qp() {
    if [ -z "$1" ]; then
        echo "Usage: qp 'commit message'"
        return 1
    fi
    git add .
    git commit -m "$1"
    git push
}

# Create new branch and push upstream (used 8+ times)
function nb() {
    if [ -z "$1" ]; then
        echo "Usage: nb 'branch-name'"
        return 1
    fi
    git checkout -b "$1"
    git push -u origin "$1"
}

# Sync with main/master and merge back
function sync_branch() {
    local current_branch=$(git branch --show-current)
    git checkout main 2>/dev/null || git checkout master
    git pull
    git checkout "$current_branch"
    git merge main 2>/dev/null || git merge master
}

# =============================================================================
# DOCKER AUTOMATION
# =============================================================================

# Build and push to GCR (used 12+ times in different variations)
function docker_gcr() {
    local image_name="$1"
    local project_id="${2:-invisible-sre-sandbox}"
    local region="${3:-us-west1}"

    if [ -z "$image_name" ]; then
        echo "Usage: docker_gcr IMAGE_NAME [PROJECT_ID] [REGION]"
        return 1
    fi

    echo "Building and pushing $image_name to $region-docker.pkg.dev/$project_id/terence-test/$image_name:latest"

    # Build
    docker build -t "$image_name:latest" .

    # Tag for GCR
    docker tag "$image_name:latest" "$region-docker.pkg.dev/$project_id/terence-test/$image_name:latest"

    # Push
    docker push "$region-docker.pkg.dev/$project_id/terence-test/$image_name:latest"

    echo "Image pushed successfully!"
}

# =============================================================================
# GCLOUD AUTOMATION
# =============================================================================

# Complete GCloud setup (auth + project + docker config)
function gcloud_init() {
    local project_id="$1"

    if [ -z "$project_id" ]; then
        echo "Usage: gcloud_init PROJECT_ID"
        return 1
    fi

    echo "Setting up GCloud for project: $project_id"

    gcloud auth login
    gcloud config set project "$project_id"
    gcloud auth configure-docker us-west1-docker.pkg.dev
    gcloud auth application-default set-quota-project "$project_id"

    echo "GCloud setup complete for $project_id"
}

# Get GKE credentials (used 5+ times)
function gke_connect() {
    local cluster_name="$1"
    local zone="${2:-us-west1}"
    local project_id="${3:-invisible-sre-sandbox}"

    if [ -z "$cluster_name" ]; then
        echo "Usage: gke_connect CLUSTER_NAME [ZONE] [PROJECT_ID]"
        return 1
    fi

    gcloud container clusters get-credentials "$cluster_name" --zone="$zone" --project="$project_id"
    kubectl cluster-info
}

# Upload DAG to Composer (used 6+ times)
function composer_upload() {
    local dag_file="$1"
    local env_name="${2:-test-composer-alerts}"
    local location="${3:-us-central1}"

    if [ -z "$dag_file" ]; then
        echo "Usage: composer_upload DAG_FILE [ENV_NAME] [LOCATION]"
        return 1
    fi

    local dag_prefix=$(gcloud composer environments describe "$env_name" --location "$location" --format="value(config.dagGcsPrefix)")
    gcloud storage cp "$dag_file" "$dag_prefix"
    echo "DAG uploaded to $dag_prefix"
}

# =============================================================================
# KUBERNETES AUTOMATION
# =============================================================================

# Quick deployment with common settings (used 10+ times in variations)
function k8s_quick_deploy() {
    local app_name="$1"
    local image="$2"
    local namespace="${3:-default}"

    if [ -z "$app_name" ] || [ -z "$image" ]; then
        echo "Usage: k8s_quick_deploy APP_NAME IMAGE [NAMESPACE]"
        return 1
    fi

    # Create namespace if it doesn't exist
    kubectl create namespace "$namespace" --dry-run=client -o yaml | kubectl apply -f -

    # Create deployment
    kubectl create deployment "$app_name" --image="$image" -n "$namespace"

    # Check status
    kubectl get pods -n "$namespace"
    kubectl get deployments -n "$namespace"
}

# Get comprehensive k8s status (used 20+ times in various forms)
function k8s_status() {
    local namespace="${1:-default}"

    echo "=== Cluster Info ==="
    kubectl cluster-info

    echo -e "\n=== Nodes ==="
    kubectl get nodes

    echo -e "\n=== Pods (namespace: $namespace) ==="
    kubectl get pods -n "$namespace"

    echo -e "\n=== Deployments (namespace: $namespace) ==="
    kubectl get deployments -n "$namespace"

    echo -e "\n=== Services (namespace: $namespace) ==="
    kubectl get services -n "$namespace"
}

# =============================================================================
# PYTHON ENVIRONMENT AUTOMATION
# =============================================================================

# Complete Python environment setup (used 10+ times)
function py_env() {
    local python_version="${1:-3.12}"

    echo "Setting up Python $python_version environment"

    # Set Python version if using pyenv
    if command -v pyenv >/dev/null 2>&1; then
        pyenv local "$python_version" 2>/dev/null || echo "Python $python_version not installed in pyenv"
    fi

    # Create and activate virtual environment
    python3 -m venv .venv
    source .venv/bin/activate

    # Upgrade pip and install requirements
    pip install --upgrade pip

    if [ -f "requirements.txt" ]; then
        pip install -r requirements.txt
        echo "Requirements installed from requirements.txt"
    else
        echo "No requirements.txt found - skipping package installation"
    fi

    echo "Python environment ready!"
}

# =============================================================================
# TERRAFORM AUTOMATION
# =============================================================================

# Complete Terraform workflow (used 8+ times)
function tf_workflow() {
    echo "Running Terraform workflow..."

    # Format and validate
    terraform fmt -recursive
    terraform validate

    if [ $? -eq 0 ]; then
        echo "Terraform files are valid. Running plan..."
        terraform plan
    else
        echo "Terraform validation failed!"
        return 1
    fi
}

# =============================================================================
# DATADOG/MONITORING AUTOMATION
# =============================================================================

# Setup Datadog secrets for K8s (used 4+ times)
function dd_k8s_setup() {
    local api_key="$1"
    local app_key="$2"
    local namespace="${3:-datadog}"

    if [ -z "$api_key" ] || [ -z "$app_key" ]; then
        echo "Usage: dd_k8s_setup DD_API_KEY DD_APP_KEY [NAMESPACE]"
        return 1
    fi

    # Create namespace
    kubectl create namespace "$namespace" --dry-run=client -o yaml | kubectl apply -f -

    # Create secret
    kubectl create secret generic datadog-secret \
        --from-literal=api-key="$api_key" \
        --from-literal=app-key="$app_key" \
        --namespace="$namespace" \
        --dry-run=client -o yaml | kubectl apply -f -

    echo "Datadog secrets created in namespace: $namespace"
}

# =============================================================================
# DEVELOPMENT WORKFLOW AUTOMATION
# =============================================================================

# Complete development setup for a new project
function dev_init() {
    local project_name="$1"

    if [ -z "$project_name" ]; then
        echo "Usage: dev_init PROJECT_NAME"
        return 1
    fi

    echo "Initializing development environment for: $project_name"

    # Create project directory
    mkdir -p "$project_name" && cd "$project_name"

    # Initialize git
    git init

    # Create basic files
    touch README.md .gitignore

    # Create Python virtual environment if Python project
    if command -v python3 >/dev/null 2>&1; then
        python3 -m venv .venv
        echo ".venv/" >> .gitignore
        echo "requirements.txt" >> .gitignore
        touch requirements.txt
    fi

    # Initial commit
    git add .
    git commit -m "Initial commit for $project_name"

    echo "Development environment initialized!"
    echo "Next steps:"
    echo "1. Add remote: git remote add origin <repo-url>"
    echo "2. Push: git push -u origin main"
    echo "3. Activate Python env: source .venv/bin/activate"
}

# =============================================================================
# UTILITY FUNCTIONS
# =============================================================================

# Show all available automation commands
function auto_help() {
    echo "Available automation commands:"
    echo ""
    echo "Git:"
    echo "  qp 'message'           - Quick commit and push"
    echo "  nb 'branch'            - New branch and push upstream"
    echo "  sync_branch            - Sync with main/master"
    echo ""
    echo "Docker:"
    echo "  docker_gcr IMAGE       - Build and push to GCR"
    echo ""
    echo "GCloud:"
    echo "  gcloud_init PROJECT    - Complete GCloud setup"
    echo "  gke_connect CLUSTER    - Connect to GKE cluster"
    echo "  composer_upload DAG    - Upload DAG to Composer"
    echo ""
    echo "Kubernetes:"
    echo "  k8s_quick_deploy APP IMAGE - Quick deployment"
    echo "  k8s_status [NAMESPACE] - Comprehensive cluster status"
    echo ""
    echo "Python:"
    echo "  py_env [VERSION]       - Setup Python environment"
    echo ""
    echo "Terraform:"
    echo "  tf_workflow            - Format, validate, plan"
    echo ""
    echo "Monitoring:"
    echo "  dd_k8s_setup API APP   - Setup Datadog for K8s"
    echo ""
    echo "Development:"
    echo "  dev_init PROJECT       - Initialize new project"
    echo ""
    echo "Use 'auto_help' to see this help again"
}

# Load message
echo "Automation commands loaded! Use 'auto_help' for available commands."