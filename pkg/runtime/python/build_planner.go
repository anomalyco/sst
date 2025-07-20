package python

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sst/sst/v3/pkg/runtime"
)

// BuildPlanner creates optimized build plans for incremental builds
type BuildPlanner struct {
	// dependencyAnalyzer provides dependency information
	dependencyAnalyzer *DependencyAnalyzer

	// mutex protects concurrent access
	mutex sync.RWMutex

	// config stores planner configuration
	config BuildPlannerConfig
}

// BuildPlannerConfig configures the build planner
type BuildPlannerConfig struct {
	// DependencyAnalyzer for analyzing package dependencies
	DependencyAnalyzer *DependencyAnalyzer

	// EnableParallelBuilds enables parallel building of independent packages
	EnableParallelBuilds bool

	// MaxParallelBuilds is the maximum number of parallel builds
	MaxParallelBuilds int

	// OptimizeBuildOrder enables build order optimization
	OptimizeBuildOrder bool

	// EnableCacheReuse enables reusing cached build results
	EnableCacheReuse bool
}

// NewBuildPlanner creates a new build planner with the given configuration
func NewBuildPlanner(config BuildPlannerConfig) *BuildPlanner {
	if config.MaxParallelBuilds == 0 {
		config.MaxParallelBuilds = 4
	}

	return &BuildPlanner{
		dependencyAnalyzer: config.DependencyAnalyzer,
		config:             config,
	}
}

// CreateBuildPlan creates an optimized build plan for the given input
func (bp *BuildPlanner) CreateBuildPlan(ctx context.Context, input *runtime.BuildInput, layout *LayoutInfo, dependencies *DependencyAnalysis) (*BuildPlan, error) {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()

	plan := &BuildPlan{
		Packages:             []*PackageBuildInfo{},
		BuildOrder:           []string{},
		ParallelGroups:       [][]string{},
		RequiredDependencies: []string{},
		CacheHits:            []string{},
		ForcedRebuilds:       []string{},
	}

	// Convert dependency analysis to package build info
	if err := bp.createPackageBuildInfo(dependencies, plan); err != nil {
		return nil, fmt.Errorf("failed to create package build info: %w", err)
	}

	// Determine which packages need rebuilding
	if err := bp.determineRebuildRequirements(plan); err != nil {
		return nil, fmt.Errorf("failed to determine rebuild requirements: %w", err)
	}

	// Optimize build order
	if bp.config.OptimizeBuildOrder {
		if err := bp.optimizeBuildOrder(plan, dependencies); err != nil {
			return nil, fmt.Errorf("failed to optimize build order: %w", err)
		}
	} else {
		plan.BuildOrder = dependencies.BuildOrder
	}

	// Create parallel build groups
	if bp.config.EnableParallelBuilds {
		if err := bp.createParallelGroups(plan, dependencies); err != nil {
			return nil, fmt.Errorf("failed to create parallel groups: %w", err)
		}
	}

	// Collect required dependencies
	bp.collectRequiredDependencies(plan, dependencies)

	// Estimate build duration
	bp.estimateBuildDuration(plan)

	return plan, nil
}

// createPackageBuildInfo converts dependency analysis to package build info
func (bp *BuildPlanner) createPackageBuildInfo(dependencies *DependencyAnalysis, plan *BuildPlan) error {
	for _, localPkg := range dependencies.LocalPackages {
		buildInfo := &PackageBuildInfo{
			PackageName:        localPkg.Name,
			PackageDir:         localPkg.Path,
			Dependencies:       localPkg.Dependencies,
			SourceFiles:        localPkg.SourceFiles,
			RequiresRebuild:    localPkg.BuildRequired,
			RebuildReason:      localPkg.ChangeReason,
			EstimatedBuildTime: localPkg.EstimatedBuildTime,
			CanUseCache:        !localPkg.HasChanges,
			CacheKey:           bp.generateCacheKey(localPkg),
		}

		plan.Packages = append(plan.Packages, buildInfo)
	}

	return nil
}

// generateCacheKey generates a cache key for a package
func (bp *BuildPlanner) generateCacheKey(pkg *LocalPackageInfo) string {
	return fmt.Sprintf("%s:%s", pkg.Name, pkg.Path)
}

