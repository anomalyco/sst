# ❍ Monorepo Example with Python and Node.js

Deploy python & node applications from a monorepo using sst.

## Requirements

* [Node.js](https://nodejs.org/)
* [Python](https://www.python.org/)
* [Pnpm](https://pnpm.io/installation)
* [uv](https://docs.astral.sh/uv/getting-started/installation/)

## Setup

### Install the dependencies (local dev):

#### Node.js
```bash
$ pnpm install
```

#### Python
```bash
$ uv sync --all-packages
```

### Development

```bash
$ pnpm sst dev
```

### Deployment

```bash
$ pnpm sst deploy
```

## Python

SST uses [uv](https://github.com/astral-sh/uv) to manage your python runtime. If you do not have uv installed, you can install it [here](https://docs.astral.sh/uv/getting-started/installation/). Any sst workspace package can be built and deployed to aws lambda using sst. In this example we deploy an API handler to lambda from `apps/python-backend`. The handler depends on shared code from the `packages/python-lib/` directory using uv's workspaces feature. (Note: builds currently do not tree shake so lots of workspaces can make larger builds than necessary.)

Python functions can be deployed just like other SST functions, the only difference is that the functions themselves must be configured within a uv workspace, there is no drop-in-mode.

If you are using live lambdas for your python functions, it is recommended to specify your python version to match your Lambda runtime otherwise you may encounter issues with dependencies.

```toml title="apps/python-backend/src/pyproject.toml"
[project]
name = "aws-python-node-monorepo"
version = "0.1.0"
description = "An SST app"
authors = [
    {name = "<your_name_here>", email = "<your_email_here>" },
]
requires-python = "==3.11.*"
```

Live lambda will locally run your python code by building the workspace and running the specified handler. You can have multiple handlers in the same workspace and have multiple workspaces in the same project that co-exist with other languages.

```markdown
.
├── apps
│   └── python-backend
│       ├── pyproject.toml
│       └── src
│           └── functions
│               ├── __init__.py
│               └── api.py       
│  
└── packages
    ├── python-lib
    │   ├── pyproject.toml
    │   └── src
    │       └── lib
    │           ├── __init__.py
    │           └── ping.py      

```

**Important:** Monorepo configurations with Python require an additional config for SST to work as intended. You can specify the `monorepoPath` in the Function config in `sst.config.ts` like so:

```typescript title="sst.config.ts"
new sst.aws.Function("MyPythonFunction", {
  runtime: "python3.11",
  python: {
    monorepoPath: "apps/python-backend/src",
  },
  handler: "apps/python-backend/src/functions/api.handler",
  url: true,
});
```

## Node.js

A simple node.js function is deployed from `apps/node-backend` in this example. The function depends on shared code from the `packages/node-lib/` directory. Pnpm workspaces are used to manage the shared code and dependencies. A path alias has been set up in the `tsconfig.json` file to allow for easy imports of shared code.

```json title="apps/node-backend/tsconfig.json"
{
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
        "@repo/lib/*": ["../../packages/node-lib/src/*"]
    }
  }
}
```

