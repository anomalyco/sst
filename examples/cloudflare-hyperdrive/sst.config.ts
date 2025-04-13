/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
	app(input) {
		return {
			name: "hyperdrive",
			removal: input?.stage === "production" ? "retain" : "remove",
			home: "aws",
			providers: { cloudflare: "5.49.1", random: "4.18.0" },
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

		const tunnelSecret = new random.RandomString("TunnelSecret", {
			length: 32,
		});

		const tunnel = new cloudflare.Tunnel("Tunnel", {
			name: `${$app.name}-${$app.stage}-tunnel`,
			secret: tunnelSecret.result.apply((v) =>
				Buffer.from(v).toString("base64"),
			),
			accountId: sst.cloudflare.DEFAULT_ACCOUNT_ID,
		});

		const record = new cloudflare.Record("TunnelRecord", {
			name: $interpolate`hyperdrive.${domain}`,
			zoneId: zone.id,
			type: "CNAME",
			value: $interpolate`${tunnel.id}.cfargotunnel.com`,
			proxied: true,
		});

		new sst.aws.Service("Cloudflared", {
			cluster,
			containers: [
				{
					name: "cloudflared",
					image: "cloudflare/cloudflared:latest",
					command: ["tunnel", "run"],
					environment: {
						TUNNEL_TOKEN: tunnel.tunnelToken,
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
						command: $interpolate`docker run --network host cloudflare/cloudflared:latest tunnel run --token ${tunnel.tunnelToken}`,
					},
				},
			],
		});

		new cloudflare.TunnelConfig("TunnelConfig", {
			accountId: sst.cloudflare.DEFAULT_ACCOUNT_ID,
			tunnelId: tunnel.id,
			config: {
				ingressRules: [
					{
						service: $interpolate`tcp://${postgres.host}:${postgres.port}`,
					},
				],
			},
		});

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
					accessClientId: "dummy",
					accessClientSecret: "dummy",
					scheme: "postgres",
				},
			},
		);

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
