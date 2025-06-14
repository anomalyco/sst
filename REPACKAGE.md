# SST Repackage Initiative

## Overview
The repackage branch is restructuring SST's architecture by migrating from a monolithic `platform/` directory to a modular plugin-based system. This enables better separation of concerns, independent versioning, and easier maintenance of cloud provider-specific components.

## What's Being Done

### 1. Plugin System Architecture
- **Created plugin structure**: `plugin/base/`, `plugin/aws/`, `plugin/cloudflare/`, `plugin/example/`
- **Base plugin**: Core SST functionality (components, linking, runtime)
- **Provider plugins**: Cloud-specific implementations (AWS, Cloudflare)
- **Workspace configuration**: Updated to include `plugin/*` packages with shared catalog dependencies

### 2. Component Migration
- **Moved components**: Migrated all AWS components from `platform/src/components/aws/` to `plugin/aws/src/`
- **Preserved functionality**: All existing components (Function, Bucket, VPC, etc.) maintained
- **Updated imports**: Components now use plugin-based imports (`sst-plugin-aws` vs `@sst/platform`)

### 3. Build System Updates
- **Plugin builds**: Each plugin has independent build scripts and TypeScript configs
- **Dependency management**: Plugins declare their own dependencies (Pulumi providers, AWS SDK, etc.)
- **Export structure**: Plugins export components via standard npm package exports

## Completed Steps ✅
- [x] Created plugin directory structure
- [x] Migrated AWS components to `plugin/aws/`
- [x] Set up base plugin with core functionality
- [x] Updated workspace configuration with catalog dependencies
- [x] Implemented plugin build system
- [x] Updated examples to use new plugin imports
- [x] **Added comprehensive test suite**: ✅ **COMPLETED** - Created tests using Bun for all plugins
  - ✅ Cloudflare plugin: 3 test files (component.test.ts, bucket.test.ts, binding.test.ts) - 21 tests passing
  - ✅ AWS plugin: 1 test file (component.test.ts) - 6 tests passing  
  - ✅ Base plugin: 1 test file (component.test.ts) - 6 tests passing
  - ✅ All plugins configured with `bun test` script
  - ✅ Tests follow pattern of placing .test.ts files alongside source files

## Plugin Architecture Analysis & Implementation Plan

### 📊 Current Plugin Status Overview

| Plugin | Components | Tests | Status | Priority |
|--------|------------|-------|--------|----------|
| **Base** | 25 files | 1 test file (6 tests) | ✅ Complete | Core Foundation |
| **AWS** | 131 files | 17 test files (271 tests) | ✅ Complete + Migration Testing | Production Ready |
| **Cloudflare** | 24 files | 3 test files (21 tests) | 🚧 Needs Migration | High Priority |
| **Example** | 1 file | 0 tests | ✅ Template | Reference |

### 🎯 Plugin Implementation Breakdown

#### 1. **Base Plugin** ✅ **COMPLETE**
**Location**: `plugin/base/src/`
**Purpose**: Core SST functionality and shared utilities

**Architecture**:
- **Core Components**: `component.ts`, `app.ts`, `config.ts`, `dev.ts`
- **Runtime System**: `runtime/` (link.ts, run.ts, shim.ts)
- **Internal Utilities**: `internal/` (component.ts, dns.ts, transform.ts, util.ts, etc.)
- **Public APIs**: `linkable.ts`, `naming.ts`, `resource.ts`, `secret.ts`
- **Error Handling**: `error.ts` with comprehensive error types
- **Experimental Features**: `experimental/` (dev-command.ts)

**Key Features**:
- ✅ Component base class with transformation system
- ✅ Linkable resource pattern for cross-component communication
- ✅ Runtime linking and execution framework
- ✅ Comprehensive naming and transformation utilities
- ✅ Error handling and validation system
- ✅ Development and configuration management

**Testing Status**: 6 tests covering component functionality

#### 2. **AWS Plugin** ✅ **PRODUCTION READY**
**Location**: `plugin/aws/src/`
**Purpose**: Complete AWS cloud provider implementation

