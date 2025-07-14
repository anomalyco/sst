# AWS Aurora DSQL Multi-Region Example

This example shows how to create and connect to a multi-region Amazon Aurora DSQL cluster using SST.

## What's in this example

This example creates:

1. **Multi-Region DSQL Cluster** - An active-active Aurora DSQL cluster spanning two AWS regions.
2. **Lambda Function** - A single function linked to the cluster. It demonstrates how to connect to **both** the primary and peer cluster endpoints from one place.

## How to use

1. **Deploy the stack**

   ```bash
   npm install
   sst deploy
   ```

2. **Test the connection**
   The Lambda function, when invoked, will connect to both the primary and peer DSQL cluster endpoints and run a test query against each.

3. **View the results**
   Check the Lambda function logs to see the successful connection and query results from both regions.

## Key features demonstrated

### Multi-Region Cluster Definition

In `sst.config.ts`, a multi-region cluster is defined by providing the `multiRegion` property.

```typescript
// sst.config.ts
const cluster = new sst.aws.Dsql("MyCluster", {
  multiRegion: {
    witnessRegion: "us-west-2",
    peerRegion: "us-east-2",
  },
});
```

### Resource Linking

A single function is linked to the cluster. SST automatically and securely provides the function with all the necessary connection details for both the primary and peer clusters.

```typescript
// sst.config.ts
const fn = new sst.aws.Function("MyFunction", {
  handler: "src/lambda.handler",
  link: [cluster],
});
```

### Connecting to Both Regions

The Lambda function uses the `Resource` object to get the unique connection details for each regional cluster and connects to both.

```typescript
// src/lambda.ts
import { DsqlSigner } from "@aws-sdk/dsql-signer";
import { Resource } from "sst";

async function connectToCluster(region: string, endpoint: string) {
  const signer = new DsqlSigner({ region, hostname: endpoint });
  // ... connect using signer
}

// Connect to the primary cluster
const primaryTime = await connectToCluster(
  Resource.MyCluster.region,
  Resource.MyCluster.publicEndpoint,
);

// Connect to the peer cluster
const peerTime = await connectToCluster(
  Resource.MyCluster.peerRegion,
  Resource.MyCluster.peerPublicEndpoint,
);
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
