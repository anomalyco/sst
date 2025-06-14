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

### 2.1 Main CLI (`cmd/sst/`) - **NEXT STEP**
**Files to test**: `main.go`, `deploy.go`, `init.go`, `remove.go`, `upgrade.go`

**Step-by-step:**
1. **NEXT STEP** Create `cmd/sst/main_test.go`
   - Test command parsing
   - Test flag handling
   - Test help output

2. Create `cmd/sst/deploy_test.go`
   - Test deployment workflow
   - Test error handling
   - Mock cloud provider interactions

3. Create `cmd/sst/init_test.go`
   - Test project initialization
   - Test template selection
   - Test file generation

4. Create `cmd/sst/remove_test.go`
   - Test resource cleanup
   - Test confirmation prompts
   - Test rollback scenarios

**Test utilities needed:**
- CLI testing framework (cobra testing)
- Mock cloud providers
- Capture stdout/stderr

### 2.2 CLI Utilities (`cmd/sst/cli/`)
**Files to test**: `cli.go`, `project.go`

**Step-by-step:**
1. Create `cmd/sst/cli/cli_test.go`
   - Test CLI initialization
   - Test configuration loading
   - Test error formatting

2. Create `cmd/sst/cli/project_test.go`
   - Test project context detection
   - Test workspace handling
   - Test multi-project scenarios

## Phase 3: Server & Resources (Medium Priority)

### 3.1 Server Core (`pkg/server/`)
**Files to test**: `server.go`, `client.go`

**Step-by-step:**
1. Create `pkg/server/server_test.go`
   - Test server startup/shutdown
   - Test request handling
   - Test middleware

2. Create `pkg/server/client_test.go`
   - Test client connections
   - Test request/response cycles
   - Test error handling

### 3.2 AWS Resources (`pkg/server/resource/`)
**Files to test**: All AWS resource handlers

**Step-by-step:**
1. Create `pkg/server/resource/aws_test.go`
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

## Phase 5: Integration Tests (HIGH PRIORITY - MISSING)

⚠️ **CRITICAL GAP**: No integration tests exist that verify SST works with real AWS infrastructure.

### 5.1 AWS Infrastructure Integration Tests
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

### 5.2 End-to-End Deployment Workflows
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

**Week 1**: Phase 5 (Integration Tests) - **CRITICAL PRIORITY**
**Week 2-3**: Phase 1 (Core Business Logic)
**Week 4-5**: Phase 2 (CLI Commands)  
**Week 6-7**: Phase 3 (Server & Resources)
**Week 8**: Phase 4 (Utilities)

## Success Metrics

- **Integration Test Coverage**: >90% of AWS resources tested with real deployments ⚠️ **MISSING**
- **Unit Test Coverage**: >80% for core packages ✅ **IN PROGRESS**
- **E2E Test Coverage**: All critical deployment workflows covered ⚠️ **MISSING**
- **CI/CD**: All tests pass on every PR ⚠️ **NO CI/CD EXISTS**
- **Performance**: Integration test suite runs in <15 minutes
- **Reliability**: Tests are stable and not flaky
- **Cost Control**: Integration tests cost <$10/day to run

## Getting Started

1. **Start with**: `pkg/project/project_test.go`
2. **Use**: `go test -v ./pkg/project/` to run
3. **Pattern**: Follow existing test patterns in `pkg/id/id_test.go`
4. **Mock**: Use testify/mock for external dependencies
5. **Iterate**: Add tests incrementally, one package at a time