**Architecture**:
- **Base Component**: `component.ts` - AWSComponent with comprehensive AWS naming rules
- **Core Services**: 
  - **Compute**: `function.ts`, `cluster.ts`, `service.ts`, `fargate.ts`, `task.ts`
  - **Storage**: `bucket.ts`, `efs.ts`, `vector.ts`
  - **Database**: `dynamo.ts`, `aurora.ts`, `postgres.ts`, `mysql.ts`, `redis.ts`
  - **Networking**: `vpc.ts`, `cdn.ts`, `dns.ts`
  - **Auth & Security**: `auth.ts`, `cognito-*.ts`, `iam-edit.ts`, `permission.ts`
  - **API & Messaging**: `apigateway*.ts`, `bus.ts`, `queue.ts`, `sns-topic.ts`, `email.ts`
  - **Monitoring**: `logging.ts`, `cron.ts`, `realtime.ts`
- **Framework Support**: `nextjs.ts`, `astro.ts`, `react.ts`, `remix.ts`, `nuxt.ts`, `svelte-kit.ts`, etc.
- **Utilities**: `util/` (30+ utility files for AWS-specific operations)
- **Providers**: `providers/` (custom Pulumi providers for AWS operations)
- **Step Functions**: `step-functions/` (workflow orchestration components)
- **Base Classes**: `base/` (shared site and SSR implementations)

**Migration Features**:
- ✅ **Version Registration System**: `registerVersion()` with migration warnings
- ✅ **Backward Compatibility**: `.v1` static properties for legacy components
- ✅ **Force Upgrade Mechanism**: `forceUpgrade` parameter for breaking changes
- ✅ **Component Versioning**: Auth v2, VPC v2, Redis v2, Postgres v2, Service v2, Cluster v2

**Testing Status**: 271 comprehensive tests covering:
- ✅ Migration logic (33 tests)
- ✅ Component functionality (148 tests)
- ✅ Integration scenarios (37 tests)
- ✅ Real-world usage patterns (47 tests)
- ✅ Error handling and edge cases

#### 3. **Cloudflare Plugin** 🚧 **NEEDS COMPREHENSIVE MIGRATION**
**Location**: `plugin/cloudflare/src/`
**Purpose**: Cloudflare cloud provider implementation

**Current Architecture**:
- **Base Component**: `component.ts` - Basic Component class (incomplete)
- **Core Services**:
  - **Compute**: `worker.ts`, `cron.ts`
  - **Storage**: `bucket.ts`, `kv.ts`, `d1.ts`
  - **Networking**: `dns.ts`
  - **Auth**: `auth.ts`
  - **Framework Support**: `remix.ts`, `ssr-site.ts`, `static-site.ts`
- **Utilities**: `helpers/` (fetch.ts, worker-builder.ts)
- **Providers**: `providers/` (dns-record.ts, kv-data.ts, worker-url.ts, zone-lookup.ts)
- **Configuration**: `account-id.ts`, `binding.ts`, `queue.ts`

**Critical Issues Identified**:
- ❌ **Incomplete Base Component**: Missing version registration, component tracking
- ❌ **Import Inconsistencies**: Not using `sst-plugin` namespace consistently
- ❌ **Missing Migration System**: No version handling or backward compatibility
- ❌ **Limited Testing**: Only 21 basic tests vs AWS's 271 comprehensive tests
- ❌ **Incomplete Utilities**: Missing Cloudflare-specific utility functions

**Required Migration Work**:
1. **Component Base Class Enhancement** (High Priority)
2. **Import Path Standardization** (High Priority)
3. **Migration System Implementation** (Medium Priority)
4. **Comprehensive Testing Suite** (Medium Priority)
5. **Utility Function Expansion** (Low Priority)

#### 4. **Example Plugin** ✅ **TEMPLATE COMPLETE**
**Location**: `plugin/example/src/`
**Purpose**: Template and reference implementation

**Architecture**:
- **Simple Structure**: Single `index.ts` file
- **Template Pattern**: Demonstrates basic plugin structure
- **Documentation**: Serves as starting point for new plugins

## Remaining Steps 🚧

### 🔥 **Critical Priority - Cloudflare Plugin Migration**

