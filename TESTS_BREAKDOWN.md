# SST Testing Strategy Breakdown

## Current Test Status
- ✅ **4 packages** have tests (11 test files total)
- ❌ **76+ packages** missing tests
- 🎯 **Target**: Comprehensive test coverage for critical functionality

## Phase 1: Core Business Logic (High Priority)

### 1.1 Project Management (`pkg/project/`)
**Files to test**: `project.go`, `create.go`, `install.go`, `run.go`, `add.go`, `env.go`

**Step-by-step:**
1. Create `pkg/project/project_test.go`
   - Test project detection and validation
   - Test project configuration loading
   - Mock filesystem operations

2. Create `pkg/project/create_test.go`
   - Test project creation with different templates
   - Test directory structure creation
   - Test error handling for invalid inputs

3. Create `pkg/project/install_test.go`
   - Test dependency installation
   - Test package manager detection (npm, bun, etc.)
   - Mock external command execution

4. Create `pkg/project/run_test.go`
   - Test command execution
   - Test environment variable handling
   - Test process spawning and monitoring

**Test utilities needed:**
- Mock filesystem (`afero` or similar)
- Mock command execution
- Temporary directory helpers

### 1.2 Runtime Management (`pkg/runtime/`)
**Files to test**: `runtime.go`, `node/`, `python/`, `rust/`, `golang/`, `worker/`

**Step-by-step:**
1. Create `pkg/runtime/runtime_test.go`
   - Test runtime detection
   - Test runtime version parsing
   - Test runtime compatibility checks

2. Create `pkg/runtime/node/node_test.go`
   - Test Node.js version detection
   - Test package.json parsing
   - Test build process

3. Create `pkg/runtime/python/python_test.go`
   - Test Python version detection
   - Test requirements.txt parsing
   - Test virtual environment handling

4. Create `pkg/runtime/golang/golang_test.go`
   - Test Go module detection
   - Test build configuration
   - Test dependency resolution

**Test utilities needed:**
- Mock runtime binaries
- Fake version outputs
- Mock build processes

### 1.3 State Management (`pkg/state/`)
**Files to test**: `state.go`, `decrypt.go`

**Step-by-step:**
1. Create `pkg/state/state_test.go`
   - Test state persistence
   - Test state loading and saving
   - Test state validation

2. Create `pkg/state/decrypt_test.go`
   - Test encryption/decryption
   - Test key management
   - Test error handling for corrupted data

**Test utilities needed:**
- Mock encryption keys
- Temporary state files
- Mock AWS/cloud credentials

## Phase 2: CLI Commands (High Priority)

### 2.1 Main CLI (`cmd/sst/`)
**Files to test**: `main.go`, `deploy.go`, `init.go`, `remove.go`, `upgrade.go`

**Step-by-step:**
1. Create `cmd/sst/main_test.go`
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

## Phase 5: Integration Tests (Low Priority)

### 5.1 End-to-End Workflows
**Step-by-step:**
1. Create `test/integration/` directory
2. Create `deploy_workflow_test.go`
   - Test full deployment cycle
   - Test with real cloud resources (optional)
   - Test rollback scenarios

3. Create `project_lifecycle_test.go`
   - Test project creation to deployment
   - Test multiple environments
   - Test cleanup

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

**Week 1-2**: Phase 1 (Core Business Logic)
**Week 3-4**: Phase 2 (CLI Commands)  
**Week 5-6**: Phase 3 (Server & Resources)
**Week 7**: Phase 4 (Utilities)
**Week 8**: Phase 5 (Integration Tests)

## Success Metrics

- **Unit Test Coverage**: >80% for core packages
- **Integration Test Coverage**: Key workflows covered
- **CI/CD**: All tests pass on every PR
- **Performance**: Test suite runs in <5 minutes
- **Reliability**: Tests are stable and not flaky

## Getting Started

1. **Start with**: `pkg/project/project_test.go`
2. **Use**: `go test -v ./pkg/project/` to run
3. **Pattern**: Follow existing test patterns in `pkg/id/id_test.go`
4. **Mock**: Use testify/mock for external dependencies
5. **Iterate**: Add tests incrementally, one package at a time