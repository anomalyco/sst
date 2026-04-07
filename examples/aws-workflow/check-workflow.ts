import assert from "node:assert/strict";
import { workflow } from "../../sdk/js/src/aws/workflow.ts";

interface AwsCliCredentials {
  AccessKeyId: string;
  SecretAccessKey: string;
  SessionToken?: string;
}

interface ListVersionsResponse {
  Versions?: Array<{
    Version?: string;
  }>;
}

interface GetFunctionResponse {
  Configuration?: {
    LoggingConfig?: {
      LogGroup?: string;
    };
  };
}

interface FilterLogEventsResponse {
  events?: Array<{
    message?: string;
  }>;
}

interface StartedExecution {
  started: workflow.StartResponse;
  executionArn: string;
  executionName: string;
  createdAtFrom: Date;
}

interface PendingCallbackExecution extends StartedExecution {
  callbackId: string;
  runningDescribe: workflow.DescribeResponse;
}

interface Outputs {
  workflow?: string;
}

const region =
  getArg("region") ?? process.env.AWS_REGION ?? process.env.AWS_DEFAULT_REGION ?? "us-east-1";

async function main() {
  await loadAwsCredentials();

  const functionName =
    getArg("name") ?? process.env.WORKFLOW_NAME ?? (await readDefaultWorkflowName());
  if (!functionName) {
    throw new Error("Missing workflow name. Pass --name or set WORKFLOW_NAME.");
  }

  const qualifier =
    getArg("qualifier") ??
    process.env.WORKFLOW_QUALIFIER ??
    (await resolvePublishedQualifier(functionName));
  const logGroup = await resolveLogGroup(functionName);

  const resource: workflow.Resource = {
    name: functionName,
    qualifier,
  };
  const options: workflow.Options = {
    aws: {
      region,
    },
  };

  const lifecycle = await verifyLifecycleMethods(resource, options);
  const callbacks = {
    succeed: await verifySucceedCallback(resource, options, logGroup),
    fail: await verifyFailCallback(resource, options, logGroup),
    heartbeat: await verifyHeartbeatCallback(resource, options, logGroup),
  };

  console.log(
    JSON.stringify(
      {
        functionName,
        qualifier,
        region,
        logGroup,
        lifecycle,
        callbacks,
      },
      null,
      2,
    ),
  );
}

function getArg(name: string) {
  const flag = `--${name}`;

  for (let index = 2; index < Bun.argv.length; index++) {
    const value = Bun.argv[index];
    if (value === flag) return Bun.argv[index + 1];
    if (value.startsWith(`${flag}=`)) return value.slice(flag.length + 1);
  }
}

async function readDefaultWorkflowName() {
  const file = Bun.file(new URL("./.sst/outputs.json", import.meta.url));
  if (!(await file.exists())) return;

  const outputs = (await file.json()) as Outputs;
  return outputs.workflow;
}

async function verifyLifecycleMethods(
  resource: workflow.Resource,
  options: workflow.Options,
) {
  const execution = await startExecution(resource, options, "workflow-method-check", {
    message: "workflow method check",
  });
  let stopped = false;

  try {
    const runningDescription = await waitForExecutionStatus(
      execution.executionArn,
      "RUNNING",
      options,
      "running describe",
    );

    assert.equal(runningDescription.name, execution.executionName);
    assert.equal(runningDescription.arn, execution.executionArn);
    assert.match(
      runningDescription.functionArn,
      new RegExp(`${resource.name}(:|$)`),
    );

    const runningListEntry = await waitForExecutionInList(
      resource,
      execution.createdAtFrom,
      execution.executionArn,
      "RUNNING",
      options,
      "running list",
    );

    assert.equal(runningListEntry.name, execution.executionName);
    assert.equal(runningListEntry.status, "RUNNING");

    const stopResult = await workflow.stop(
      execution.executionArn,
      {
        error: new Error("workflow method check stop"),
      },
      options,
    );
    stopped = true;

    assert.equal(stopResult.arn, execution.executionArn);
    assert.equal(stopResult.status, "STOPPED");

    const stoppedDescription = await waitForExecutionStatus(
      execution.executionArn,
      "STOPPED",
      options,
      "stopped describe",
    );
    const stoppedListEntry = await waitForExecutionInList(
      resource,
      execution.createdAtFrom,
      execution.executionArn,
      "STOPPED",
      options,
      "stopped list",
    );

    assert.equal(stoppedDescription.name, execution.executionName);
    assert.equal(stoppedListEntry.name, execution.executionName);

    return {
      start: execution.started,
      runningList: summarizeExecution(runningListEntry),
      runningDescribe: summarizeExecution(runningDescription),
      stop: stopResult,
      stoppedList: summarizeExecution(stoppedListEntry),
      stoppedDescribe: summarizeExecution(stoppedDescription),
    };
  } finally {
    if (!stopped) {
      await cleanupExecution(execution.executionArn, options, "workflow method check");
    }
  }
}