#### **Phase 1: Foundation Fixes** ✅ **COMPLETED**
- [x] **Fix Component Base Class**: Update `component.ts` to match AWS plugin pattern ✅ **COMPLETED**
  - ✅ Add `registerVersion()` method
  - ✅ Add `componentType` and `componentName` properties  
  - ✅ Implement comprehensive Cloudflare naming rules
  - ✅ Add proper transformation system
  - ✅ Created comprehensive tests (8 tests passing)
  - ✅ Maintain backward compatibility with `Component` export
- [x] **Standardize Import Paths**: Update all files to use `sst-plugin` namespace ✅ **COMPLETED**
  - ✅ **worker.ts** - Fixed imports and global variable usage
  - ✅ **bucket.ts** - Fixed imports, updated to CloudflareComponent, tests passing
  - ✅ **kv.ts** - Fixed imports, updated to CloudflareComponent, added tests (5 tests)
  - ✅ **d1.ts** - Fixed imports, updated to CloudflareComponent, added tests (8 tests)
  - ✅ **auth.ts** - Fixed imports, updated to CloudflareComponent, added tests (7 tests)
  - ✅ **cron.ts** - Fixed imports, updated to CloudflareComponent, added tests (8 tests), fixed Pulumi type bug
  - ✅ **helpers/worker-builder.ts** - Fixed imports to use sst-plugin, added tests (8 tests)
  - ✅ **dns.ts** - Fixed imports to use sst-plugin namespace, added tests (9 tests)
  - ✅ **queue.ts** - Fixed imports, updated to CloudflareComponent, added tests (7 tests)
  - ✅ **binding.ts** - Already has correct imports, tests passing (6 tests)
  - ✅ **account-id.ts** - Already has correct structure, added tests (6 tests)
  - ✅ **providers/kv-data.ts** - Already has correct imports, added tests (8 tests)
  - ✅ **providers/worker-url.ts** - Already has correct imports, added tests (9 tests)
  - ✅ **providers/zone-lookup.ts** - Already has correct imports, added tests (9 tests)
  - ✅ **remix.ts** - Fixed imports, updated to CloudflareComponent, created base utilities, added tests (8 tests)
  - ✅ **ssr-site.ts** - Fixed imports, updated to use local base files and sst-plugin types
  - ✅ **static-site.ts** - Fixed imports, updated to CloudflareComponent, created base-static-site, added tests (8 tests)
  - ✅ **providers/dns-record.ts** - Fixed imports, simplified dynamic provider, added tests (5 tests)
  - ✅ **helpers/fetch.ts** - Already has correct structure, added comprehensive tests (6 tests)

#### **Phase 2: Migration System** ✅ **COMPLETED**
- [x] **Implement Version Management**: Add migration capabilities ✅ **COMPLETED**
  - ✅ Version registration for breaking changes
  - ✅ Backward compatibility support
  - ✅ Force upgrade mechanism
- [x] **Component Enhancement**: Update all components ✅ **COMPLETED**
  - ✅ Proper base class inheritance
  - ✅ Component registration calls
  - ✅ Error handling improvements

#### **Phase 3: Comprehensive Testing** (4-5 hours)
- [ ] **Create Migration Tests**: Following AWS plugin pattern
  - Version handling tests
  - Force upgrade tests
  - Backward compatibility tests
- [ ] **Component-Specific Tests**: Test each Cloudflare component
  - Worker, Bucket, KV, D1, DNS components
  - Configuration validation
  - Integration scenarios
- [ ] **Integration Tests**: Cross-component scenarios
  - Component linking
  - Naming consistency
  - Resource dependencies

#### **Phase 4: Advanced Features** (2-3 hours)
- [ ] **Utility Functions**: Add Cloudflare-specific utilities
- [ ] **Provider Enhancements**: Improve custom providers
- [ ] **Documentation**: Update component documentation

### 🚀 **High Priority - Platform Migration Completion**

