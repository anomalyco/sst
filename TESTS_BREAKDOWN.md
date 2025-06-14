# SST Testing Strategy Breakdown

## Current Test Status
- ✅ **4 packages** have unit tests (11 test files total)
- ❌ **76+ packages** missing unit tests
- ⚠️ **CRITICAL GAP**: **NO integration tests** exist for real AWS infrastructure
- ⚠️ **NO CI/CD** pipeline exists
- 🎯 **Target**: Comprehensive test coverage for critical functionality + real AWS testing

## Phase 1: Core Business Logic (High Priority) ✅ COMPLETED

### 1.1 Project Management (`pkg/project/`) ✅ COMPLETED
**Files to test**: `project.go`, `create.go`, `install.go`, `run.go`, `add.go`, `env.go`

**Step-by-step:**
1. ✅ **COMPLETED** Create `pkg/project/project_test.go`
   - ✅ Test project detection and validation
   - ✅ Test project configuration loading
   - ✅ Test path resolution functions
   - ✅ Test stage name validation
   - ✅ Test error types

2. ✅ **COMPLETED** Create `pkg/project/create_test.go`
   - ✅ Test project creation with different templates
   - ✅ Test directory structure creation
   - ✅ Test error handling for invalid inputs

3. ✅ **COMPLETED** Create `pkg/project/install_test.go`
   - ✅ Test dependency installation
   - ✅ Test package manager detection (npm, bun, etc.)
   - ✅ Mock external command execution

4. ✅ **COMPLETED** Create `pkg/project/run_test.go`
   - ✅ Test command execution entry point
   - ✅ Test environment variable handling
   - ✅ Test process spawning and monitoring
   - ✅ Test protected stage validation
   - ✅ Test command validation (deploy, remove, diff, refresh)
   - ✅ Test StackInput structure validation

**Test utilities needed:**
- Mock filesystem (`afero` or similar)
- Mock command execution
- Temporary directory helpers

### 1.2 Runtime Management (`pkg/runtime/`) ✅ COMPLETED
**Files to test**: `runtime.go`, `node/`, `python/`, `rust/`, `golang/`, `worker/`

**Step-by-step:**
1. ✅ **COMPLETED** Create `pkg/runtime/runtime_test.go`
   - ✅ Test runtime detection
   - ✅ Test runtime version parsing
   - ✅ Test runtime compatibility checks

2. ✅ **COMPLETED** Create `pkg/runtime/node/node_test.go`
   - ✅ Test Node.js version detection and matching
   - ✅ Test build process with JS and TypeScript handlers
   - ✅ Test file extension resolution (.js, .ts, .jsx, .tsx, .mjs, .cjs, .mts, .cts)
   - ✅ Test properties JSON unmarshaling and configuration
   - ✅ Test worker creation and management
   - ✅ Test error handling for missing handlers
   - ✅ Test concurrency configuration

3. ✅ **COMPLETED** Create `pkg/runtime/python/python_test.go`
   - ✅ Test Python version detection
   - ✅ Test requirements.txt parsing
   - ✅ Test virtual environment handling

4. ✅ **COMPLETED** Create `pkg/runtime/golang/golang_test.go`
   - ✅ Test Go module detection
   - ✅ Test build configuration
   - ✅ Test dependency resolution
   - ✅ Test runtime matching for "go" runtime string
   - ✅ Test ShouldRebuild functionality for .go files
   - ✅ Test Build method with dev/production modes and architecture options
   - ✅ Test Worker creation and log handling
   - ✅ Test properties parsing through Build method

**Test utilities needed:**
- Mock runtime binaries
- Fake version outputs
- Mock build processes

### 1.3 State Management (`pkg/state/`) ✅ COMPLETED
**Files to test**: `state.go`, `decrypt.go`

**Step-by-step:**
1. ✅ **COMPLETED** Create `pkg/state/state_test.go`
   - ✅ Test state persistence through Remove and Repair functions
   - ✅ Test state loading and saving via checkpoint modifications
   - ✅ Test state validation with mutation types and dependency cleanup

2. ✅ **COMPLETED** Create `pkg/state/decrypt_test.go`
   - ✅ Test encryption/decryption
   - ✅ Test key management
   - ✅ Test error handling for corrupted data

