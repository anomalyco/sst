package flag

import (
	"os"
)

var SST_LOG = os.Getenv("SST_LOG")
var SST_LOG_CHILDREN = isTrue("SST_LOG_CHILDREN")
var SST_PRINT_LOGS = isTrue("SST_PRINT_LOGS")
var SST_NO_CLEANUP = isTrue("SST_NO_CLEANUP")
var SST_PASSPHRASE = os.Getenv("SST_PASSPHRASE")
var SST_PULUMI_PATH = os.Getenv("SST_PULUMI_PATH")
var SST_BUN_PATH = os.Getenv("SST_BUN_PATH")

// SST_BUILD_CONCURRENCY is deprecated, use SST_FUNCTION_BUILD_CONCURRENCY instead
var SST_BUILD_CONCURRENCY = os.Getenv("SST_BUILD_CONCURRENCY")
var SST_BUILD_CONCURRENCY_FUNCTION = os.Getenv("SST_BUILD_CONCURRENCY_FUNCTION")
var SST_BUILD_CONCURRENCY_SITE = os.Getenv("SST_BUILD_CONCURRENCY_SITE")
var SST_SKIP_DEPENDENCY_CHECK = isTrue("SST_SKIP_DEPENDENCY_CHECK")
var SST_TELEMETRY_DISABLED = isTrue("SST_TELEMETRY_DISABLED") || isTrue("DO_NOT_TRACK")
var SST_BUN_VERSION = os.Getenv("SST_BUN_VERSION")
var SST_VERBOSE = isTrue("SST_VERBOSE")
var SST_EXPERIMENTAL = isTrue("SST_EXPERIMENTAL") || isTrue("SST_EXPERIMENTAL_RUN")
var SST_RUN_ID = os.Getenv("SST_RUN_ID")
var SST_SKIP_APPSYNC = isTrue("SST_SKIP_APPSYNC")
var SST_NO_BUN = isTrue("NO_BUN") || isTrue("SST_NO_BUN")

// SST_DEV_SKIP_UNCHANGED enables connect-only dev: skip the per-startup `pulumi up` when
// the fully-resolved deployment fingerprint (config/infra bundle + resolved env, links,
// secrets, and versions) matches the last successful deploy, and let the dev bridge reuse
// already-deployed state.
var SST_DEV_SKIP_UNCHANGED = isTrue("SST_DEV_SKIP_UNCHANGED")

// SST_DEV_SKIP_MAX_AGE (seconds) forces a real deploy when the persisted fingerprint is
// older than this, so out-of-band drift (resources changed outside this stack, which
// input-hashing cannot detect) is reconciled periodically. Empty or "0" disables the limit.
var SST_DEV_SKIP_MAX_AGE = os.Getenv("SST_DEV_SKIP_MAX_AGE")

func isTrue(name string) bool {
	val, ok := os.LookupEnv(name)
	if !ok {
		return false
	}
	if val == "1" {
		return true
	}
	if val == "true" {
		return true
	}
	return false
}
