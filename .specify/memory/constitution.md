# Speckit Constitution

## MANDATORY READING

**This document is LAW. Every principle is REQUIRED, not optional.**

Before ANY work begins, you MUST:
1. Read this entire document
2. Understand each principle applies to YOUR current task
3. Plan how you will satisfy EACH requirement
4. If you cannot satisfy a requirement, STOP and inform the user

**Skipping requirements is NEVER acceptable.**

---

## Core Principles

### I. Library-First with CLI Interface

Complex business logic and reusable functionality should be extracted into standalone libraries where it makes sense. Libraries must be self-contained, independently testable, and thoroughly documented. Every library must expose functionality via CLI with text in/out protocol for consistent, scriptable interfaces.

**Checkpoint:**
- [ ] Is this functionality reusable? → Extract to library
- [ ] Does the library have CLI interface? → Add one
- [ ] Is the library independently testable? → Verify tests work in isolation

---

### II. Test-First Development

**THIS IS NOT OPTIONAL. THIS IS MANDATORY.**

Test-Driven Development is the REQUIRED development approach:

1. **Write failing tests FIRST** - before ANY implementation code
2. **Implement minimum code** to make tests pass
3. **Refactor** while keeping tests green
4. **Achieve ≥80% coverage** before considering task complete

**Violations:**
- Writing implementation before tests = VIOLATION
- "Type specs" without real assertions = VIOLATION
- Skipping tests for "simple" code = VIOLATION
- "Infrastructure doesn't need tests" = VIOLATION
- Coverage below 80% = VIOLATION

**Checkpoint before implementation:**
- [ ] Test file exists at `*.test.ts`
- [ ] Test file has real test cases (describe/it blocks)
- [ ] Tests run and FAIL (proving they test something)
- [ ] Only THEN may implementation begin

**Checkpoint after implementation:**
- [ ] All tests pass
- [ ] Coverage ≥80% on statements, branches, functions, lines
- [ ] No skipped tests
- [ ] No placeholder assertions

*Detailed standards: See [Testing Standards](../../docs/testing-standards.md)*

---

### III. Integration Testing

Integration testing is REQUIRED for validating interactions between components, services, and external systems.

**Requirements:**
- All new libraries must have contract tests
- All service-to-service communication must be tested
- All external API integrations must be validated

**Checkpoint:**
- [ ] Contract tests exist for library interfaces
- [ ] Integration tests cover service interactions
- [ ] External API calls are mocked and tested

*Detailed standards: See [Testing Standards](../../docs/testing-standards.md)*

---

### IV. Observability