**Test utilities needed:**
- Mock encryption keys
- Temporary state files
- Mock AWS/cloud credentials

## Phase 2: CLI Commands (High Priority) - **NEXT PHASE**

### 2.1 Main CLI (`cmd/sst/`) ✅ COMPLETED
**Files to test**: `main.go`, `deploy.go`, `init.go`, `remove.go`, `upgrade.go`

**Step-by-step:**
1. ✅ **COMPLETED** Create `cmd/sst/main_test.go`
   - ✅ Test command parsing
   - ✅ Test flag handling
   - ✅ Test help output
   - ✅ Test root command structure and configuration
   - ✅ Test all major CLI commands (init, dev, deploy, secret, shell, etc.)
   - ✅ Test flag types and validation
   - ✅ Test command descriptions and examples
   - ✅ Test hidden commands and telemetry configuration
   - ✅ Test context handling and version management

2. ✅ **COMPLETED** Create `cmd/sst/deploy_test.go`
   - ✅ Test deployment workflow structure and configuration
   - ✅ Test flag handling (target, continue, dev)
   - ✅ Test target parsing logic for comma-separated components
   - ✅ Test command integration and documentation
   - ✅ Test concurrency environment variable documentation
   - ✅ Test error handling patterns

3. ✅ **COMPLETED** Create `cmd/sst/init_test.go`
   - ✅ Test project initialization and template detection
   - ✅ Test framework detection (Next.js, React Router, Astro, Angular, etc.)
   - ✅ Test fileContains helper function with various scenarios
   - ✅ Test template detection priority order and logic
   - ✅ Test error handling for existing projects and edge cases

4. ✅ **COMPLETED** Create `cmd/sst/remove_test.go`
   - ✅ Test resource cleanup command structure
   - ✅ Test confirmation prompts and flag handling
   - ✅ Test target parsing and validation
   - ✅ Test command documentation and examples

**Test utilities needed:**
- CLI testing framework (cobra testing)
- Mock cloud providers
- Capture stdout/stderr

### 2.2 CLI Utilities (`cmd/sst/cli/`) ✅ COMPLETED
**Files to test**: `cli.go`, `project.go`

**Step-by-step:**
1. ✅ **COMPLETED** Create `cmd/sst/cli/cli_test.go`
   - ✅ Test CLI initialization and configuration
   - ✅ Test argument parsing and flag handling
   - ✅ Test command path management
   - ✅ Test ArgumentList string formatting
   - ✅ Test stage guessing logic
   - ✅ Test command initialization

2. ✅ **COMPLETED** Create `cmd/sst/cli/project_test.go`
   - ✅ Test project context detection and discovery
   - ✅ Test workspace handling and configuration loading
   - ✅ Test log configuration and initialization
   - ✅ Test error scenarios for project initialization

## Phase 3: Server & Resources (Medium Priority) - **NEXT PHASE**

### 3.1 Server Core (`pkg/server/`) ✅ COMPLETED
**Files to test**: `server.go`, `client.go`

**Step-by-step:**
1. ✅ **COMPLETED** Create `pkg/server/server_test.go`
   - ✅ Test server startup/shutdown
   - ✅ Test request handling
   - ✅ Test middleware
   - ✅ Test port assignment functionality
   - ✅ Test RPC endpoint HTTP method validation
   - ✅ Test HttpConn implementation
   - ✅ Test server file path resolution
   - ✅ Test integration with project configuration

2. ✅ **COMPLETED** Create `pkg/server/client_test.go`
   - ✅ Test client connections
   - ✅ Test request/response cycles
   - ✅ Test error handling
   - ✅ Test server discovery functionality
   - ✅ Test registry operations

### 3.2 AWS Resources (`pkg/server/resource/`)
**Files to test**: All AWS resource handlers

**Step-by-step:**
1. **NEXT STEP** Create `pkg/server/resource/aws_test.go`
   - Test AWS client initialization
   - Test credential handling
   - Test region selection

2. Create individual test files for each resource type:
   - `aws-function_test.go`
   - `aws-bucket_test.go`
   - `aws-distribution_test.go`
   - etc.

**Test utilities needed:**
- AWS SDK mocks (`aws-sdk-go-v2/aws/testing`)
- Mock AWS responses
- Test AWS credentials

