# <!--name--> AWS Serverless Redis

An example of deploying a Redis serverless cache using [Amazon ElastiCache Serverless](https://aws.amazon.com/elasticache/serverless/).

This example demonstrates the new `serverless` option in SST's Redis component, which automatically scales based on usage and offers pay-per-use pricing.

## Key Features

- **Serverless Redis**: No instance management required
- **Automatic Scaling**: Scales based on usage
- **Pay-per-use**: Only pay for what you consume
- **Simplified VPC**: No NAT Gateway required for serverless
- **Same Client Interface**: Uses the same Redis client as traditional clusters

## Get started

1. **Clone and deploy**

   ```bash
   git clone https://github.com/sst/sst
   cd sst/examples/aws-redis-serverless
   npm install
   sst deploy
   ```

2. **Test the Redis connection**

   Once deployed, you can invoke the function to test the Redis connection:

   ```bash
   sst invoke MyApp
   ```

## Usage

The serverless Redis is created with default limits:

```ts title="sst.config.ts" {4-7}
const redis = new sst.aws.Redis("MyRedis", {
  vpc,
  serverless: {
    dataStorage: { maximum: 10, unit: "GB" },
    ecpuPerSeconds: { maximum: 5000 }
  }
});
```

You can also enable serverless mode with defaults:

```ts title="sst.config.ts" {3}
const redis = new sst.aws.Redis("MyRedis", {
  vpc,
  serverless: true
});
```

The client code remains exactly the same:

```ts title="index.ts" {4-6,11-12}
import { Cluster } from "ioredis";
import { Resource } from "sst";

const client = new Cluster([{
  host: Resource.MyRedis.host,
  port: Resource.MyRedis.port,
}], {
  redisOptions: {
    tls: { checkServerIdentity: () => undefined },
    username: Resource.MyRedis.username,
    password: Resource.MyRedis.password
  }
});
```

## Architecture

```
┌──────────────┐    ┌─────────────────────┐
│    Lambda    │───▶│  ElastiCache       │
│   Function   │    │  Serverless Redis  │
└──────────────┘    └─────────────────────┘
        │                      │
        └──────────────────────┘
              VPC Network
```

## Cost

Serverless Redis pricing is based on:
- **Data Storage**: Per GB stored
- **ElastiCache Processing Units (ECPUs)**: Per second of processing

Example cost for light usage:
- Storage: 1 GB = ~$0.125/month  
- ECPUs: 1000 ECPU/second = ~$0.0034/hour

This is significantly cheaper than traditional instances for variable workloads.

## Differences from Traditional Redis

| Traditional Redis | Serverless Redis |
|-------------------|------------------|
| Fixed instance costs | Pay-per-use pricing |
| Manual scaling | Automatic scaling |
| Requires NAT Gateway | Simplified networking |
| Cluster/non-cluster modes | Simplified configuration |
| Instance-based limits | Usage-based limits |