#### **Vercel Plugin Creation** (3-4 hours)
- [ ] **Create Vercel Plugin Structure**: `plugin/vercel/`
- [ ] **Migrate Components**: Move from `platform/src/components/vercel/`
- [ ] **Implement Base Component**: VercelComponent class
- [ ] **Add Testing Suite**: Comprehensive tests following AWS pattern
- [ ] **Clean Up Platform**: Remove `platform/src/components/vercel/`

#### **CLI Integration Updates** (4-6 hours)
- [ ] **Modify Go CLI**: Update to load plugins instead of platform components
- [ ] **Plugin Discovery**: Implement automatic plugin loading
- [ ] **Import Path Updates**: Fix all examples and internal references

### 🔧 **Medium Priority - System Enhancements**

#### **Plugin Versioning System** (2-3 hours)
- [ ] **Independent Versioning**: Each plugin manages its own version
- [ ] **Cross-Plugin Compatibility**: Version compatibility matrix
- [ ] **Migration Tooling**: Tools to help users migrate configs

#### **Documentation & Testing** (3-4 hours)
- [ ] **Documentation Updates**: Reflect new plugin architecture
- [ ] **Testing Infrastructure**: CI/CD for individual plugins
- [ ] **Performance Optimization**: Plugin loading and bundling

### 🌟 **Low Priority - Future Enhancements**

#### **Plugin Ecosystem** (Long-term)
- [ ] **Plugin Marketplace**: System for third-party plugins
- [ ] **Plugin Templates**: Standardized templates for new providers
- [ ] **Community Tools**: Plugin development and testing tools

---

## 🔧 Detailed Implementation Plans

### **Cloudflare Plugin Migration Implementation**

#### **Step 1: Component Base Class Enhancement**
**File**: `plugin/cloudflare/src/component.ts`
**Duration**: 1 hour

**Current Issues**:
```typescript
// Current incomplete implementation
export class Component extends BaseComponent {
  // Missing: componentType, componentName properties
  // Missing: registerVersion method
  // Missing: comprehensive naming rules
}
```

**Required Changes**:
```typescript
// Target implementation (based on AWS plugin pattern)
export class CloudflareComponent extends BaseComponent {
  private componentType: string;
  private componentName: string;

  constructor(type: string, name: string, args?: Record<string, sst.Input<any>>, opts?: sst.ComponentOptions) {
    super(type, name, args, {
      ...opts,
      transformations: [
        // Cloudflare-specific naming and transformation rules
        (args) => {
          // Implement Cloudflare resource naming conventions
          // Handle Cloudflare-specific resource types
          // Apply proper prefixing and validation
        },
        ...(opts?.transformations || []),
      ],
    });
    
    this.componentType = type;
    this.componentName = name;
  }

  protected registerVersion(args: {
    new: string;
    old?: string;
    message?: string;
    forceUpgrade?: boolean;
  }) {
    // Implement version registration system
  }
}
```

#### **Step 2: Import Path Standardization**
**Files**: All `.ts` files in `plugin/cloudflare/src/`
**Duration**: 1-2 hours

**Pattern Changes**:
```typescript
// Before (platform-style imports)
import { Input } from "../input";
import { Component } from "./component";
import { Transform } from "../component";

// After (plugin-style imports)
import * as sst from "sst-plugin";
import { CloudflareComponent } from "./component";
import { Transform } from "sst-plugin/internal/transform";
```

**Files to Update**:
- `worker.ts`, `bucket.ts`, `kv.ts`, `d1.ts`, `dns.ts`
- `auth.ts`, `cron.ts`, `queue.ts`
- `remix.ts`, `ssr-site.ts`, `static-site.ts`
- All provider files in `providers/`
- All helper files in `helpers/`

#### **Step 3: Migration System Implementation**
**Duration**: 2-3 hours

**Add Version Management**:
```typescript
// Example for Worker component
export class Worker extends CloudflareComponent {
  public static v1 = WorkerV1; // Backward compatibility

  constructor(name: string, args: WorkerArgs, opts?: sst.ComponentOptions) {
    super("cloudflare:Worker", name, args, opts);
    
    // Register version for migration tracking
    this.registerVersion({
      new: "2.0.0",
      old: sst.version["Worker"],
      message: "Worker v2 introduces new configuration options...",
      forceUpgrade: args.forceUpgrade,
    });
  }
}
```