Observability is REQUIRED for all production code per the [Tresic Observability & Traceability Standard](https://tresic.atlassian.net/wiki/spaces/PM/pages/753929).

**Requirements:**

1. **Trace Context Propagation**
   - W3C Trace Context (`traceparent` header) for all HTTP handlers
   - Use `ErrorContext`/`InfoContext` (not `Error`/`Info`) for automatic trace injection
   - Trace IDs in all log entries for correlation

2. **Prometheus Metrics (Decorator Pattern)**
   - Every new service MUST have an `InstrumentedService` decorator
   - Every new repository MUST have an `InstrumentedRepository` decorator
   - Record: method duration, call counts (success/error), business errors
   - Labels: `layer`, `service`, `method`, `status`, `error_type`

3. **Structured Logging**
   - JSON format with automatic PII redaction
   - Required fields: `traceId`, `service`, `level`, `message`
   - Use `observability.LoggerFromContext()` for trace-aware logging

4. **Business Error Tracking**
   - Define specific error types (e.g., `not_found`, `invalid_input`)
   - Record via `metrics.RecordBusinessError(service, method, errorType)`

**Checkpoint:**
- [ ] Handler uses `ErrorContext(ctx, ...)` not `Error(...)`
- [ ] `InstrumentedService` wrapper created and wired in `app.go`
- [ ] `InstrumentedRepository` wrapper created and wired in `app.go`
- [ ] Business errors mapped to metric labels
- [ ] README documents metrics and error types

---

### V. Documentation First

**DOCUMENTATION MUST EXIST BEFORE IMPLEMENTATION BEGINS.**

All features REQUIRE:
- Specification documents
- API documentation
- User guides (where applicable)
- Architectural decision records (for significant decisions)

**Violations:**
- "I'll document later" = VIOLATION
- "Code is self-documenting" = VIOLATION
- "It's just internal" = VIOLATION
- Missing README for new module = VIOLATION

**Checkpoint before implementation:**
- [ ] README.md exists for new directories/modules
- [ ] API documentation drafted for new functions
- [ ] Architecture documented for new components

**Checkpoint after implementation:**
- [ ] Documentation is complete and accurate
- [ ] Examples provided where helpful
- [ ] Documentation committed with code

---

### VI. Quality Standards

All code MUST be production-ready. No exceptions.

**Required for ALL code:**
- Complete error handling for ALL failure scenarios
- Input validation and sanitization
- Proper logging at appropriate levels
- Security measures (auth, authz, rate limiting)
- Resource cleanup (connections, file handles)

**Forbidden patterns:**
- `// TODO: implement` = VIOLATION
- `// FIXME` = VIOLATION
- `// Phase 2` = VIOLATION
- Empty catch blocks = VIOLATION
- `return null` without handling = VIOLATION
- Placeholder code = VIOLATION
- Incomplete implementations = VIOLATION

**Checkpoint:**
- [ ] All error paths handled
- [ ] All inputs validated
- [ ] No placeholder code
- [ ] No deferred work

*Detailed standards: See [Quality Gates](../../docs/quality-gates.md)*

---

### VII. APIs as First-Class Features

All APIs are first-class features. Developer experience = user experience.

**Requirements:**
- Document every API as if public-facing
- Include examples and error responses
- Version APIs appropriately

**Checkpoint:**
- [ ] API documented with request/response examples
- [ ] Error responses documented
- [ ] Breaking changes versioned

---

### VIII. Scope-Based Authorization

All capabilities must be built with fine-grained scopes.

**Requirements:**
- Every feature associated with permission scopes
- Every API endpoint has required scopes
- Every UI component checks permissions

**Checkpoint:**
- [ ] Scopes defined for new features
- [ ] Authorization checks implemented
- [ ] Scope documentation updated

---

### IX. Feature Flag Architecture

All new features MUST be behind feature flags.

**Requirements:**
- Feature flags for controlled rollouts
- Support ring-based deployments
- Support customer/user targeting

**Checkpoint:**
- [ ] Feature flag defined
- [ ] Default state documented
- [ ] Rollout plan specified

---

### X. Backend/Frontend Isolation

Clear architectural isolation between backend and frontend.

**Requirements:**
- Frontend NEVER directly accesses databases
- All data access through well-defined APIs
- Clear API contracts between layers

**Checkpoint:**
- [ ] No direct database access from frontend
- [ ] API layer exists between frontend and data
- [ ] Contracts documented

---

### XI. Built for Compliance

All decisions must account for compliance requirements.

**Standards to consider:**
- SOC2
- ISO27001
- HiTrust
- HIPAA

**Requirements:**
- Document compliance rationale for decisions
- Evidence of compliance consideration
- Audit trail for changes

**Checkpoint:**
- [ ] Compliance implications considered
- [ ] Decision rationale documented
- [ ] Audit requirements met

---

## Development Workflow

All development follows this specification-driven workflow:

```
1. User requirements      → Spec created
2. Feature specification  → Reviewed and approved
3. Implementation plan    → Reviewed and approved
4. Task breakdown         → Jira tickets created
5. TDD implementation     → Tests FIRST, then code
6. Quality validation     → Coverage, docs, security
7. Deployment             → Feature flagged rollout
```

**Each phase has defined deliverables and approval gates.**

---

### Git Branch Naming

All branches MUST follow this naming convention:

| Branch Type | Pattern | Example |
|------------|---------|---------|
| Feature | `feature/{JIRA-KEY}-{short-description}` | `feature/IDB-31-get-reseller-user` |
| Bugfix | `bugfix/{JIRA-KEY}-{short-description}` | `bugfix/IDB-45-fix-pagination` |
| Hotfix | `hotfix/{JIRA-KEY}-{short-description}` | `hotfix/IDB-99-critical-auth-fix` |
| Release | `release/v{version}` | `release/v1.2.0` |

**Requirements:**
- All feature/bugfix/hotfix branches MUST include the Jira ticket key
- Use lowercase with hyphens (kebab-case)
- Keep descriptions short but meaningful (3-5 words max)

**Checkpoint:**
- [ ] Branch starts with correct prefix (`feature/`, `bugfix/`, etc.)
- [ ] Jira ticket key included in branch name
- [ ] Description is kebab-case and concise

---

## Enforcement

### Before Starting ANY Task

1. Read this constitution
2. Identify which principles apply (hint: most of them)
3. Plan how to satisfy each requirement
4. If unsure, ask the user - don't skip

### During Implementation

1. Follow TDD strictly (tests first)
2. Document as you go
3. Handle all error cases
4. No shortcuts, no deferrals

### Before Marking Task Complete

1. Tests pass with ≥80% coverage
2. Documentation complete
3. No TODOs or FIXMEs
4. Quality gates pass
5. Jira ticket updated

### If You're Tempted to Skip Something

**DON'T.**

The requirement exists for a reason. If you skip it:
- You create technical debt
- You fail the user
- You make future work harder
- You prove yourself unreliable

Do the work properly or tell the user you cannot.

---

See [Contributing Guide](../../CONTRIBUTING.md) for detailed workflow procedures.
