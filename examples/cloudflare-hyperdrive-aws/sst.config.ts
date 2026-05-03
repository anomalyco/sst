/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Cloudflare Hyperdrive with AWS Postgres
 *
 * Connect a Cloudflare Worker to an AWS RDS Postgres database through Cloudflare
 * Hyperdrive. Since RDS lives in a private VPC, a Cloudflare tunnel runs on
 * Fargate to expose the database to Hyperdrive. Cloudflare Access locks the
 * tunnel down to Hyperdrive using a service token.
 *
 * The worker uses the `sst.cloudflare.Hyperdrive` binding. The lambda connects
 * directly through the VPC for comparison.
 */
export default $config({
  app(input) {
    return {
      name: "cloudflare-hyperdrive-aws",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
      providers: { cloudflare: "6.13.0", random: "4.18.0" },
    };
  },
  async run() {
    const domain = "sst-dev.org";

    const vpc = new sst.aws.Vpc("Vpc", { nat: "managed" });
    const cluster = new sst.aws.Cluster("Cluster", { vpc });
    const postgres = new sst.aws.Postgres("Postgres", { vpc });

    const zone = cloudflare.getZoneOutput({ filter: { name: domain } });

    const tunnelSecret = new random.RandomString("TunnelSecret", {
      length: 32,
    });

    const tunnel = new cloudflare.ZeroTrustTunnelCloudflared("Tunnel", {
      accountId: sst.cloudflare.DEFAULT_ACCOUNT_ID,
      name: `${$app.name}-${$app.stage}-tunnel`,
      tunnelSecret: tunnelSecret.result.apply((v) =>
        Buffer.from(v).toString("base64"),
      ),
    });

    const record = new cloudflare.DnsRecord("TunnelRecord", {
      name: `hyperdrive-${$app.stage}.${domain}`,
      ttl: 1,
      type: "CNAME",
      zoneId: zone.zoneId,
      content: $interpolate`${tunnel.id}.cfargotunnel.com`,
      proxied: true,
    });

    const tunnelConfig = new cloudflare.ZeroTrustTunnelCloudflaredConfig(
      "TunnelConfig",
      {
        accountId: sst.cloudflare.DEFAULT_ACCOUNT_ID,
        tunnelId: tunnel.id,
        config: {
          ingresses: [
            {
              hostname: record.name,
              service: $interpolate`tcp://${postgres.host}:${postgres.port}`,
            },
            { service: "http_status:404" },
          ],
        },
      },
    );

    const tunnelToken = cloudflare.getZeroTrustTunnelCloudflaredTokenOutput({
      accountId: sst.cloudflare.DEFAULT_ACCOUNT_ID,
      tunnelId: tunnel.id,
    }).token;

    const serviceToken = new cloudflare.ZeroTrustAccessServiceToken(
      "HyperdriveServiceToken",
      {
        name: `${$app.name}-${$app.stage}-hyperdrive-token`,
        accountId: sst.cloudflare.DEFAULT_ACCOUNT_ID,
      },
    );

    new cloudflare.ZeroTrustAccessApplication("HyperdriveAccess", {
      accountId: sst.cloudflare.DEFAULT_ACCOUNT_ID,
      type: "self_hosted",
      name: `${$app.name}-${$app.stage}-hyperdrive`,
      domain: record.name,
      destinations: [{ uri: record.name, type: "public" }],
      appLauncherVisible: false,
      policies: [
        {
          decision: "non_identity",
          includes: [
            { serviceToken: { tokenId: serviceToken.id } },
          ],
          name: `${$app.name}-${$app.stage}-hyperdrive-policy`,
        },
      ],
    });

    const cloudflaredService = new sst.aws.Service(
      "Cloudflared",
      {
        wait: true,
        capacity: "spot",
        cluster,
        containers: [
          {
            name: "cloudflared",
            image: "cloudflare/cloudflared:latest",
            command: ["tunnel", "run"],
            environment: {
              TUNNEL_TOKEN: tunnelToken,
              TUNNEL_METRICS: "0.0.0.0:20241",
            },
            health: {
              command: [
                "CMD",
                "cloudflared",
                "tunnel",
                "--metrics",
                "localhost:20241",
                "ready",
              ],
              startPeriod: "60 seconds",
              timeout: "5 seconds",
              interval: "30 seconds",
              retries: 3,
            },
            dev: {
              autostart: true,
              command: $interpolate`docker run \
                --rm \
                -e TUNNEL_LOGLEVEL=info \
                --network ${$app.name} \
                --name ${$app.name}-${$app.stage}-cloudflared \
                cloudflare/cloudflared:latest \
                tunnel run --token ${tunnelToken}`,
            },
          },
        ],
      },
      // Make sure the tunnel's ingress rules exist before the container starts,
      // otherwise cloudflared serves 503s until it re-polls the config.
      { dependsOn: [tunnelConfig] },
    );

    const hyperdrive = new sst.cloudflare.Hyperdrive(
      "Database",
      {
        origin: {
          host: record.name,
          user: postgres.username,
          password: postgres.password,
          database: postgres.database,
          accessClientId: serviceToken.clientId,
          accessClientSecret: serviceToken.clientSecret,
          scheme: "postgres",
        },
      },
      {
        dependsOn: [
          postgres,
          tunnel,
          record,
          tunnelConfig,
          cloudflaredService,
          cluster,
        ],
      },
    );

    const worker = new sst.cloudflare.Worker("Worker", {
      handler: "./worker.ts",
      link: [hyperdrive],
      url: true,
    });

    const lambda = new sst.aws.Function("Lambda", {
      handler: "lambda.handler",
      link: [postgres],
      vpc,
      url: true,
    });

    return {
      worker: worker.url,
      lambda: lambda.url,
    };
  },
});
