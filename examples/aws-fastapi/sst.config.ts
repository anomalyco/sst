/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## FastAPI
 *
 * Deploy a Python FastAPI app as a Lambda function with a linked value.
 */
export default $config({
	app(input) {
		return {
			name: "aws-fastapi",
			removal: input?.stage === "production" ? "retain" : "remove",
			home: "aws",
			providers: {
				aws: true,
			},
		};
	},
	async run() {
		const linkableValue = new sst.Linkable("MyLinkableValue", {
			properties: {
				foo: "Hello World",
			},
		});

		const fastapi = new sst.aws.Function("FastAPI", {
			handler: "functions/src/functions/api.handler",
			runtime: "python3.11",
			url: true,
			link: [linkableValue],
		});

		return {
			fastapi: fastapi.url,
		};
	},
});
