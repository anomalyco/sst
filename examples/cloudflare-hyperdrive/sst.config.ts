/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
	app(input) {
		return {
			name: "hyperdrive",
			removal: input?.stage === "production" ? "retain" : "remove",
			home: "aws",
			providers: { cloudflare: "6.1.1", random: "4.18.0" },
		};
	},
	async run() {
		const vpc = new sst.aws.Vpc("Vpc", {
			nat: "managed",
		});

		const cluster = new sst.aws.Cluster("Cluster", { vpc });

		const postgres = new sst.aws.Postgres("Postgres", { vpc });

		const domain = "sst.cheap";

		const zone = cloudflare.getZoneOutput({ name: domain });

	const tunnelSecret = new random.RandomString("TunnelSecret", {length: 32});

  const tunnel = new cloudflare.ZeroTrustTunnelCloudflared("Tunnel", {
    accountId: sst.cloudflare.DEFAULT_ACCOUNT_ID,
    name: `${$app.name}-${$app.stage}-tunnel`,
    tunnelSecret: tunnelSecret.result.apply((v) =>
      Buffer.from(v).toString("base64"),
    ),
  });

  const record = new cloudflare.DnsRecord("TunnelRecord", {
    name: $interpolate`hyperdrive-${$app.stage}.${tld}`,
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
            service: $interpolate`tcp://${$app.name}-${$app.stage}-postgres:${postgres.port}`,
          },
          {
            service: $interpolate`tcp://${postgres.host}:${postgres.port}`,
          },
        ],
      },
    },
  );

  const tunnelToken = cloudflare
    .getZeroTrustTunnelCloudflaredToken({
      accountId: sst.cloudflare.DEFAULT_ACCOUNT_ID,
      tunnelId: tunnel.id,
    })
    .then((result) => result.token);

  const hyperdriveZeroTrustAccessServiceToken =
    new cloudflare.ZeroTrustAccessServiceToken(
      "HyperdriveZeroTrustAccessServiceToken",
      {
        name: `${$app.name}-${$app.stage}-zero-trust-access-service-token`,
        accountId: sst.cloudflare.DEFAULT_ACCOUNT_ID,
        duration: "forever",
      },
    );

  new cloudflare.ZeroTrustAccessApplication("ZeroTrustAccessApplication", {
    accountId: sst.cloudflare.DEFAULT_ACCOUNT_ID,
    type: "self_hosted",
    name: `${$app.name}-${$app.stage}-zero-trust-access-application`,
    domain: record.name,
    destinations: [{ uri: record.name, type: "public" }],
    appLauncherVisible: false,
    policies: [
      {
        decision: "non_identity",
        includes: [
          {
            serviceToken: { tokenId: hyperdriveZeroTrustAccessServiceToken.id },
          },
        ],
        name: `${$app.name}-${$app.stage}-zero-trust-access-policy`,
      },
    ],
  });

  new cloudflare.ZeroTrustAccessPolicy("HyperdriveZeroTrustAccessPolicy", {
    accountId: sst.cloudflare.DEFAULT_ACCOUNT_ID,
    name: `${$app.name}-${$app.stage}-zero-trust-access-policy`,
    decision: "non_identity",
    includes: [
      {
        serviceToken: {
          tokenId: hyperdriveZeroTrustAccessServiceToken.id,
        },
      },
    ],
  });

  const cloudflaredService = new sst.aws.Service("Cloudflared", {
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
  });

  const accessClientId = cloudflare
    .getZeroTrustAccessServiceToken({
      accountId: sst.cloudflare.DEFAULT_ACCOUNT_ID,
      serviceTokenId: hyperdriveZeroTrustAccessServiceToken.id,
    })
    .then((result) => result.clientId);

  const hyperdriveConfig = new cloudflare.HyperdriveConfig(
    "HyperdriveConfig",
    {
      name: `${$app.name}-${$app.stage}-config`,
      accountId: sst.cloudflare.DEFAULT_ACCOUNT_ID,
      origin: {
        host: record.name,
        user: postgres.username,
        password: postgres.password,
        database: postgres.database,
        accessClientId: accessClientId,
        accessClientSecret: hyperdriveZeroTrustAccessServiceToken.clientSecret,
        scheme: "postgres",
      },
    },
    {
      // wait on everthing to be ready, this does not work. Stop and then restart SST to fix.
      dependsOn: [
        postgres,
        tunnel,
        record,
        tunnelConfig,
        cloudflaredService,
        cluster,
      ],
    });
	
		const worker = new sst.cloudflare.Worker("Worker", {
			handler: "worker.ts",
			url: true,
			transform: {
				worker: {
					placements: [
						{
							mode: "smart",
						},
					],
					hyperdriveConfigBindings: [
						{
							binding: "HYPERDRIVE",
							id: $interpolate`${hyperdriveConfig.resourceId}`,
						},
					],
				},
			},
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
