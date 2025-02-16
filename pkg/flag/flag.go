package flag

import (
	"os"
)

var SST_LOG = os.Getenv("SST_LOG")
var SST_PRINT_LOGS = isTrue("SST_PRINT_LOGS")
var SST_NO_CLEANUP = isTrue("SST_NO_CLEANUP")
var SST_PASSPHRASE = os.Getenv("SST_PASSPHRASE")
var SST_PULUMI_PATH = os.Getenv("SST_PULUMI_PATH")

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
