# Bash History Analysis Report

## Command Frequency Analysis

### Most Used Commands (Top 15)

| Command | Frequency | Category |
|---------|-----------|----------|
| `git commit -am` | 18 | Git |
| `git push` | 16 | Git |
| `kubectl get` | 15 | Kubernetes |
| `gcloud` | 14 | GCloud |
| `docker build` | 12 | Docker |
| `pip install` | 11 | Python |
| `kubectl apply` | 10 | Kubernetes |
| `source .venv/bin/activate` | 9 | Python |
| `terraform` | 8 | Infrastructure |
| `git checkout` | 8 | Git |
| `kubectl create` | 7 | Kubernetes |
| `docker push` | 7 | Docker |
| `gcloud auth` | 6 | GCloud |
| `airflow` | 6 | Airflow |
| `helm` | 5 | Kubernetes |

## Workflow Patterns Identified

### 1. Git Commit-Push Workflow (16 occurrences)
**Pattern**: `git commit -am "message"` → `git push`

**Current manual steps**:
```bash
git commit -am "message"
git push
```

**Automation opportunity**: Single command function
```bash
qp() { git commit -am "$1" && git push; }
```

### 2. Docker Build-Tag-Push to GCR (12 occurrences)
**Pattern**: `docker build` → `docker tag` → `docker push` to GCR

**Current manual steps**:
```bash
docker build -t five-tasks-fastapi:latest .
docker tag five-tasks-fastapi:latest us-west1-docker.pkg.dev/PROJECT/REPO/IMAGE:latest
docker push us-west1-docker.pkg.dev/PROJECT/REPO/IMAGE:latest
```

**Automation opportunity**: Single function with project defaults

### 3. Python Environment Setup (9 occurrences)
**Pattern**: `python3 -m venv .venv` → `source .venv/bin/activate` → `pip install --upgrade pip` → `pip install -r requirements.txt`

**Automation opportunity**: Environment setup function

### 4. Kubernetes Deployment Workflow (10 occurrences)
**Pattern**: `kubectl create namespace` → `kubectl create deployment` → `kubectl get pods/deployments`

**Automation opportunity**: Single deployment function

### 5. GCloud Authentication Sequence (8 occurrences)
**Pattern**: `gcloud auth login` → `gcloud config set project` → `gcloud auth configure-docker`

**Automation opportunity**: Complete GCloud setup function

## High-Impact Automation Opportunities

### 1. **Git Workflow Automation** (Save: ~5 commands/day)
- Quick commit-push function
- New branch creation with upstream
- Branch sync with main/master

### 2. **Docker-GCR Integration** (Save: ~8 commands/deployment)
- Automated build-tag-push to Google Container Registry
- Project and region defaults
- Error handling and validation

### 3. **Kubernetes Management** (Save: ~6 commands/deployment)
- Quick deployment with common patterns
- Comprehensive status checking
- Namespace creation automation

### 4. **GCloud Environment Setup** (Save: ~4 commands/session)
- Complete authentication and project setup
- Docker registry configuration
- Quota project setting

### 5. **Development Environment** (Save: ~7 commands/project)
- Python virtual environment automation
- Requirements installation
- Project initialization

## Repetitive Command Sequences

### Datadog Setup (4 occurrences)
```bash
kubectl create namespace datadog
kubectl create secret generic datadog-secret --from-literal=api-key=...
kubectl apply -f datadog-agent.yaml
```

### Terraform Workflow (6 occurrences)
```bash
terraform fmt -recursive
terraform validate
terraform plan
```

### Composer DAG Upload (6 occurrences)
```bash
export DAG_PREFIX=$(gcloud composer environments describe ...)
gcloud storage cp dags/file.py "$DAG_PREFIX"
```

## Time Savings Estimation

| Workflow | Manual Steps | Automated Steps | Time Saved | Daily Usage | Daily Savings |
|----------|--------------|-----------------|------------|-------------|---------------|
| Git Commit-Push | 2 commands | 1 function | 30 sec | 5x | 2.5 min |
| Docker to GCR | 3-4 commands | 1 function | 60 sec | 2x | 2 min |
| Python Env Setup | 4-5 commands | 1 function | 90 sec | 2x | 3 min |
| K8s Deployment | 3-4 commands | 1 function | 45 sec | 3x | 2.25 min |
| GCloud Setup | 4 commands | 1 function | 60 sec | 1x | 1 min |

**Total estimated daily time savings: ~11 minutes**
**Weekly time savings: ~55 minutes**

## Recommended Implementation Priority

1. **High Priority** (Daily use, high impact):
   - Git workflow automation
   - Docker-GCR automation
   - Python environment setup

2. **Medium Priority** (Regular use):
   - Kubernetes deployment automation
   - GCloud setup automation
   - Terraform workflow

3. **Low Priority** (Occasional use):
   - Datadog setup automation
   - Composer DAG upload
   - Development project initialization

## Error-Prone Manual Processes Identified

1. **Docker tag formatting** - Frequent typos in GCR URLs
2. **Project ID switching** - Multiple projects cause confusion
3. **Virtual environment activation** - Often forgotten
4. **Kubernetes namespace creation** - Manual step often missed
5. **Git upstream setting** - New branches need manual upstream setting

## Implementation Notes

- All automation functions include error checking and helpful usage messages
- Default values are based on your most common project patterns
- Functions are designed to be backwards compatible with existing workflows
- Each function provides clear success/failure feedback