#### **Step 4: Comprehensive Testing Suite**
**Duration**: 4-5 hours

**Test Structure** (following AWS plugin pattern):
```
plugin/cloudflare/test/
├── migration/
│   ├── version-handling.test.ts
│   ├── force-upgrade.test.ts
│   └── backward-compatibility.test.ts
├── components/
│   ├── worker.test.ts
│   ├── bucket.test.ts
│   ├── kv.test.ts
│   ├── d1.test.ts
│   └── dns.test.ts
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

**Target Test Coverage**: 150+ tests (similar to AWS plugin scope)

### **Vercel Plugin Creation Implementation**

#### **Step 1: Plugin Structure Setup**
**Duration**: 1 hour

**Create Directory Structure**:
```
plugin/vercel/
├── package.json
├── tsconfig.json
├── script/
│   └── build.ts
├── src/
│   ├── component.ts
│   ├── index.ts
│   └── [component files]
└── test/
    └── [test files]
```

#### **Step 2: Component Migration**
**Duration**: 2-3 hours

**Migrate from**: `platform/src/components/vercel/`
**Apply Same Patterns**: Follow AWS/Cloudflare plugin architecture

#### **Step 3: Base Component Implementation**
**Duration**: 1 hour

```typescript
export class VercelComponent extends BaseComponent {
  private componentType: string;
  private componentName: string;

  constructor(type: string, name: string, args?: Record<string, sst.Input<any>>, opts?: sst.ComponentOptions) {
    super(type, name, args, {
      ...opts,
      transformations: [
        // Vercel-specific transformations
      ],
    });
  }
}
```

### **CLI Integration Updates Implementation**

#### **Step 1: Go CLI Plugin Loading**
**Files**: `cmd/sst/main.go`, `cmd/sst/mosaic/`
**Duration**: 3-4 hours

**Changes Required**:
- Update component discovery to load from plugins
- Implement plugin resolution system
- Update import paths in Go code

#### **Step 2: Plugin Discovery System**
**Duration**: 2-3 hours

**Implementation**:
- Automatic plugin detection in `node_modules`
- Plugin registration and loading
- Version compatibility checking

---

## 📈 Success Metrics & Validation

### **Cloudflare Plugin Success Criteria**
- [ ] All 24 components successfully migrated
- [ ] 150+ comprehensive tests passing
- [ ] Migration system functional (version registration, force upgrade)
- [ ] Backward compatibility maintained
- [ ] Import paths standardized
- [ ] Build system working correctly

### **Overall Repackage Success Criteria**
- [ ] All plugins (Base, AWS, Cloudflare, Vercel) fully functional
- [ ] CLI integration complete
- [ ] All examples updated to use plugin imports
- [ ] Platform components removed
- [ ] Documentation updated
- [ ] CI/CD pipeline working for all plugins

### **Testing Standards**
- **Base Plugin**: Maintain current 6 tests
- **AWS Plugin**: Maintain current 271 tests ✅
- **Cloudflare Plugin**: Target 150+ tests (currently 21)
- **Vercel Plugin**: Target 50+ tests (new)

---

## 🚀 Implementation Timeline

### **Week 1: Cloudflare Plugin Migration**
- **Days 1-2**: Component base class and import standardization
- **Days 3-4**: Migration system implementation
- **Days 5-7**: Comprehensive testing suite

### **Week 2: Vercel Plugin & CLI Integration**
- **Days 1-3**: Vercel plugin creation and migration
- **Days 4-7**: CLI integration updates and plugin discovery

### **Week 3: Finalization & Documentation**
- **Days 1-3**: Platform cleanup and example updates
- **Days 4-5**: Documentation updates
- **Days 6-7**: Final testing and validation

**Total Estimated Duration**: 3 weeks (15-20 working days)

---

## Benefits
- **Modularity**: Users only install needed cloud provider plugins
- **Independent releases**: Plugins can be updated without full SST releases  
- **Better maintenance**: Cleaner separation between core and provider-specific code
- **Extensibility**: Easier for community to create custom plugins