// determineRebuildRequirements determines which packages need rebuilding
func (bp *BuildPlanner) determineRebuildRequirements(plan *BuildPlan) error {
	for _, pkg := range plan.Packages {
		if pkg.RequiresRebuild {
			plan.ForcedRebuilds = append(plan.ForcedRebuilds, pkg.PackageName)
		} else if pkg.CanUseCache {
			plan.CacheHits = append(plan.CacheHits, pkg.PackageName)
		}
	}

	return nil
}

// optimizeBuildOrder optimizes the build order for better performance
func (bp *BuildPlanner) optimizeBuildOrder(plan *BuildPlan, dependencies *DependencyAnalysis) error {
	// Start with the dependency-based build order
	optimizedOrder := make([]string, len(dependencies.BuildOrder))
	copy(optimizedOrder, dependencies.BuildOrder)

	// Apply optimizations
	optimizedOrder = bp.prioritizeChangedPackages(optimizedOrder, plan)
	optimizedOrder = bp.optimizeForParallelism(optimizedOrder, dependencies)

	plan.BuildOrder = optimizedOrder
	return nil
}

// prioritizeChangedPackages moves packages that need rebuilding to the front
func (bp *BuildPlanner) prioritizeChangedPackages(buildOrder []string, plan *BuildPlan) []string {
	var changedPackages []string
	var unchangedPackages []string

	changedSet := make(map[string]bool)
	for _, pkg := range plan.Packages {
		if pkg.RequiresRebuild {
			changedSet[pkg.PackageName] = true
		}
	}

	for _, pkgName := range buildOrder {
		if changedSet[pkgName] {
			changedPackages = append(changedPackages, pkgName)
		} else {
			unchangedPackages = append(unchangedPackages, pkgName)
		}
	}

	// Return changed packages first, then unchanged
	return append(changedPackages, unchangedPackages...)
}

// optimizeForParallelism reorders packages to maximize parallel build opportunities
func (bp *BuildPlanner) optimizeForParallelism(buildOrder []string, dependencies *DependencyAnalysis) []string {
	if !bp.config.EnableParallelBuilds {
		return buildOrder
	}

	// Group packages by dependency level
	levels := bp.calculateDependencyLevels(dependencies.DependencyGraph)

	// Sort packages within each level by estimated build time (longest first)
	levelGroups := make(map[int][]string)
	for _, pkgName := range buildOrder {
		level := levels[pkgName]
		levelGroups[level] = append(levelGroups[level], pkgName)
	}

	// Sort each level by build time
	for level := range levelGroups {
		bp.sortByBuildTime(levelGroups[level], dependencies)
	}

	// Reconstruct build order
	var optimizedOrder []string
	maxLevel := 0
	for level := range levelGroups {
		if level > maxLevel {
			maxLevel = level
		}
	}

	for level := 0; level <= maxLevel; level++ {
		optimizedOrder = append(optimizedOrder, levelGroups[level]...)
	}

	return optimizedOrder
}

// calculateDependencyLevels calculates the dependency level for each package
func (bp *BuildPlanner) calculateDependencyLevels(graph map[string][]string) map[string]int {
	levels := make(map[string]int)
	visited := make(map[string]bool)

	var calculateLevel func(string) int
	calculateLevel = func(pkg string) int {
		if visited[pkg] {
			return levels[pkg]
		}

		visited[pkg] = true
		maxDepLevel := -1

		for _, dep := range graph[pkg] {
			depLevel := calculateLevel(dep)
			if depLevel > maxDepLevel {
				maxDepLevel = depLevel
			}
		}

		levels[pkg] = maxDepLevel + 1
		return levels[pkg]
	}

	for pkg := range graph {
		calculateLevel(pkg)
	}

	return levels
}

// sortByBuildTime sorts packages by estimated build time (longest first)
func (bp *BuildPlanner) sortByBuildTime(packages []string, dependencies *DependencyAnalysis) {
	// Create a map of package names to build times
	buildTimes := make(map[string]time.Duration)
	for _, pkg := range dependencies.LocalPackages {
		buildTimes[pkg.Name] = pkg.EstimatedBuildTime
	}

	// Simple bubble sort by build time (descending)
	for i := 0; i < len(packages)-1; i++ {
		for j := i + 1; j < len(packages); j++ {
			if buildTimes[packages[i]] < buildTimes[packages[j]] {
				packages[i], packages[j] = packages[j], packages[i]
			}
		}
	}
}

