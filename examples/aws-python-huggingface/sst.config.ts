/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Lambda Python Hugging Face
 *
 * Uses a Python Lambda container image to deploy a lightweight
 * [Hugging Face](https://huggingface.co/) model.
 *
 * Uses the [transformers](https://github.com/huggingface/transformers) library to
 * generate text using the
 * [TinyStories-33M](https://huggingface.co/roneneldan/TinyStories-33M) model. The
 * backend is the pytorch cpu runtime.
 * 
 * :::note
 * This is not a production ready example.
 * :::
 *
 * This example also shows how it is possible to use custom index resolution to get
 * dependencies from a private pypi server such as the pytorch cpu link. This
 * example also shows how to use a custom Dockerfile to handle complex builds such
 * as installing pytorch and pruning the build size.
 */
export default $config({
	app(input) {
		return {
			name: "aws-python-huggingface",
			removal: input?.stage === "production" ? "retain" : "remove",
			home: "aws",
		};
	},
	async run() {
		new sst.aws.Function("MyPythonFunction", {
			python: {
				container: true,
			},
			handler: "functions/src/functions/api.handler",
			runtime: "python3.12",
			timeout: "60 seconds",
			url: true,
		});
	},
});
