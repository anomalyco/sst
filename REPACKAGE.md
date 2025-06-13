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

## Remaining Steps 🚧

### High Priority
- [ ] **Complete platform migration**: Remove remaining components from `platform/src/components/`
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