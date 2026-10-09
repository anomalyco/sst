/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Lambda Python container
 *
 * Python Lambda function that use large dependencies like `numpy` and `pandas`, can
 * hit the 250MB Lambda package limit. To work around this, you can deploy them
 * as a container image to Lambda.
 *
 * :::tip
 * Container images on Lambda have a limit of 10GB.
 * :::
 * 
 * In this example, we deploy two functions as container image.
 *
 * ```ts title="sst.config.ts" {2-4}
 * const base = new sst.aws.Function("PythonFn", {
 *   python: {
 *     container: true,
 *   },
 *   handler: "./functions/src/functions/api.handler",
 *   runtime: "python3.11",
 *   link: [linkableValue],
 *   url: true,
 * });
 * ```
 *
 * Now when you run `sst deploy`, it uses a built-in Dockerfile to build the image
 * and deploy it. You'll need to have the Docker daemon running.
 *
 * :::note
 * You need to have the Docker daemon running locally.
 * :::
 *
 * To use a custom Dockerfile, you can place a `Dockerfile` in the root of the
 * uv workspace for your function.
 *
 * ```ts title="sst.config.ts" {5}
 * const custom = new sst.aws.Function("PythonFnCustom", {
 *   python: {
 *     container: true,
 *   },
 *   handler: "./custom_dockerfile/src/custom_dockerfile/api.handler",
 *   runtime: "python3.11",
 *   link: [linkableValue],
 *   url: true,
 * });
 * ```
 *
 * Here we have a `Dockerfile` in the `custom_dockerfile/` directory.
 *
 * ```dockerfile title="custom_dockerfile/Dockerfile"
 * # The python version to use is supplied as an arg from SST
 * ARG PYTHON_VERSION=3.11
 * 
 * # Use an official AWS Lambda base image for Python
 * FROM public.ecr.aws/lambda/python:${PYTHON_VERSION}
 *
 * # ...
 * ```
 *
 * You can also pass build arguments to the Docker build using `buildArgs`. This is
 * useful for passing secrets or configuration values that are needed during the
 * Docker build process, such as authentication tokens for private package registries.
 *
 * ```ts title="sst.config.ts" {3-7}
 * const withBuildArgs = new sst.aws.Function("PythonFnBuildArgs", {
 *   python: {
 *     container: {
 *       buildArgs: {
 *         MY_BUILD_ARG: "my-value",
 *       },
 *     },
 *   },
 *   handler: "./build_args/src/build_args/api.handler",
 *   runtime: "python3.11",
 *   url: true,
 * });
 * ```
 *
 * In your Dockerfile, declare the ARG after the FROM statement to make it available
 * in the build stage.
 *
 * ```dockerfile title="build_args/Dockerfile" {7-11}
 * ARG PYTHON_VERSION=3.11
 * FROM public.ecr.aws/lambda/python:${PYTHON_VERSION}
 *
 * # Declare build args AFTER FROM to make them available in the build stage
 * ARG MY_BUILD_ARG="default_value"
 *
 * # Set as ENV so it's available at runtime
 * ENV MY_BUILD_ARG=${MY_BUILD_ARG}
 * ```
 *
 * The project structure looks something like this.
 *
 * ```txt {5,10}
 * ├── sst.config.ts
 * ├── pyproject.toml
 * ├── custom_dockerfile
 * │   ├── pyproject.toml
 * │   ├── Dockerfile
 * │   └── src
 * │       └── custom_dockerfile
 * │           └── api.py
 * └── build_args
 *     ├── pyproject.toml
 *     ├── Dockerfile
 *     └── src
 *         └── build_args
 *             └── api.py
 * ```
 *
 * Locally, you want to set the Python version in your `pyproject.toml` to make sure
 * that `sst dev` uses the same version as `sst deploy`.
 */
export default $config({
	app(input) {
		return {
			name: "aws-python-container",
			removal: input?.stage === "production" ? "retain" : "remove",
			home: "aws",
		};
	},
	async run() {
		const linkableValue = new sst.Linkable("MyLinkableValue", {
			properties: {
				foo: "Hello World",
			},
		});

		const base = new sst.aws.Function("PythonFn", {
			python: {
				container: true,
			},
			handler: "./functions/src/functions/api.handler",
			runtime: "python3.11",
			link: [linkableValue],
			url: true,
		});

		const custom = new sst.aws.Function("PythonFnCustom", {
			python: {
				container: true,
			},
			handler: "./custom_dockerfile/src/custom_dockerfile/api.handler",
			runtime: "python3.11",
			link: [linkableValue],
			url: true,
		});

		const withBuildArgs = new sst.aws.Function("PythonFnBuildArgs", {
			python: {
				container: {
					buildArgs: {
						MY_BUILD_ARG: "hello-from-build-args",
					},
				},
			},
			handler: "./build_args/src/build_args/api.handler",
			runtime: "python3.11",
			url: true,
		});

		return {
			base: base.url,
			custom: custom.url,
			withBuildArgs: withBuildArgs.url,
		};
	},
});
