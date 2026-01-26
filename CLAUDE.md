# Claude Code Commands and Workflows

## Software Engineering Commands

### `/dev-setup`
**Purpose**: Initialize development environment for a new project
**Actions**:
- Check for package.json, requirements.txt, go.mod, etc.
- Install dependencies
- Set up git hooks if available
- Run initial linting and formatting
- Create .env.example if needed

### `/code-review`
**Purpose**: Comprehensive code review of current changes
**Actions**:
- Run git diff to see changes
- Use code-refactor-cleanup agent for quality improvements
- Run linting and type checking
- Check for security issues
- Suggest optimizations

### `/test-fix`
**Purpose**: Run tests and fix any failures
**Actions**:
- Identify test command (npm test, pytest, go test, etc.)
- Run tests and capture failures
- Analyze failures and implement fixes
- Re-run tests to verify fixes
- Update test coverage if needed

### `/deploy-prep`
**Purpose**: Prepare code for deployment
**Actions**:
- Run all linting and type checks
- Execute full test suite
- Check for security vulnerabilities
- Review recent commits
- Create deployment checklist

### `/debug-issue`
**Purpose**: Systematic debugging workflow
**Actions**:
- Analyze error logs and stack traces
- Check recent changes that might cause issues
- Review relevant code sections
- Suggest debugging steps
- Implement fixes if root cause identified

## Site Reliability Engineering Commands

### `/health-check`
**Purpose**: Comprehensive system health assessment
**Actions**:
- Check service status and uptime
- Review recent logs for errors
- Analyze resource usage (CPU, memory, disk)
- Check network connectivity
- Review monitoring alerts

### `/incident-response`
**Purpose**: Structured incident response workflow
**Actions**:
- Assess impact and severity
- Check system metrics and logs
- Identify potential root causes
- Document findings
- Suggest mitigation steps

### `/performance-audit`
**Purpose**: Performance analysis and optimization
**Actions**:
- Use security-performance-auditor agent
- Analyze application metrics
- Check database query performance
- Review API response times
- Identify bottlenecks

### `/infra-validate`
**Purpose**: Validate infrastructure code and configurations
**Actions**:
- Use infra-code-linter agent on Terraform, K8s manifests
- Check security configurations
- Validate resource limits and requests
- Review network policies
- Check compliance requirements

### `/rollback-plan`
**Purpose**: Create rollback strategy for deployments
**Actions**:
- Document current state
- Identify rollback points
- Create step-by-step rollback procedure
- Test rollback in staging if possible
- Prepare monitoring during rollback

## Security Commands

### `/security-scan`
**Purpose**: Comprehensive security assessment
**Actions**:
- Use security-performance-auditor agent
- Scan for common vulnerabilities (SQL injection, XSS)
- Check dependency vulnerabilities
- Review authentication and authorization
- Analyze secrets management

### `/compliance-check`
**Purpose**: Check compliance with security standards
**Actions**:
- Review code for security best practices
- Check for hardcoded secrets
- Validate encryption usage
- Review access controls
- Generate compliance report

## Git and DevOps Commands

### `/git-cleanup`
**Purpose**: Clean up git repository and history
**Actions**:
- Use git-ops-expert agent
- Remove merged branches
- Clean up commit history if needed
- Optimize repository size
- Update branch protection rules

### `/ci-cd-fix`
**Purpose**: Debug and fix CI/CD pipeline issues
**Actions**:
- Analyze pipeline logs
- Check configuration files
- Validate environment variables
- Test pipeline steps locally
- Suggest improvements

### `/release-prep`
**Purpose**: Prepare for software release
**Actions**:
- Review changelog and version bumps
- Run full test suite
- Check for breaking changes
- Validate migration scripts
- Create release notes

## Monitoring and Observability Commands

### `/logs-analyze`
**Purpose**: Analyze application logs for issues
**Actions**:
- Search for error patterns in logs
- Identify unusual activity
- Correlate errors with deployments
- Suggest log improvements
- Create monitoring alerts

### `/metrics-review`
**Purpose**: Review system and application metrics
**Actions**:
- Analyze performance trends
- Identify anomalies
- Check SLA compliance
- Review capacity planning
- Suggest optimization areas

## Kubernetes Commands

### `/k8s-debug`
**Purpose**: Debug Kubernetes deployment issues
**Actions**:
- Check pod status and logs
- Review resource usage
- Validate configurations
- Check network policies
- Analyze events

### `/k8s-security`
**Purpose**: Security assessment for Kubernetes workloads
**Actions**:
- Review security contexts
- Check RBAC policies
- Validate network policies
- Scan for vulnerabilities
- Review secrets management

## Emergency Commands

### `/emergency-debug`
**Purpose**: Fast-track debugging for production issues
**Actions**:
- Immediate system status check
- Recent error log analysis
- Quick performance metrics review
- Identify immediate mitigation steps
- Create incident timeline

### `/hotfix-deploy`
**Purpose**: Rapid hotfix deployment workflow
**Actions**:
- Create hotfix branch
- Implement minimal viable fix
- Run essential tests only
- Prepare deployment commands
- Create rollback plan

## Usage Notes

- Commands should be run with `/command-name`
- Most commands will use multiple agents in parallel for efficiency
- Always run linting and testing after code changes
- Security scans should be run regularly, especially before releases
- Use emergency commands only for production issues requiring immediate attention

## Agent Usage Patterns

- **code-refactor-cleanup**: After feature implementation, before PR merge
- **security-performance-auditor**: Regular security scans, after auth changes
- **infra-code-linter**: After infrastructure code changes
- **git-ops-expert**: Complex git operations, repository maintenance