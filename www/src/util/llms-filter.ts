const INTERNAL_COMPONENTS = new Set([
  // AWS internal
  "docs/component/aws/alb",
  "docs/component/aws/cdn",
  "docs/component/aws/app-sync-resolver",
  "docs/component/aws/app-sync-function",
  "docs/component/aws/bucket-notification",
  "docs/component/aws/app-sync-data-source",
  "docs/component/aws/apigatewayv1-api-key",
  "docs/component/aws/bus-queue-subscriber",
  "docs/component/aws/cognito-user-pool-client",
  "docs/component/aws/bus-lambda-subscriber",
  "docs/component/aws/apigatewayv2-url-route",
  "docs/component/aws/apigatewayv1-authorizer",
  "docs/component/aws/apigatewayv1-usage-plan",
  "docs/component/aws/apigatewayv2-authorizer",
  "docs/component/aws/queue-lambda-subscriber",
  "docs/component/aws/sns-topic-queue-subscriber",
  "docs/component/aws/dynamo-lambda-subscriber",
  "docs/component/aws/realtime-lambda-subscriber",
  "docs/component/aws/sns-topic-lambda-subscriber",
  "docs/component/aws/apigatewayv1-lambda-route",
  "docs/component/aws/apigatewayv2-lambda-route",
  "docs/component/aws/apigateway-websocket-route",
  "docs/component/aws/providers/function-environment-update",
  "docs/component/aws/apigatewayv1-integration-route",
  "docs/component/aws/apigatewayv2-private-route",
  "docs/component/aws/cognito-identity-provider",
  "docs/component/aws/kinesis-stream-lambda-subscriber",
  // AWS StepFunctions internal
  "docs/component/aws/step-functions/fail",
  "docs/component/aws/step-functions/map",
  "docs/component/aws/step-functions/wait",
  "docs/component/aws/step-functions/task",
  "docs/component/aws/step-functions/pass",
  "docs/component/aws/step-functions/state",
  "docs/component/aws/step-functions/choice",
  "docs/component/aws/step-functions/parallel",
  "docs/component/aws/step-functions/succeed",
  // AWS deprecated
  "docs/component/aws/cron",
  "docs/component/aws/opencontrol",
  "docs/component/aws/vpc-v1",
  "docs/component/aws/redis-v1",
  "docs/component/aws/service-v1",
  "docs/component/aws/cluster-v1",
  "docs/component/aws/postgres-v1",
  "docs/component/aws/vector",
  // Cloudflare internal
  "docs/component/cloudflare/queue-worker-subscriber",
  // Cloudflare deprecated
  "docs/component/cloudflare/static-site",
  // Cross-provider internal
  "docs/component/aws/dns",
  "docs/component/vercel/dns",
  "docs/component/cloudflare/dns",
  "docs/component/cloudflare/binding",
  "docs/component/aws/permission",
]);

export function isExcludedFromLlms(slug: string): boolean {
  if (INTERNAL_COMPONENTS.has(slug)) return true;
  // Exclude individual example pages from llms.txt (catalog link suffices)
  if (slug.startsWith("docs/examples/")) return true;
  return false;
}

export function isExcludedFromLlmsFull(slug: string): boolean {
  if (INTERNAL_COMPONENTS.has(slug)) return true;
  // Exclude monolithic examples page (individual pages cover the content)
  if (slug === "docs/examples") return true;
  return false;
}
