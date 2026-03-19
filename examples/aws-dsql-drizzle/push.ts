import { DsqlSigner } from "@aws-sdk/dsql-signer";
import { Resource } from "sst";
import { execSync } from "child_process";

const signer = new DsqlSigner({
  region: Resource.MyCluster.region,
  hostname: Resource.MyCluster.endpoint,
});

const token = await signer.getDbConnectAdminAuthToken();

execSync("bunx drizzle-kit push", {
  stdio: "inherit",
  env: {
    ...process.env,
    DSQL_TOKEN: token,
  },
});