// createParallelGroups creates groups of packages that can be built in parallel
func (bp *BuildPlanner) createParallelGroups(plan *BuildPlan, dependencies *DependencyAnalysis) error {
	if !bp.config.EnableParallelBuilds {
		// Create sequential groups (one package per group)
		for _, pkgName := range plan.BuildOrder {
			plan.ParallelGroups = append(plan.ParallelGroups, []string{pkgName})
		}
		return nil
	}

	// Calculate dependency levels
	levels := bp.calculateDependencyLevels(dependencies.DependencyGraph)

	// Group packages by dependency level
	levelGroups := make(map[int][]string)
	for _, pkgName := range plan.BuildOrder {
		level := levels[pkgName]
		levelGroups[level] = append(levelGroups[level], pkgName)
	}

	// Create parallel groups from level groups
	maxLevel := 0
	for level := range levelGroups {
		if level > maxLevel {
			maxLevel = level
		}
	}

	for level := 0; level <= maxLevel; level++ {
		if packages, exists := levelGroups[level]; exists && len(packages) > 0 {
			// Split large groups to respect MaxParallelBuilds
			for len(packages) > 0 {
				groupSize := bp.config.MaxParallelBuilds
				if len(packages) < groupSize {
					groupSize = len(packages)
				}

				group := packages[:groupSize]
				packages = packages[groupSize:]

				plan.ParallelGroups = append(plan.ParallelGroups, group)
			}
		}
	}

	return nil
}

// collectRequiredDependencies collects all external dependencies needed for the build
func (bp *BuildPlanner) collectRequiredDependencies(plan *BuildPlan, dependencies *DependencyAnalysis) {
	depSet := make(map[string]bool)

	// Collect dependencies from packages that need rebuilding
	for _, pkg := range plan.Packages {
		if pkg.RequiresRebuild {
			for _, dep := range pkg.Dependencies {
				if !bp.isLocalPackage(dep, dependencies.LocalPackages) {
					depSet[dep] = true
				}
			}
		}
	}

	// Convert set to slice
	for dep := range depSet {
		plan.RequiredDependencies = append(plan.RequiredDependencies, dep)
	}
}

// isLocalPackage checks if a dependency is a local package
func (bp *BuildPlanner) isLocalPackage(depName string, localPackages []*LocalPackageInfo) bool {
	for _, pkg := range localPackages {
		if pkg.Name == depName {
			return true
		}
	}
	return false
}

// estimateBuildDuration estimates the total build duration
func (bp *BuildPlanner) estimateBuildDuration(plan *BuildPlan) {
	if bp.config.EnableParallelBuilds && len(plan.ParallelGroups) > 0 {
		// For parallel builds, duration is the sum of the longest package in each group
		var totalDuration time.Duration

		for _, group := range plan.ParallelGroups {
			var maxGroupDuration time.Duration

			for _, pkgName := range group {
				for _, pkg := range plan.Packages {
					if pkg.PackageName == pkgName && pkg.RequiresRebuild {
						if pkg.EstimatedBuildTime > maxGroupDuration {
							maxGroupDuration = pkg.EstimatedBuildTime
						}
						break
					}
				}
			}

			totalDuration += maxGroupDuration
		}

		plan.EstimatedDuration = totalDuration
	} else {
		// For sequential builds, duration is the sum of all package build times
		var totalDuration time.Duration

		for _, pkg := range plan.Packages {
			if pkg.RequiresRebuild {
				totalDuration += pkg.EstimatedBuildTime
			}
		}

		plan.EstimatedDuration = totalDuration
	}

	// Add overhead for dependency installation and setup
	overhead := 10 * time.Second
	if len(plan.RequiredDependencies) > 10 {
		overhead = 30 * time.Second
	}

	plan.EstimatedDuration += overhead
}

// OptimizeBuildPlan applies additional optimizations to an existing build plan
func (bp *BuildPlanner) OptimizeBuildPlan(plan *BuildPlan, dependencies *DependencyAnalysis) error {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()

	// Re-evaluate cache usage
	if err := bp.reevaluateCacheUsage(plan); err != nil {
		return fmt.Errorf("failed to re-evaluate cache usage: %w", err)
	}

	// Optimize parallel groups
	if bp.config.EnableParallelBuilds {
		if err := bp.optimizeParallelGroups(plan, dependencies); err != nil {
			return fmt.Errorf("failed to optimize parallel groups: %w", err)
		}
	}

	// Re-estimate duration
	bp.estimateBuildDuration(plan)

	return nil
}