## Phase 4: Utilities & Infrastructure (Medium Priority)

### 4.1 Process Management (`pkg/process/`)
**Step-by-step:**
1. Create `pkg/process/process_test.go`
   - Test process spawning
   - Test process monitoring
   - Test signal handling
   - Test cross-platform behavior

### 4.2 Tunneling (`pkg/tunnel/`)
**Step-by-step:**
1. Create `pkg/tunnel/tunnel_test.go`
   - Test tunnel creation
   - Test proxy functionality
   - Test connection handling

### 4.3 JavaScript/NPM Utilities (`pkg/js/`, `pkg/npm/`)
**Step-by-step:**
1. Create `pkg/js/js_test.go`
   - Test JavaScript parsing
   - Test module resolution
   - Test bundling

2. Create `pkg/npm/npm_test.go`
   - Test package.json operations
   - Test dependency resolution
   - Test npm command execution

## Phase 5: Pulumi Testing Strategy (HIGH PRIORITY - MISSING)

⚠️ **CRITICAL GAP**: No comprehensive testing exists for SST's Pulumi-based infrastructure components.

SST uses Pulumi extensively for infrastructure provisioning. We need to implement all three types of Pulumi testing:

### 5.1 Pulumi Unit Tests (TypeScript Platform Components)
**Location**: `platform/test/components/pulumi/`
**Purpose**: Test SST components in isolation using Pulumi mocks

**Step-by-step:**
1. **Create `platform/test/components/pulumi/` directory structure:**
   ```
   platform/test/components/pulumi/
   ├── aws/
   │   ├── function.test.ts
   │   ├── bucket.test.ts
   │   ├── apigateway.test.ts
   │   ├── cluster.test.ts
   │   ├── auth.test.ts
   │   └── helpers/
   │       ├── pulumi-mocks.ts
   │       └── test-utils.ts
   ├── cloudflare/
   │   ├── worker.test.ts
   │   ├── static-site.test.ts
   │   └── ssr-site.test.ts
   └── shared/
       ├── component.test.ts
       ├── naming.test.ts
       └── linkable.test.ts
   ```

2. **Create `platform/test/components/pulumi/helpers/pulumi-mocks.ts`**
   - Standardized Pulumi mocks for all SST components
   - Mock AWS/Cloudflare provider responses
   - Helper functions for testing component properties
   - Resource validation utilities

3. **Create `platform/test/components/pulumi/aws/function.test.ts`**
   - Test AWS Function component creation and configuration
   - Verify runtime selection and build process
   - Test environment variables and linking
   - Test IAM role and policy generation
   - Test VPC configuration
   - Test timeout and memory settings

4. **Create `platform/test/components/pulumi/aws/bucket.test.ts`** ✅ **PARTIALLY EXISTS**
   - Extend existing bucket.test.ts with comprehensive tests
   - Test bucket policies and CORS configuration
   - Test public/private bucket settings
   - Test bucket notifications and subscribers
   - Test bucket versioning and lifecycle rules

5. **Create `platform/test/components/pulumi/aws/apigateway.test.ts`**
   - Test API Gateway v1 and v2 components
   - Test route configuration and integration
   - Test authorizers and authentication
   - Test custom domains and certificates
   - Test CORS and request/response transformations

6. **Create `platform/test/components/pulumi/aws/cluster.test.ts`**
   - Test ECS/Fargate cluster configuration
   - Test service definitions and task configurations
   - Test load balancer integration
   - Test auto-scaling settings
   - Test VPC and security group configuration

**Test utilities needed:**
- Pulumi runtime mocks with realistic AWS responses
- Component property validation helpers
- Resource dependency verification
- Mock filesystem for asset handling

### 5.2 Pulumi Property Tests (Infrastructure Validation)
**Location**: `platform/test/policies/`
**Purpose**: Define and enforce infrastructure compliance rules

**Step-by-step:**
1. **Create `platform/test/policies/` directory structure:**
   ```
   platform/test/policies/
   ├── aws/
   │   ├── security-policies.ts
   │   ├── cost-optimization.ts
   │   ├── compliance.ts
   │   └── best-practices.ts
   ├── cloudflare/
   │   ├── security-policies.ts
   │   └── performance.ts
   └── shared/
       ├── naming-conventions.ts
       └── resource-limits.ts
   ```

