# Quick Claude Commands Reference

## Usage in Chat
Type these commands in Claude Code sessions:

## Development Commands

### `/dev-setup`
Initialize development environment
- Checks project dependencies
- Sets up linting/formatting
- Creates environment templates

### `/code-review`
Comprehensive code review
- Analyzes current git changes
- Runs quality and security checks
- Provides improvement suggestions

### `/test-fix`
Run tests and fix failures
- Identifies test framework
- Executes tests
- Fixes failing tests automatically

### `/deploy-prep`
Prepare for deployment
- Runs all quality checks
- Executes full test suite
- Creates deployment checklist

## SRE Commands

### `/health-check`
System health assessment
- Checks service status
- Reviews error logs
- Analyzes resource usage

### `/incident-response`
Structured incident handling
- Assesses impact and severity
- Checks system metrics
- Identifies root causes

### `/performance-audit`
Performance optimization
- Analyzes application metrics
- Checks database performance
- Identifies bottlenecks

### `/infra-validate`
Infrastructure validation
- Lints Terraform/K8s files
- Checks security configs
- Validates resource limits

## Security Commands

### `/security-scan`
Comprehensive security check
- Scans for vulnerabilities
- Checks dependencies
- Reviews authentication

### `/compliance-check`
Security compliance validation
- Reviews security practices
- Checks for hardcoded secrets
- Validates encryption usage

## Git Commands

### `/git-cleanup`
Repository maintenance
- Removes merged branches
- Cleans commit history
- Optimizes repository

### `/release-prep`
Prepare software release
- Reviews changelog
- Runs full test suite
- Creates release notes

## Kubernetes Commands

### `/k8s-debug`
Debug K8s deployments
- Checks pod status/logs
- Reviews resource usage
- Validates configurations

### `/k8s-security`
K8s security assessment
- Reviews security contexts
- Checks RBAC policies
- Validates network policies

## Emergency Commands

### `/emergency-debug`
Fast production debugging
- Immediate system status
- Quick error log analysis
- Identifies mitigation steps

### `/hotfix-deploy`
Rapid hotfix deployment
- Creates hotfix branch
- Implements minimal fix
- Prepares deployment

## Quick Agent Commands

### Run Single Agent
- `/run code-refactor-cleanup` - Code quality improvements
- `/run security-performance-auditor` - Security & performance scan
- `/run infra-code-linter` - Infrastructure code validation
- `/run git-ops-expert` - Complex git operations

### Run Multiple Agents (Parallel)
- `/run security-audit` - Runs security-performance-auditor + infra-code-linter
- `/run pre-merge` - Runs code-refactor-cleanup + security-performance-auditor
- `/run infra-check` - Runs infra-code-linter + security-performance-auditor

## Common Workflows

### After Feature Implementation
```
/code-review
/test-fix (if tests fail)
/deploy-prep
```

### Before Release
```
/security-scan
/performance-audit
/release-prep
```

### Production Issue
```
/emergency-debug
/incident-response
/performance-audit (if performance related)
```

### Infrastructure Changes
```
/infra-validate
/security-scan
/deploy-prep
```

## Tips
- Commands run automatically with appropriate agents
- Most commands use multiple agents in parallel for speed
- Emergency commands prioritize speed over comprehensiveness
- Use `/help command-name` for detailed command information