async function verifySucceedCallback(
  resource: workflow.Resource,
  options: workflow.Options,
  logGroup: string,
) {
  const execution = await startPendingCallbackExecution(
    resource,
    options,
    logGroup,
    "workflow-callback-succeed-check",
  );
  let completed = false;

  try {
    await workflow.succeed(
      execution.callbackId,
      {
        payload: {
          message: "workflow callback succeed check",
          resolvedAt: new Date().toISOString(),
        },
      },
      options,
    );

    const succeededDescription = await waitForExecutionStatus(
      execution.executionArn,
      "SUCCEEDED",
      options,
      "succeed describe",
    );
    completed = true;

    return {
      start: execution.started,
      runningDescribe: summarizeExecution(execution.runningDescribe),
      succeededDescribe: summarizeExecution(succeededDescription),
    };
  } finally {
    if (!completed) {
      await cleanupExecution(execution.executionArn, options, "succeed callback check");
    }
  }
}

async function verifyFailCallback(
  resource: workflow.Resource,
  options: workflow.Options,
  logGroup: string,
) {
  const execution = await startPendingCallbackExecution(
    resource,
    options,
    logGroup,
    "workflow-callback-fail-check",
  );
  let completed = false;

  try {
    await workflow.fail(
      execution.callbackId,
      {
        error: {
          data: {
            executionName: execution.executionName,
          },
          message: "workflow callback fail check",
          type: "CallbackCheckError",
        },
      },
      options,
    );

    const failedDescription = await waitForExecutionStatus(
      execution.executionArn,
      "FAILED",
      options,
      "fail describe",
    );
    completed = true;

    return {
      start: execution.started,
      runningDescribe: summarizeExecution(execution.runningDescribe),
      failedDescribe: summarizeExecution(failedDescription),
    };
  } finally {
    if (!completed) {
      await cleanupExecution(execution.executionArn, options, "fail callback check");
    }
  }
}

async function verifyHeartbeatCallback(
  resource: workflow.Resource,
  options: workflow.Options,
  logGroup: string,
) {
  const execution = await startPendingCallbackExecution(
    resource,
    options,
    logGroup,
    "workflow-callback-heartbeat-check",
  );
  let completed = false;

  try {
    await workflow.heartbeat(execution.callbackId, options);

    const runningAfterHeartbeat = await waitForExecutionStatus(
      execution.executionArn,
      "RUNNING",
      options,
      "heartbeat running describe",
    );

    await workflow.succeed(
      execution.callbackId,
      {
        payload: {
          message: "workflow callback heartbeat check",
          resolvedAt: new Date().toISOString(),
        },
      },
      options,
    );

    const succeededDescription = await waitForExecutionStatus(
      execution.executionArn,
      "SUCCEEDED",
      options,
      "heartbeat succeed describe",
    );
    completed = true;

    return {
      start: execution.started,
      runningDescribe: summarizeExecution(execution.runningDescribe),
      runningAfterHeartbeat: summarizeExecution(runningAfterHeartbeat),
      succeededDescribe: summarizeExecution(succeededDescription),
    };
  } finally {
    if (!completed) {
      await cleanupExecution(execution.executionArn, options, "heartbeat callback check");
    }
  }
}

async function startExecution(
  resource: workflow.Resource,
  options: workflow.Options,
  namePrefix: string,
  payload: Record<string, unknown>,
): Promise<StartedExecution> {
  const createdAtFrom = new Date(Date.now() - 60_000);
  const executionName = `${namePrefix}-${Date.now()}`;
  const started = await workflow.start(
    resource,
    {
      name: executionName,
      payload,
    },
    options,
  );

  assert.equal(started.statusCode, 202, `${namePrefix} start should return 202`);
  assert.ok(started.arn, `${namePrefix} start should return an execution ARN`);

  return {
    started,
    executionArn: started.arn,
    executionName,
    createdAtFrom,
  };
}

async function startPendingCallbackExecution(
  resource: workflow.Resource,
  options: workflow.Options,
  logGroup: string,
  namePrefix: string,
): Promise<PendingCallbackExecution> {
  const logStartTime = Date.now() - 5_000;
  const execution = await startExecution(resource, options, namePrefix, {
    message: namePrefix,
  });

  const runningDescribe = await waitForExecutionStatus(
    execution.executionArn,
    "RUNNING",
    options,
    `${namePrefix} running describe`,
  );
  const callbackId = await waitForCallbackId(
    logGroup,
    execution.executionArn,
    logStartTime,
  );

  return {
    ...execution,
    callbackId,
    runningDescribe,
  };
}

async function waitForExecutionStatus(
  executionArn: string,
  status: workflow.ExecutionStatus,
  options: workflow.Options,
  label: string,
) {
  return waitFor(label, async () => {
    const described = await workflow.describe(executionArn, options);
    return described.status === status ? described : undefined;
  });
}