2. **Create `platform/test/policies/aws/security-policies.ts`**
   - Enforce S3 bucket encryption and public access restrictions
   - Validate IAM roles follow least privilege principle
   - Ensure Lambda functions are not publicly accessible
   - Verify VPC security groups don't allow unrestricted access
   - Check that RDS instances are not publicly accessible

3. **Create `platform/test/policies/aws/cost-optimization.ts`**
   - Enforce resource tagging for cost tracking
   - Validate instance types are appropriate for workload
   - Check for unused resources (orphaned EIPs, volumes)
   - Ensure auto-scaling is configured appropriately

4. **Create `platform/test/policies/aws/compliance.ts`**
   - Enforce encryption at rest and in transit
   - Validate backup and disaster recovery configurations
   - Check logging and monitoring requirements
   - Ensure compliance with organizational standards

5. **Create `platform/test/policies/shared/naming-conventions.ts`**
   - Validate resource naming follows SST conventions
   - Ensure consistent tagging across resources
   - Check resource prefixes and suffixes

**Policy enforcement:**
- Run policies during `sst deploy` and `sst dev`
- Integrate with CI/CD pipelines
- Provide clear violation messages and remediation guidance

### 5.3 Pulumi Integration Tests (Real Infrastructure)
**Location**: `test/integration/pulumi/`
**Purpose**: Test actual infrastructure deployment and functionality

**Step-by-step:**
1. **Create `test/integration/pulumi/` directory structure:**
   ```
   test/integration/pulumi/
   ├── aws/
   │   ├── basic-deployment.test.go
   │   ├── function-deployment.test.go
   │   ├── api-deployment.test.go
   │   ├── full-stack.test.go
   │   └── fixtures/
   │       ├── simple-function/
   │       ├── api-with-auth/
   │       └── full-app/
   ├── cloudflare/
   │   ├── worker-deployment.test.go
   │   ├── static-site.test.go
   │   └── fixtures/
   │       ├── simple-worker/
   │       └── static-site/
   ├── helpers/
   │   ├── pulumi-integration.go
   │   ├── aws-setup.go
   │   ├── cloudflare-setup.go
   │   └── cleanup.go
   └── examples/
       ├── aws-api.test.go
       ├── aws-nextjs.test.go
       └── cloudflare-worker.test.go
   ```

2. **Create `test/integration/pulumi/helpers/pulumi-integration.go`**
   - Pulumi integration test framework wrapper
   - SST-specific test utilities and helpers
   - Resource validation and runtime testing functions
   - Cleanup and teardown automation

3. **Create `test/integration/pulumi/aws/basic-deployment.test.go`**
   ```go
   func TestBasicAWSDeployment(t *testing.T) {
       integration.ProgramTest(t, &integration.ProgramTestOptions{
           Dir: path.Join("fixtures", "simple-function"),
           Config: map[string]string{
               "aws:region": "us-east-1",
           },
           ExtraRuntimeValidation: func(t *testing.T, stack integration.RuntimeValidationStackInfo) {
               // Validate Lambda function exists and is invokable
               // Test function environment variables
               // Verify IAM roles and policies
           },
       })
   }
   ```

4. **Create `test/integration/pulumi/aws/function-deployment.test.go`**
   - Deploy various function configurations (Node.js, Python, Go)
   - Test function invocation and response
   - Validate environment variables and secrets
   - Test function updates and rollbacks
   - Verify logging and monitoring

5. **Create `test/integration/pulumi/aws/api-deployment.test.go`**
   - Deploy API Gateway with multiple routes
   - Test HTTP endpoints and authentication
   - Validate CORS configuration
   - Test custom domains and certificates
   - Verify request/response transformations

6. **Create `test/integration/pulumi/aws/full-stack.test.go`**
   - Deploy complete SST application (API + Frontend + Database)
   - Test end-to-end functionality
   - Validate service-to-service communication
   - Test database connections and queries
   - Verify CDN and static asset delivery

**Integration test infrastructure:**
- Dedicated AWS test accounts with appropriate permissions
- Automated cleanup of test resources
- Parallel test execution with resource isolation
- Cost monitoring and budget alerts for test environments

### 5.4 Example Project Testing
**Purpose**: Validate all SST example projects work correctly

