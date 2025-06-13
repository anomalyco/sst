# AWS Plugin Migration Testing Strategy

## Overview

This document outlines the comprehensive testing strategy for verifying that the AWS plugin components have been correctly migrated and are functioning as expected. The tests focus on component versioning, migration paths, naming conventions, and backward compatibility.

## Current State Analysis

### Existing Test Coverage
- ✅ Basic component naming tests (`component.test.ts`)
- ✅ AWS resource naming limits validation
- ✅ Physical name transformations
- ❌ **Missing**: Component migration testing
- ❌ **Missing**: Version compatibility testing
- ❌ **Missing**: Integration testing between components

### Migration Patterns Identified

#### 1. **Version Registration Pattern**
Components use `registerVersion()` to handle breaking changes:
```typescript
self.registerVersion({
  new: _version,
  old: sst.version[name],
  message: "Migration instructions...",
  forceUpgrade: args.forceUpgrade,
});
```

#### 2. **V1 Legacy Support Pattern**
```typescript
export class ComponentName extends AWSComponent {
  public static v1 = ComponentNameV1;
  // ...
}
```

#### 3. **Components with Active Migration**
- **Auth** (v2) - OpenAuth migration with `forceUpgrade`
- **Vpc** (v2) - Network architecture changes
- **Redis** (v2) - Configuration changes
- **Postgres** (v2) - Database setup changes
- **Service** (v2) - Container service changes
- **Cluster** (v2) - ECS cluster changes

## Testing Strategy Breakdown

### Phase 1: Component Migration Tests
**Objective**: Verify migration logic and version handling

#### Test Categories:
1. **Version Detection Tests**
   - Test version registration mechanism
   - Verify migration warning messages
   - Test `forceUpgrade` parameter handling

2. **Backward Compatibility Tests**
   - Test v1 component accessibility
   - Verify legacy component functionality
   - Test mixed version scenarios

3. **Breaking Change Detection Tests**
   - Test components that should trigger migration warnings
   - Verify error messages match expected format
   - Test version downgrade prevention

### Phase 2: Component Integration Tests
**Objective**: Verify components work correctly together

#### Test Categories:
1. **Cross-Component Dependencies**
   - Function + Bucket integration
   - VPC + Service integration
   - Auth + Router integration

2. **Resource Naming Consistency**
   - Test naming across component versions
   - Verify physical name generation
   - Test naming collision prevention

### Phase 3: Infrastructure Component Tests
**Objective**: Test core AWS components individually

#### Priority Components:
1. **Function** - Lambda function creation and configuration
2. **Bucket** - S3 bucket setup and policies
3. **Dynamo** - DynamoDB table configuration
4. **Queue** - SQS queue setup
5. **Vpc** - Network infrastructure
6. **Auth** - Authentication system

### Phase 4: Migration Scenario Tests
**Objective**: Test real-world migration scenarios

#### Scenarios:
1. **Fresh Installation** - New components with latest versions
2. **Incremental Migration** - Upgrading one component at a time
3. **Force Upgrade** - Using `forceUpgrade` parameter
4. **Rollback Prevention** - Preventing version downgrades

## Step-by-Step Implementation Plan

### Step 1: Setup Test Infrastructure
**Duration**: 30 minutes

1. **Create test directory structure**
   ```bash
   cd /Users/alexeypolitov/LocalProjects/GitHub/sst-joe/plugin/aws
   mkdir -p src/test/{migration,integration,components,scenarios}
   ```

2. **Setup test utilities**
   - Create mock Pulumi environment
   - Setup SST plugin mocks
   - Create test helpers for component creation

3. **Configure test runner**
   - Verify Bun test configuration
   - Setup test coverage reporting
   - Configure test environment variables

### Step 2: Create Migration Tests
**Duration**: 2 hours

1. **Create `src/test/migration/version-handling.test.ts`**
   - Test `registerVersion()` functionality
   - Test version comparison logic
   - Test migration message generation

2. **Create `src/test/migration/force-upgrade.test.ts`**
   - Test `forceUpgrade` parameter handling
   - Test upgrade validation
   - Test invalid forceUpgrade values

3. **Create `src/test/migration/backward-compatibility.test.ts`**
   - Test v1 component access via `.v1` property
   - Test legacy component instantiation
   - Test mixed version scenarios

### Step 3: Create Component-Specific Tests
**Duration**: 3 hours

1. **Create `src/test/components/auth.test.ts`**
   ```typescript
   // Test Auth component migration from v1 to v2
   // Test OpenAuth integration
   // Test forceUpgrade mechanism
   ```

2. **Create `src/test/components/vpc.test.ts`**
   ```typescript
   // Test VPC component migration
   // Test network configuration changes
   // Test AZ handling
   ```

3. **Create `src/test/components/function.test.ts`**
   ```typescript
   // Test Function component creation
   // Test environment variable handling
   // Test linking mechanism
   ```

4. **Create `src/test/components/bucket.test.ts`**
   ```typescript
   // Test Bucket component creation
   // Test CORS configuration
   // Test notification setup
   ```

5. **Create tests for other critical components**
   - `dynamo.test.ts`
   - `queue.test.ts`
   - `redis.test.ts`
   - `postgres.test.ts`

### Step 4: Create Integration Tests
**Duration**: 2 hours

1. **Create `src/test/integration/component-linking.test.ts`**
   - Test Function + Bucket linking
   - Test Auth + Router integration
   - Test VPC + Service integration

2. **Create `src/test/integration/naming-consistency.test.ts`**
   - Test naming across component versions
   - Test physical name generation consistency
   - Test naming collision prevention