async function waitForExecutionInList(
  resource: workflow.Resource,
  createdAtFrom: Date,
  executionArn: string,
  status: workflow.ExecutionStatus,
  options: workflow.Options,
  label: string,
) {
  return waitFor(label, async () => {
    const listed = await workflow.list(
      resource,
      {
        status,
        createdAt: {
          from: createdAtFrom,
          order: "desc",
        },
      },
      options,
    );

    return listed.executions.find((execution) => execution.arn === executionArn);
  });
}

async function resolveLogGroup(functionName: string) {
  const response = await awsJson<GetFunctionResponse>([
    "lambda",
    "get-function",
    "--function-name",
    functionName,
    "--region",
    region,
  ]);

  const logGroup = response.Configuration?.LoggingConfig?.LogGroup;
  if (!logGroup) {
    throw new Error(`Missing log group for ${functionName}`);
  }

  return logGroup;
}

async function waitForCallbackId(
  logGroup: string,
  executionArn: string,
  startTime: number,
) {
  return waitFor("callback id", async () => {
    const response = await awsJson<FilterLogEventsResponse>([
      "logs",
      "filter-log-events",
      "--log-group-name",
      logGroup,
      "--start-time",
      String(startTime),
      "--region",
      region,
    ]);

    for (const event of response.events ?? []) {
      const log = parseLogMessage(event.message);
      if (!log || log.executionArn !== executionArn) continue;

      const message = log.message;
      if (
        message &&
        typeof message === "object" &&
        typeof message.callbackId === "string"
      ) {
        return message.callbackId;
      }
    }
  });
}

async function cleanupExecution(
  executionArn: string,
  options: workflow.Options,
  label: string,
) {
  try {
    await workflow.stop(
      executionArn,
      {
        error: new Error(`${label} cleanup`),
      },
      options,
    );
  } catch (error) {
    console.error(`cleanup stop failed: ${formatError(error)}`);
  }
}

async function loadAwsCredentials() {
  if (process.env.AWS_ACCESS_KEY_ID && process.env.AWS_SECRET_ACCESS_KEY) {
    process.env.AWS_REGION ??= region;
    return;
  }

  const credentials = await awsJson<AwsCliCredentials>([
    "configure",
    "export-credentials",
    "--format",
    "process",
  ]);

  process.env.AWS_ACCESS_KEY_ID = credentials.AccessKeyId;
  process.env.AWS_SECRET_ACCESS_KEY = credentials.SecretAccessKey;
  if (credentials.SessionToken) {
    process.env.AWS_SESSION_TOKEN = credentials.SessionToken;
  }
  process.env.AWS_REGION = region;
}

async function resolvePublishedQualifier(functionName: string) {
  const response = await awsJson<ListVersionsResponse>([
    "lambda",
    "list-versions-by-function",
    "--function-name",
    functionName,
    "--region",
    region,
  ]);

  const versions = (response.Versions ?? [])
    .map((version) => version.Version)
    .filter((version): version is string => Boolean(version) && version !== "$LATEST")
    .sort((left, right) => Number(left) - Number(right));

  const qualifier = versions.at(-1);
  if (!qualifier) {
    throw new Error(`No published qualifier found for ${functionName}`);
  }

  return qualifier;
}

async function waitFor<T>(
  label: string,
  input: () => Promise<T | undefined>,
  timeout = 45_000,
  interval = 2_000,
) {
  const startedAt = Date.now();
  let lastError: unknown;

  while (Date.now() - startedAt < timeout) {
    try {
      const result = await input();
      if (result !== undefined) return result;
    } catch (error) {
      lastError = error;
    }

    await Bun.sleep(interval);
  }

  if (lastError) {
    throw new Error(`${label} timed out: ${formatError(lastError)}`);
  }

  throw new Error(`${label} timed out`);
}

function summarizeExecution(execution: workflow.Execution) {
  return {
    arn: execution.arn,
    name: execution.name,
    status: execution.status,
    createdAt: execution.createdAt.toISOString(),
    endedAt: execution.endedAt?.toISOString(),
    functionArn: execution.functionArn,
  };
}

function parseLogMessage(message: string | undefined) {
  if (!message) return;

  try {
    const parsed = JSON.parse(message) as {
      executionArn?: string;
      message?: {
        callbackId?: string;
      } | string;
    };

    return parsed;
  } catch {
    return;
  }
}

function formatError(error: unknown) {
  if (error instanceof Error) return `${error.name}: ${error.message}`;
  return String(error);
}

async function awsJson<T>(args: string[]) {
  const process = Bun.spawn(["aws", ...args], {
    stderr: "pipe",
    stdout: "pipe",
  });
  const [exitCode, stdout, stderr] = await Promise.all([
    process.exited,
    new Response(process.stdout).text(),
    new Response(process.stderr).text(),
  ]);

  if (exitCode !== 0) {
    throw new Error(stderr.trim() || `aws ${args.join(" ")} failed`);
  }

  return JSON.parse(stdout) as T;
}

await main();