**Step-by-step:**
1. **Create automated tests for each example in `examples/`:**
   - `test/integration/examples/aws-api.test.go`
   - `test/integration/examples/aws-nextjs.test.go`
   - `test/integration/examples/aws-astro.test.go`
   - `test/integration/examples/cloudflare-worker.test.go`
   - etc.

2. **Each example test should:**
   - Deploy the example project to a test environment
   - Validate all resources are created correctly
   - Test the deployed application functionality
   - Verify cleanup works properly
   - Check for any security or compliance issues

### 5.5 Integration Tests (Real Infrastructure)
**Step-by-step:**
1. Create `test/integration/` directory structure:
   ```
   test/
   ├── integration/
   │   ├── aws/
   │   │   ├── basic_deployment_test.go
   │   │   ├── function_test.go
   │   │   ├── bucket_test.go
   │   │   ├── api_gateway_test.go
   │   │   └── cleanup_test.go
   │   ├── fixtures/
   │   │   ├── simple-api/
   │   │   ├── bucket-upload/
   │   │   └── multi-service/
   │   └── helpers/
   │       ├── aws_setup.go
   │       ├── test_project.go
   │       └── cleanup.go
   ```

2. **Create `test/integration/aws/basic_deployment_test.go`**
   - Test deploying a simple SST project to real AWS
   - Verify resources are created correctly
   - Test resource cleanup
   - Use dedicated test AWS account/region

3. **Create `test/integration/aws/function_test.go`**
   - Deploy Lambda function with SST
   - Test function invocation
   - Test environment variables and linking
   - Test function updates and rollbacks

4. **Create `test/integration/aws/bucket_test.go`**
   - Deploy S3 bucket with SST
   - Test file upload/download
   - Test bucket policies and permissions
   - Test bucket deletion

5. **Create `test/integration/aws/api_gateway_test.go`**
   - Deploy API Gateway with SST
   - Test HTTP endpoints
   - Test authentication/authorization
   - Test custom domains (if configured)

### 5.6 Pulumi Testing Commands and Infrastructure

**Required setup for Pulumi testing:**

1. **Environment variables for integration tests:**
   ```bash
   # AWS Integration Tests
   export SST_TEST_AWS_ACCOUNT_ID="123456789012"
   export SST_TEST_AWS_REGION="us-east-1"
   export SST_TEST_AWS_ACCESS_KEY_ID="..."
   export SST_TEST_AWS_SECRET_ACCESS_KEY="..."
   export SST_TEST_STAGE="pulumi-integration-test"
   
   # Cloudflare Integration Tests
   export SST_TEST_CLOUDFLARE_API_TOKEN="..."
   export SST_TEST_CLOUDFLARE_ACCOUNT_ID="..."
   export SST_TEST_CLOUDFLARE_ZONE_ID="..."
   ```

2. **Test configuration:**
   ```go
   // test/integration/pulumi/config.go
   type PulumiIntegrationTestConfig struct {
       AWSAccountID     string
       AWSRegion        string
       CloudflareToken  string
       TestStage        string
       CleanupAfter     bool
       Timeout          time.Duration
       PolicyPackPath   string
   }
   ```

3. **Test helpers:**
   ```go
   // test/integration/pulumi/helpers/pulumi-integration.go
   func SetupPulumiTestEnvironment() (*pulumi.Context, error)
   func CreateTestSST Project(template string) (string, error)
   func DeployWithPolicies(projectPath, stage, policyPack string) error
   func ValidateDeployment(stackName string, validators []ResourceValidator) error
   func CleanupPulumiStack(projectPath, stage string) error
   ```

**Commands to add to CONTEXT.md:**
```bash
# Pulumi Unit Tests (TypeScript)
cd platform && bun run test test/components/pulumi/

# Pulumi Property Tests (Policy validation)
cd platform && pulumi policy test test/policies/

# Pulumi Integration Tests (Real infrastructure)
go test -v ./test/integration/pulumi/...

# Run specific Pulumi integration test
go test -v ./test/integration/pulumi/aws/function-deployment.test.go

# Run Pulumi tests with cleanup
SST_TEST_CLEANUP=true go test -v ./test/integration/pulumi/...

# Run example project tests
go test -v ./test/integration/examples/...

# Validate policies against existing stack
pulumi policy validate --policy-pack platform/test/policies/aws/ my-stack

# Run property tests during deployment
sst deploy --policy-pack platform/test/policies/
```

