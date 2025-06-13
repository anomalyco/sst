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

## Remaining Steps 🚧

## Remaining Steps 🚧

### High Priority
- [x] **Complete platform migration**: ✅ **PARTIALLY DONE** - Cloudflare components copied to plugin/cloudflare/, but imports need fixing
  - ❌ **TODO**: Fix all import paths in Cloudflare plugin (sst-plugin imports, .js extensions, relative paths) - **See detailed fix plan below**
  - ❌ **TODO**: Remove platform/src/components/cloudflare/ after migration is complete
  - ❌ **TODO**: Create Vercel plugin and migrate platform/src/components/vercel/
  - ❌ **TODO**: Remove platform/src/components/vercel/ after migration is complete
- [ ] **Update CLI integration**: Modify Go CLI to load plugins instead of platform components
- [ ] **Fix import paths**: Update all examples and internal references to use plugin imports
- [ ] **Plugin discovery**: Implement automatic plugin loading in the SST runtime

### Medium Priority  
- [ ] **Cloudflare plugin completion**: Finish migrating Cloudflare components
- [ ] **Plugin versioning**: Implement independent versioning for each plugin
- [ ] **Documentation updates**: Update docs to reflect new plugin architecture
- [ ] **Testing infrastructure**: Set up tests for individual plugins

### Low Priority
- [ ] **Plugin marketplace**: Design system for third-party plugins
- [ ] **Migration tooling**: Create tools to help users migrate existing configs
- [ ] **Performance optimization**: Optimize plugin loading and bundling

## Benefits
- **Modularity**: Users only install needed cloud provider plugins
- **Independent releases**: Plugins can be updated without full SST releases  
- **Better maintenance**: Cleaner separation between core and provider-specific code
- **Extensibility**: Easier for community to create custom plugins

---

## Cloudflare Plugin Fix Analysis & Implementation Plan

### Analysis: Components vs Plugins

#### Key Differences Identified

1. **Import Structure**:
   - **Components** (platform/src/components): Import from relative paths and direct modules
   - **Plugins** (plugin/*/src): Import from `sst-plugin/*` namespace

2. **Base Component Class**:
   - **Components**: Extend `Component` from `../component` 
   - **AWS Plugin**: Extends `AWSComponent` from `./component` which extends `BaseComponent` from `sst-plugin/component`
   - **Cloudflare Plugin**: Extends `Component` from `./component` which extends `BaseComponent` from `sst-plugin/component`

3. **Type System**:
   - **Components**: Use `Input<T>` from `../input`
   - **Plugins**: Use `sst.Input<T>` from `sst-plugin`

4. **Utility Functions**:
   - **Components**: Import utilities from relative paths (`../naming`, `../duration`, etc.)
   - **Plugins**: Import utilities from `sst-plugin/*` namespace

5. **Component Registration**:
   - **AWS Plugin**: Has proper `AWSComponent` base class with comprehensive naming rules and transformations
   - **Cloudflare Plugin**: Has incomplete `Component` base class missing key features

#### Issues in Cloudflare Plugin

1. **Missing Component Features**
   - No `registerVersion` method
   - No `componentType` and `componentName` properties
   - Incomplete transformation logic compared to AWS plugin

2. **Import Inconsistencies**
   - Missing proper imports for utilities
   - Inconsistent use of `sst-plugin` namespace

3. **Missing Infrastructure**
   - No proper component versioning system
   - Missing utility functions that AWS plugin has

### Step-by-Step Fix Instructions

#### Step 1: Update Cloudflare Component Base Class

**File**: `plugin/cloudflare/src/component.ts`

**Actions**:
1. Add missing imports from `sst-plugin`
2. Add `componentType` and `componentName` properties
3. Add `registerVersion` method
4. Ensure proper inheritance from `BaseComponent`
5. Add comprehensive naming rules for Cloudflare resources

**Changes needed**:
- Import `* as sst` from `sst-plugin`
- Add private properties for component tracking
- Implement version registration system
- Expand naming rules to match AWS plugin pattern

#### Step 2: Update Import Statements Across Cloudflare Plugin

**Files**: All `.ts` files in `plugin/cloudflare/src/`

**Actions**:
1. Replace relative imports with `sst-plugin` namespace imports where appropriate
2. Ensure consistent use of `sst.Input<T>` instead of direct Input imports
3. Update utility imports to use `sst-plugin` namespace

**Pattern changes**:
- `import { Input } from "../input"` → `import * as sst from "sst-plugin"`
- `import { Transform } from "../component"` → `import { Transform } from "sst-plugin/internal/transform"`

#### Step 3: Add Missing Utility Functions

**Files**: Create utility files in `plugin/cloudflare/src/util/` if needed

**Actions**:
1. Check if Cloudflare plugin needs specific utility functions
2. Create utility files following AWS plugin pattern
3. Ensure proper exports in index files

#### Step 4: Update Component Implementations

**Files**: All component files in `plugin/cloudflare/src/`

**Actions**:
1. Update class definitions to extend from proper base component
2. Ensure proper use of `sst.Input<T>` types
3. Add proper component registration calls
4. Update transform patterns to match plugin architecture

#### Step 5: Verify Package Dependencies

**File**: `plugin/cloudflare/package.json`

**Actions**:
1. Ensure `sst-plugin` dependency is properly configured
2. Verify Pulumi Cloudflare provider version compatibility
3. Check if additional dependencies are needed

#### Step 6: Update Build Configuration

**File**: `plugin/cloudflare/script/build.ts`

**Actions**:
1. Ensure build script follows AWS plugin pattern
2. Verify proper TypeScript compilation
3. Check export configuration

#### Step 7: Add Missing Component Features

**Actions**:
1. Add version management system
2. Implement proper error handling
3. Add comprehensive transformation rules
4. Ensure proper resource naming conventions

#### Step 8: Testing and Validation

**Actions**:
1. Build the plugin and verify no compilation errors
2. Test basic component creation
3. Verify proper resource naming
4. Test component linking functionality

### Implementation Priority

1. **High Priority**: Fix component base class and imports (Steps 1-2)
2. **Medium Priority**: Update component implementations (Step 4)
3. **Low Priority**: Add utilities and build improvements (Steps 3, 6-7)

### Expected Outcomes

After implementing these fixes:
1. Cloudflare plugin will have consistent architecture with AWS plugin
2. All components will properly extend from the correct base class
3. Import statements will be consistent with plugin architecture
4. Component versioning and transformation will work correctly
5. Resource naming will follow SST conventions

### Files to Modify

#### Critical Files:
- `plugin/cloudflare/src/component.ts` - Base component class
- `plugin/cloudflare/src/index.ts` - Main exports
- All component files in `plugin/cloudflare/src/` - Individual components

#### Supporting Files:
- `plugin/cloudflare/package.json` - Dependencies
- `plugin/cloudflare/script/build.ts` - Build configuration
- `plugin/cloudflare/tsconfig.json` - TypeScript configuration