3. **Create `src/test/integration/resource-dependencies.test.ts`**
   - Test component dependency resolution
   - Test circular dependency prevention
   - Test dependency ordering

### Step 5: Create Migration Scenario Tests
**Duration**: 1.5 hours

1. **Create `src/test/scenarios/fresh-installation.test.ts`**
   - Test new project setup with latest components
   - Test component initialization
   - Test default configuration

2. **Create `src/test/scenarios/incremental-migration.test.ts`**
   - Test upgrading one component at a time
   - Test migration path validation
   - Test state preservation

3. **Create `src/test/scenarios/force-upgrade.test.ts`**
   - Test force upgrade scenarios
   - Test upgrade validation
   - Test post-upgrade state

### Step 6: Create Test Utilities and Mocks
**Duration**: 1 hour

1. **Create `src/test/utils/mock-pulumi.ts`**
   ```typescript
   // Enhanced Pulumi mocking utilities
   // Resource state simulation
   // Output value handling
   ```

2. **Create `src/test/utils/mock-sst.ts`**
   ```typescript
   // SST plugin mocking
   // Component creation helpers
   // Version simulation
   ```

3. **Create `src/test/utils/test-helpers.ts`**
   ```typescript
   // Common test utilities
   // Assertion helpers
   // Test data generators
   ```

### Step 7: Run and Validate Tests
**Duration**: 30 minutes

1. **Execute test suites**
   ```bash
   cd /Users/alexeypolitov/LocalProjects/GitHub/sst-joe/plugin/aws
   bun test src/test/migration/
   bun test src/test/components/
   bun test src/test/integration/
   bun test src/test/scenarios/
   ```

2. **Generate coverage report**
   ```bash
   bun test --coverage
   ```

3. **Validate test results**
   - Ensure all migration paths are tested
   - Verify component compatibility
   - Check for edge cases

### Step 8: Documentation and Cleanup
**Duration**: 30 minutes

1. **Update test documentation**
   - Document test patterns
   - Add troubleshooting guide
   - Update README with test instructions

2. **Clean up test code**
   - Remove debugging code
   - Optimize test performance
   - Add proper error handling

## Test File Structure

```
plugin/aws/src/test/
├── migration/
│   ├── version-handling.test.ts
│   ├── force-upgrade.test.ts
│   └── backward-compatibility.test.ts
├── components/
│   ├── auth.test.ts
│   ├── vpc.test.ts
│   ├── function.test.ts
│   ├── bucket.test.ts
│   ├── dynamo.test.ts
│   ├── queue.test.ts
│   ├── redis.test.ts
│   └── postgres.test.ts
├── integration/
│   ├── component-linking.test.ts
│   ├── naming-consistency.test.ts
│   └── resource-dependencies.test.ts
├── scenarios/
│   ├── fresh-installation.test.ts
│   ├── incremental-migration.test.ts
│   └── force-upgrade.test.ts
└── utils/
    ├── mock-pulumi.ts
    ├── mock-sst.ts
    └── test-helpers.ts
```

## Success Criteria

### ✅ Migration Tests Pass
- [ ] All version handling tests pass
- [ ] Force upgrade mechanism works correctly
- [ ] Backward compatibility is maintained
- [ ] Migration warnings are displayed correctly

### ✅ Component Tests Pass
- [ ] All critical components can be instantiated
- [ ] Component configuration is validated
- [ ] Component linking works correctly
- [ ] Resource naming follows conventions

### ✅ Integration Tests Pass
- [ ] Components work together correctly
- [ ] Dependencies are resolved properly
- [ ] No naming conflicts occur
- [ ] Resource creation order is correct

### ✅ Scenario Tests Pass
- [ ] Fresh installations work
- [ ] Incremental migrations work
- [ ] Force upgrades work
- [ ] Error scenarios are handled

## Risk Mitigation

### High-Risk Areas
1. **Auth Component Migration** - Complex OpenAuth integration
2. **VPC Component Changes** - Network infrastructure changes
3. **Cross-Component Dependencies** - Breaking changes in dependencies

### Mitigation Strategies
1. **Comprehensive Mocking** - Mock all external dependencies
2. **Isolated Testing** - Test components in isolation first
3. **Integration Testing** - Test component interactions
4. **Scenario Testing** - Test real-world usage patterns

## Maintenance

### Regular Tasks
1. **Update tests when components change**
2. **Add tests for new components**
3. **Review and update migration paths**
4. **Monitor test performance**

### Quarterly Reviews
1. **Review test coverage**
2. **Update migration documentation**
3. **Validate test effectiveness**
4. **Plan test improvements**

## Commands Reference

```bash
# Run all tests
bun test

# Run specific test category
bun test src/test/migration/
bun test src/test/components/
bun test src/test/integration/
bun test src/test/scenarios/

# Run single test file
bun test src/test/components/auth.test.ts

# Run tests with coverage
bun test --coverage

# Run tests in watch mode
bun test --watch

# Run tests with verbose output
bun test --verbose
```

## Expected Timeline

- **Total Duration**: ~8 hours
- **Phase 1**: 2.5 hours (Setup + Migration Tests)
- **Phase 2**: 3 hours (Component Tests)
- **Phase 3**: 2 hours (Integration Tests)
- **Phase 4**: 0.5 hours (Documentation)

## Next Steps

1. **Start with Step 1**: Setup test infrastructure
2. **Focus on high-risk components first**: Auth, VPC, Function
3. **Build incrementally**: Test each component before moving to integration
4. **Validate continuously**: Run tests after each implementation step
5. **Document findings**: Update this document with any discoveries