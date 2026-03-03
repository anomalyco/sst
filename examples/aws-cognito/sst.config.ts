/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Cognito User Pool
 *
 * Create a Cognito User Pool with a hosted UI domain, client, and identity pool.
 *
 */
export default $config({
  app(input) {
    return {
      name: "aws-cognito",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const userPool = new sst.aws.CognitoUserPool("MyUserPool", {
      domain: {
        prefix: `my-app-${$app.stage}`,
      },
      triggers: {
        preSignUp: {
          handler: "index.handler",
        },
      },
    });

    const client = userPool.addClient("Web", {
      callbackUrls: ['https://example.com/auth/callback']
    });

    const identityPool = new sst.aws.CognitoIdentityPool("MyIdentityPool", {
      userPools: [
        {
          userPool: userPool.id,
          client: client.id,
        },
      ],
    });

    return {
      UserPool: userPool.id,
      Client: client.id,
      IdentityPool: identityPool.id,
      DomainUrl: userPool.domainUrl,
    };
  },
});