### 5.7 Pulumi Testing Success Criteria

- ✅ **Unit Test Coverage**: >90% of SST Pulumi components tested with mocks
- ✅ **Property Test Coverage**: All critical security and compliance rules enforced
- ✅ **Integration Test Coverage**: All major SST component types tested with real infrastructure
- ✅ **Example Validation**: All example projects deploy and function correctly
- ✅ **Policy Enforcement**: Property tests run automatically during deployment
- ✅ **Performance**: Unit tests run in <30 seconds, integration tests in <15 minutes
- ✅ **Reliability**: Tests are stable and not flaky
- ✅ **Cost Control**: Integration tests cost <$20/day to run
- ✅ **Documentation**: Clear testing guidelines and examples for contributors

### 5.8 Pulumi Testing Implementation Priority

**Week 1-2: Pulumi Unit Tests (HIGH PRIORITY)**
- Set up Pulumi mocking infrastructure
- Create comprehensive tests for core AWS components (Function, Bucket, API Gateway)
- Extend existing bucket.test.ts with full coverage
- Add tests for Cloudflare components

**Week 3: Pulumi Property Tests (HIGH PRIORITY)**
- Create security and compliance policy packs
- Implement naming convention and resource limit policies
- Integrate policy validation into SST CLI commands

**Week 4-5: Pulumi Integration Tests (CRITICAL PRIORITY)**
- Set up integration test infrastructure with real AWS/Cloudflare accounts
- Create basic deployment tests for core components
- Add runtime validation and functionality testing

**Week 6: Example Project Validation (MEDIUM PRIORITY)**
- Automate testing of all example projects
- Create CI/CD pipeline for example validation
- Add performance and cost monitoring

## Phase 6: End-to-End Deployment Workflows
**Step-by-step:**
1. **Create `test/integration/e2e_deploy_test.go`**
   - Test complete project lifecycle: init → deploy → test → remove
   - Test with multiple stages (dev, staging)
   - Test deployment rollbacks
   - Test state management across deployments

2. **Create `test/integration/e2e_multi_service_test.go`**
   - Deploy complex multi-service application
   - Test service-to-service communication
   - Test shared resources (databases, queues)
   - Test dependency ordering

3. **Create `test/integration/e2e_secrets_test.go`**
   - Test secret management and deployment
   - Test environment-specific secrets
   - Test secret rotation scenarios

### 5.3 Real Infrastructure Validation
**Step-by-step:**
1. **Create `test/integration/validation/`**
   - `resource_validation_test.go` - Verify deployed resources match configuration
   - `performance_test.go` - Test deployment speed and resource limits
   - `cost_validation_test.go` - Verify resource costs are within expected ranges

2. **Create smoke tests for each example project:**
   - `test/integration/examples/aws_api_test.go`
   - `test/integration/examples/aws_nextjs_test.go`
   - `test/integration/examples/aws_astro_test.go`
   - etc.

### 5.4 Integration Test Infrastructure
**Required setup:**

1. **AWS Test Environment:**
   ```bash
   # Environment variables needed
   export SST_TEST_AWS_ACCOUNT_ID="123456789012"
   export SST_TEST_AWS_REGION="us-east-1"
   export SST_TEST_AWS_ACCESS_KEY_ID="..."
   export SST_TEST_AWS_SECRET_ACCESS_KEY="..."
   export SST_TEST_STAGE="integration-test"
   ```

2. **Test Configuration:**
   ```go
   // test/integration/config.go
   type IntegrationTestConfig struct {
       AWSAccountID string
       AWSRegion    string
       TestStage    string
       CleanupAfter bool
       Timeout      time.Duration
   }
   ```

3. **Test Helpers:**
   ```go
   // test/integration/helpers/aws_setup.go
   func SetupAWSTestEnvironment() (*aws.Config, error)
   func CreateTestProject(template string) (string, error)
   func DeployProject(projectPath, stage string) error
   func CleanupProject(projectPath, stage string) error
   func WaitForDeployment(stackName string) error
   ```

