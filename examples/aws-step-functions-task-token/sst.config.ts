/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "aws-step-functions-task-token",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    // Create a queue the state machine will send messages to
    const queue = new sst.aws.Queue("MyQueue");

    // Define all the states of the state machine
    const sendMessage = sst.aws.StepFunctions.sqsSendMessage({
      name: "SendMessage",
      integration: "token",
      queue,
      messageBody: {
        // Task token passed in the message body
        MyTaskToken: "{% $states.context.Task.Token %}",
      },
    });
    const success = sst.aws.StepFunctions.succeed({ name: "Succeed" });

    // Create the state machine
    const stepFunction = new sst.aws.StepFunctions("MyStateMachine", {
      definition: sendMessage.next(success),
    });

    // Create a function that will receive messages from the queue
    queue.subscribe({
      handler: "index.handler",
      // Linking the state machine to grant permissions to call `SendTaskSuccess`
      link: [stepFunction],
    });
  },
});
