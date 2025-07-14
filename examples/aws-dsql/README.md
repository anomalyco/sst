# AWS Aurora DSQL Example

This example shows how to create and connect to a single-region Amazon Aurora DSQL cluster using SST. For a multi-region example, see the [`aws-dsql-multiregion`](../aws-dsql-multiregion/) directory.

## What is Aurora DSQL?

Aurora DSQL is a serverless, distributed relational database optimized for transactional workloads. It provides:

- **PostgreSQL compatibility** - Use standard PostgreSQL drivers and tools
- **Serverless scaling** - Automatically scales compute, I/O, and storage
- **Active-active multi-region** - 99.99% single-region, 99.999% multi-region availability
- **Strong consistency** - ACID guarantees across all regions
- **IAM authentication** - No traditional passwords, uses IAM for auth tokens

## What's in this example

This example creates:

1. **DSQL Cluster** - A single-region Aurora DSQL cluster
2. **Lambda Function** - Connected to the cluster that demonstrates how to:
   - Generate IAM auth tokens
   - Connect using PostgreSQL drivers
   - Execute queries

## How to use

1. **Deploy the stack**

   ```bash
   npm install
   sst deploy
   ```

2. **Test the connection**
   The Lambda function when invoked will connect to the DSQL cluster and run a test query.

3. **View the results**
   Check the Lambda function logs to see the successful connection and query results.

## Key features demonstrated

### IAM Authentication

The Lambda function uses the `@aws-sdk/dsql-signer` package to generate a temporary authentication token.

```typescript
import { DsqlSigner } from "@aws-sdk/dsql-signer";
import { Resource } from "sst";

// Use the Resource object to get the region and endpoint
const signer = new DsqlSigner({
  region: Resource.MyCluster.region,
  hostname: Resource.MyCluster.publicEndpoint,
});

// Generate the token
const token = await signer.getDbConnectAdminAuthToken();
```

### PostgreSQL Connection

```typescript
// Connect using standard PostgreSQL driver
const client = new Client({
  host: Resource.MyCluster.publicEndpoint,
  port: 5432,
  database: "postgres",
  user: "admin",
  password: token,
  ssl: true,
});
```

### Resource Linking

SST's `link` feature securely provides the function with the connection details for the cluster, removing the need for environment variables.

```typescript
// Link cluster to function in sst.config.ts
const fn = new sst.aws.Function("MyFunction", {
  handler: "src/lambda.handler",
  link: [cluster],
});
```

## Cost considerations

Aurora DSQL uses serverless, consumption-based pricing:

- Pay only for compute, I/O, and storage consumed
- No instance charges
- Automatically scales with usage
- Idle clusters incur no compute charges

## Clean up

```bash
sst remove
```

This will remove all resources created by this example.