// reevaluateCacheUsage re-evaluates which packages can use cached results
func (bp *BuildPlanner) reevaluateCacheUsage(plan *BuildPlan) error {
	// Clear existing cache hits
	plan.CacheHits = []string{}

	// Re-evaluate each package
	for _, pkg := range plan.Packages {
		if !pkg.RequiresRebuild && pkg.CanUseCache {
			plan.CacheHits = append(plan.CacheHits, pkg.PackageName)
		}
	}

	return nil
}

// optimizeParallelGroups optimizes the parallel build groups for better performance
func (bp *BuildPlanner) optimizeParallelGroups(plan *BuildPlan, dependencies *DependencyAnalysis) error {
	if len(plan.ParallelGroups) <= 1 {
		return nil // Nothing to optimize
	}

	// Balance group sizes to maximize parallelism
	optimizedGroups := bp.balanceGroupSizes(plan.ParallelGroups, dependencies)
	plan.ParallelGroups = optimizedGroups

	return nil
}

// balanceGroupSizes balances the sizes of parallel groups for optimal performance
func (bp *BuildPlanner) balanceGroupSizes(groups [][]string, dependencies *DependencyAnalysis) [][]string {
	// Create a map of package build times
	buildTimes := make(map[string]time.Duration)
	for _, pkg := range dependencies.LocalPackages {
		buildTimes[pkg.Name] = pkg.EstimatedBuildTime
	}

	// Calculate current group durations
	groupDurations := make([]time.Duration, len(groups))
	for i, group := range groups {
		var maxDuration time.Duration
		for _, pkgName := range group {
			if buildTime, exists := buildTimes[pkgName]; exists {
				if buildTime > maxDuration {
					maxDuration = buildTime
				}
			}
		}
		groupDurations[i] = maxDuration
	}

	// For now, return groups as-is
	// More sophisticated balancing could be implemented here
	return groups
}

// GetBuildPlanStats returns statistics about a build plan
func (bp *BuildPlanner) GetBuildPlanStats(plan *BuildPlan) *BuildPlanStats {
	stats := &BuildPlanStats{
		TotalPackages:        len(plan.Packages),
		PackagesToBuild:      0,
		PackagesFromCache:    len(plan.CacheHits),
		ParallelGroups:       len(plan.ParallelGroups),
		EstimatedDuration:    plan.EstimatedDuration,
		RequiredDependencies: len(plan.RequiredDependencies),
	}

	for _, pkg := range plan.Packages {
		if pkg.RequiresRebuild {
			stats.PackagesToBuild++
		}
	}

	return stats
}

// BuildPlanStats contains statistics about a build plan
type BuildPlanStats struct {
	TotalPackages        int           `json:"totalPackages"`
	PackagesToBuild      int           `json:"packagesToBuild"`
	PackagesFromCache    int           `json:"packagesFromCache"`
	ParallelGroups       int           `json:"parallelGroups"`
	EstimatedDuration    time.Duration `json:"estimatedDuration"`
	RequiredDependencies int           `json:"requiredDependencies"`
}

// ValidateBuildPlan validates that a build plan is consistent and executable
func (bp *BuildPlanner) ValidateBuildPlan(plan *BuildPlan, dependencies *DependencyAnalysis) error {
	// Check that all packages in build order exist
	packageSet := make(map[string]bool)
	for _, pkg := range plan.Packages {
		packageSet[pkg.PackageName] = true
	}

	for _, pkgName := range plan.BuildOrder {
		if !packageSet[pkgName] {
			return fmt.Errorf("package %s in build order not found in package list", pkgName)
		}
	}

	// Check that parallel groups don't violate dependencies
	if err := bp.validateParallelGroups(plan, dependencies); err != nil {
		return fmt.Errorf("invalid parallel groups: %w", err)
	}

	return nil
}

// validateParallelGroups validates that parallel groups don't violate dependencies
func (bp *BuildPlanner) validateParallelGroups(plan *BuildPlan, dependencies *DependencyAnalysis) error {
	// Create a map of package positions in the build order
	positions := make(map[string]int)
	for i, pkgName := range plan.BuildOrder {
		positions[pkgName] = i
	}

	// Check each parallel group
	for _, group := range plan.ParallelGroups {
		for _, pkgName := range group {
			// Check that all dependencies of this package come before it
			for _, dep := range dependencies.DependencyGraph[pkgName] {
				if depPos, exists := positions[dep]; exists {
					if pkgPos, exists := positions[pkgName]; exists {
						if depPos >= pkgPos {
							return fmt.Errorf("dependency %s of package %s comes after it in build order", dep, pkgName)
						}
					}
				}
			}
		}
	}

	return nil
}
