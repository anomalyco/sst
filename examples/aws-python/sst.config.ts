/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Lambda Python
 *
 * SST uses [uv](https://docs.astral.sh/uv/) to manage your Python runtime, make
 * sure you have it [installed](https://docs.astral.sh/uv/getting-started/installation/).
 *
 * Any [uv workspace](https://docs.astral.sh/uv/concepts/projects/workspaces/#workspace-sources)
 * package can be built and deployed as a Lambda function using SST. Drop-in mode
 * is currently not supported.
 *
 * :::note
 * Builds currently do not tree shake so lots of workspaces can make the build
 * larger than necessary.
 * :::
 * 
 * In this example we deploy a handler from the `functions/` directory. It depends
 * on shared code from another uv workspace in the `core/` directory.
 *
 * ```txt
 * ├── sst.config.ts
 * ├── pyproject.toml
 * ├── core
 * │   ├── pyproject.toml
 * │   └── src
 * │       └── core
 * │           └── __init__.py
 * └── functions
 *     ├── pyproject.toml
 *     └── src
 *         └── functions
 *             ├── __init__.py
 *             └── api.py
 * ```
 *
 * The `handler` is the path to the handler file and the name of the handler function
 * in it.
 * 
 * ```ts title="sst.config.ts" {2}
 * new sst.aws.Function("MyPythonFunction", {
 *   handler: "functions/src/functions/api.handler",
 *   runtime: "python3.11",
 *   link: [linkableValue],
 *   url: true,
 * });
 * ```
 *
 * SST will traverse up from the handler path and look for the nearest
 * `pyproject.toml`. And will throw an error if it can't find one.
 *
 * To access linked resources, you can use the SST SDK.
 *
 * ```py title="functions/src/functions/api.py" {1}
 * from sst import Resource
 *
 * def handler(event, context):
 *     print(Resource.MyLinkableValue.foo)
 * ```
 *
 * Where the `sst` package can be added to your `pyproject.toml`.
 *
 * ```toml title="functions/pyproject.toml"
 * [tool.uv.sources]
 * sst = { git = "https://github.com/sst/sst.git", subdirectory = "sdk/python", branch = "dev" }
 * ```
 *
 * You also want to set the Python version in your `pyproject.toml` to the same
 * version as the one in Lambda.
 * 
 * ```toml title="functions/pyproject.toml"
 * requires-python = "==3.11.*"
 * ```
 *
 * This makes sure that your functions work the same in `sst dev` as `sst deploy`.
 */
export default $config({
	app(input) {
		return {
			name: "aws-python",
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

		new sst.aws.Function("MyPythonFunction", {
			handler: "functions/src/functions/api.handler",
			runtime: "python3.11",
			link: [linkableValue],
			url: true,
		});
	},
});
