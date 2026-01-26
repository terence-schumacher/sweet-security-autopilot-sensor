# Agent Workflow Configurations

## Agent Combinations for Common Tasks

### Development Workflows

#### Feature Implementation Complete
**Trigger**: After completing a new feature
**Agents**: `code-refactor-cleanup` → `security-performance-auditor` → `infra-code-linter`
**Order**: Sequential
```
1. code-refactor-cleanup: Clean up implementation, remove duplication
2. security-performance-auditor: Scan for security issues and performance problems
3. infra-code-linter: Validate any config/infrastructure changes
```

#### Pre-Merge Checklist
**Trigger**: Before merging PR
**Agents**: `code-refactor-cleanup` + `security-performance-auditor` (parallel)
**Order**: Parallel
```
// Run both agents simultaneously
- code-refactor-cleanup: Final code quality pass
- security-performance-auditor: Security and performance validation
```

### SRE Workflows

#### Production Issue Investigation
**Trigger**: When performance issues reported
**Agents**: `security-performance-auditor` → `git-ops-expert`
**Order**: Sequential
```
1. security-performance-auditor: Identify performance bottlenecks and security issues
2. git-ops-expert: Check recent changes that might have caused issues
```

#### Infrastructure Changes
**Trigger**: After modifying Terraform/K8s configs
**Agents**: `infra-code-linter` → `security-performance-auditor`
**Order**: Sequential
```
1. infra-code-linter: Validate infrastructure code syntax and best practices
2. security-performance-auditor: Check security implications of infrastructure changes
```

### Security Workflows

#### Security Audit
**Trigger**: Regular security review or before major release
**Agents**: `security-performance-auditor` + `infra-code-linter` (parallel)
**Order**: Parallel
```
// Run both agents simultaneously for comprehensive coverage
- security-performance-auditor: Application and runtime security
- infra-code-linter: Infrastructure and configuration security
```

#### Authentication Module Changes
**Trigger**: After modifying auth/authorization code
**Agents**: `security-performance-auditor` → `code-refactor-cleanup`
**Order**: Sequential
```
1. security-performance-auditor: Deep security analysis of auth changes
2. code-refactor-cleanup: Clean up auth code for maintainability
```

### Git Operations

#### Repository Maintenance
**Trigger**: Weekly/monthly repository cleanup
**Agents**: `git-ops-expert` → `code-refactor-cleanup`
**Order**: Sequential
```
1. git-ops-expert: Clean branches, optimize repo, fix history issues
2. code-refactor-cleanup: Clean up any code quality issues found
```

#### Complex Merge Resolution
**Trigger**: When facing complex merge conflicts
**Agents**: `git-ops-expert` only
**Order**: Single
```
git-ops-expert: Handle complex branching, merging, and conflict resolution
```

### Emergency Workflows

#### Production Incident
**Trigger**: Critical production issue
**Agents**: `security-performance-auditor` (immediate) → others as needed
**Order**: Sequential with fast-track
```
1. security-performance-auditor: Immediate performance and security analysis
2. Follow-up agents based on findings
```

#### Hotfix Deployment
**Trigger**: Emergency code fix needed
**Agents**: `security-performance-auditor` + `infra-code-linter` (parallel, minimal scope)
**Order**: Parallel
```
// Fast parallel execution with minimal checks
- security-performance-auditor: Quick security validation
- infra-code-linter: Essential infrastructure validation only
```

## Agent Specialization Matrix

| Task Type | Primary Agent | Secondary Agent | Use Case |
|-----------|---------------|-----------------|----------|
| Code Quality | code-refactor-cleanup | - | After feature development |
| Security Issues | security-performance-auditor | - | Regular security scans |
| Performance Problems | security-performance-auditor | - | When app is slow |
| Infrastructure Changes | infra-code-linter | security-performance-auditor | Terraform/K8s updates |
| Git Problems | git-ops-expert | - | Complex git operations |
| Memory Leaks | security-performance-auditor | code-refactor-cleanup | Performance debugging |
| Config Validation | infra-code-linter | - | Docker/YAML/Terraform files |
| Repository Cleanup | git-ops-expert | code-refactor-cleanup | Maintenance tasks |

## Proactive Agent Usage

### Automatic Triggers
- **code-refactor-cleanup**: Auto-trigger after any significant code changes
- **security-performance-auditor**: Auto-trigger after auth/security related changes
- **infra-code-linter**: Auto-trigger after config file modifications
- **git-ops-expert**: Manual trigger only (complex operations)

### Best Practices
1. Use agents in parallel when possible for speed
2. Security-performance-auditor should always run after auth changes
3. Always run infra-code-linter after infrastructure modifications
4. Use git-ops-expert for complex git operations only
5. Run code-refactor-cleanup before major code reviews

### Performance Optimization
- Run lightweight agents first in sequences
- Use parallel execution when agents don't depend on each other
- Cache agent results when possible
- Skip redundant scans if no relevant changes detected