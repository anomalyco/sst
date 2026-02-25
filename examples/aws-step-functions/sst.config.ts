/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "aws-step-functions",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    // Create a function to be invoked by the state machine
    const app = new sst.aws.Function("MyApp", {
      handler: "index.handler",
    });

    // Define all the states of the state machine
    const lambdaInvoke = sst.aws.StepFunctions.lambdaInvoke({
      name: "LambdaInvoke",
      function: app,
      payload: {
        foo: "bar",
      },
    });
    const pass = sst.aws.StepFunctions.pass({ name: "Pass" });
    const wait = sst.aws.StepFunctions.wait({
      name: "Wait",
      time: "2 seconds",
    });
    const choice = sst.aws.StepFunctions.choice({ name: "Choice" });
    const parallel = sst.aws.StepFunctions.parallel({ name: "Parallel" });
    const parallelA = sst.aws.StepFunctions.pass({ name: "ParallelA" });
    const parallelB = sst.aws.StepFunctions.pass({ name: "ParallelB" });
    const parallelC = sst.aws.StepFunctions.pass({ name: "ParallelC" });
    const mapA = sst.aws.StepFunctions.pass({ name: "MapA" });
    const mapB = sst.aws.StepFunctions.pass({ name: "MapB" });
    const map = sst.aws.StepFunctions.map({
      name: "Map",
      processor: mapA.next(mapB),
      items: ["a", "b", "c"],
    });
    const success = sst.aws.StepFunctions.succeed({ name: "Succeed" });
    const fail = sst.aws.StepFunctions.fail({ name: "Fail" });
    const last = sst.aws.StepFunctions.pass({ name: "Last" });

    // Create the state machine
    new sst.aws.StepFunctions("MyStateMachine", {
      definition: lambdaInvoke
        .catch(fail)
        .next(pass)
        .next(wait)
        .next(parallel.branch(parallelA.next(parallelB)).branch(parallelC))
        .next(map)
        .next(
          choice
            .when("{% 1+1 = 2 %}", success)
            .when("{% 1+1 = 3 %}", fail)
            .otherwise(last)
        ),
    });
  },
});