### 5.5 CI/CD Integration Tests
**Step-by-step:**
1. **Create `.github/workflows/integration-tests.yml`**
   ```yaml
   name: Integration Tests
   on:
     push:
       branches: [main]
     pull_request:
       branches: [main]
   
   jobs:
     integration:
       runs-on: ubuntu-latest
       steps:
         - uses: actions/checkout@v4
         - name: Setup Go
           uses: actions/setup-go@v4
         - name: Run Integration Tests
           env:
             AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
             AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
           run: go test -v ./test/integration/...
   ```

2. **Create nightly full integration test suite**
   - Test all example projects
   - Test against multiple AWS regions
   - Generate deployment reports

### 5.6 Integration Test Execution
**Commands to add to CONTEXT.md:**
```bash
# Run all integration tests
go test -v ./test/integration/...

# Run specific integration test
go test -v ./test/integration/aws/basic_deployment_test.go

# Run integration tests with cleanup
SST_TEST_CLEANUP=true go test -v ./test/integration/...

# Run integration tests against specific region
SST_TEST_AWS_REGION=us-west-2 go test -v ./test/integration/...
```

### 5.7 Integration Test Success Criteria
- ✅ Deploy and cleanup resources successfully
- ✅ Verify deployed resources match SST configuration
- ✅ Test actual functionality (API calls, file uploads, etc.)
- ✅ Handle deployment failures gracefully
- ✅ Clean up all resources after tests
- ✅ Run in under 15 minutes for basic suite
- ✅ Generate deployment cost reports

## Testing Infrastructure Setup

### Required Dependencies
Add to `go.mod`:
```go
// Testing dependencies
github.com/stretchr/testify v1.8.4
github.com/golang/mock v1.6.0
github.com/spf13/afero v1.9.5  // Mock filesystem
github.com/aws/aws-sdk-go-v2/aws/testing v1.0.0
```

### Test Utilities Package
Create `pkg/testutil/` with:
1. `mock_fs.go` - Filesystem mocking helpers
2. `mock_cmd.go` - Command execution mocking
3. `mock_aws.go` - AWS service mocking
4. `fixtures.go` - Test data and fixtures
5. `helpers.go` - Common test helpers

### CI/CD Integration
1. Update GitHub Actions to run tests
2. Add test coverage reporting
3. Add integration test environment
4. Set up test databases/resources

## Execution Timeline

**Week 1-2**: Phase 5 (Pulumi Testing) - **CRITICAL PRIORITY**
- Pulumi Unit Tests for core components
- Pulumi Property Tests for security/compliance
- Set up Pulumi integration test infrastructure

**Week 3-4**: Phase 5 (Pulumi Integration) - **CRITICAL PRIORITY**  
- Real infrastructure integration tests
- Example project validation
- CI/CD pipeline integration

**Week 5-6**: Phase 6 (End-to-End Workflows) - **HIGH PRIORITY**
- Complete deployment lifecycle testing
- Multi-service application testing
- Performance and reliability testing

**Week 7-8**: Phase 3-4 (Server & Utilities) - **MEDIUM PRIORITY**
- Server core and AWS resources testing
- Utilities and infrastructure testing

## Success Metrics

- **Pulumi Unit Test Coverage**: >90% of SST components tested with mocks ⚠️ **MISSING**
- **Pulumi Property Test Coverage**: All security/compliance rules enforced ⚠️ **MISSING**
- **Pulumi Integration Test Coverage**: >90% of AWS/Cloudflare resources tested with real deployments ⚠️ **MISSING**
- **Unit Test Coverage**: >80% for core packages ✅ **IN PROGRESS**
- **E2E Test Coverage**: All critical deployment workflows covered ⚠️ **MISSING**
- **Example Project Validation**: All examples deploy and function correctly ⚠️ **MISSING**
- **CI/CD**: All tests pass on every PR ⚠️ **NO CI/CD EXISTS**
- **Performance**: Integration test suite runs in <15 minutes
- **Reliability**: Tests are stable and not flaky
- **Cost Control**: Integration tests cost <$20/day to run

## Getting Started

1. **Start with**: `pkg/project/project_test.go`
2. **Use**: `go test -v ./pkg/project/` to run
3. **Pattern**: Follow existing test patterns in `pkg/id/id_test.go`
4. **Mock**: Use testify/mock for external dependencies
5. **Iterate**: Add tests incrementally, one package at a time