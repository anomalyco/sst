/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "playground",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const ret: Record<string, $util.Output<string>> = {};

    const vpc = addVpc();
    const bucket = addBucket();
    const auth = addAuth();
    const oc = addOpenControl();
    //const queue = addQueue();
    //const efs = addEfs();
    //const email = addEmail();
    //const apiv1 = addApiV1();
    //const apiv2 = addApiV2();
    //const apiws = addApiWebsocket();
    const ssrSite = addSsrSite();
    const staticSite = addStaticSite();
    const router = addRouter();
    //const app = addFunction();
    //const cluster = addCluster();
    //const service = addService();
    //const task = addTask();
    //const postgres = addAuroraPostgres();
    //const postgres = addPostgres();
    //const redis = addRedis();
    //const cron = addCron();
    //const topic = addTopic();
    //const bus = addBus();

    return ret;

    function addVpc() {
      const vpc = new sst.aws.Vpc("MyVpc", { nat: "ec2" });
      return vpc;
    }

    function addBucket() {
      const bucket = new sst.aws.Bucket("MyBucket", {
        access: "public",
      });

      //const queue = new sst.aws.Queue("MyQueue");
      //queue.subscribe("functions/bucket/index.handler");

      //const topic = new sst.aws.SnsTopic("MyTopic");
      //topic.subscribe("MyTopicSubscriber", "functions/bucket/index.handler");

      //bucket.notify({
      //  notifications: [
      //    {
      //      name: "LambdaSubscriber",
      //      function: "functions/bucket/index.handler",
      //      filterSuffix: ".json",
      //      events: ["s3:ObjectCreated:*"],
      //    },
      //    {
      //      name: "QueueSubscriber",
      //      queue,
      //      filterSuffix: ".png",
      //      events: ["s3:ObjectCreated:*"],
      //    },
      //    {
      //      name: "TopicSubscriber",
      //      topic,
      //      filterSuffix: ".csv",
      //      events: ["s3:ObjectCreated:*"],
      //    },
      //  ],
      //});
      ret.bucket = bucket.name;
      return bucket;
    }

    function addAuth() {
      const GOOGLE_CLIENT_ID = new sst.Secret("GOOGLE_CLIENT_ID");
      const GOOGLE_CLIENT_SECRET = new sst.Secret("GOOGLE_CLIENT_SECRET");
      const auth = new sst.aws.Auth("MyAuth", {
        domain: "auth.playground.sst.sh",
        issuer: {
          handler: "functions/auth/index.handler",
          link: [GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET],
        },
      });
      return auth;
    }

    function addOpenControl() {
      const oc = new sst.aws.OpenControl("MyOpenControl", {
        server: {
          handler: "functions/open-control/index.handler",
          link: [bucket],
          policies: ["arn:aws:iam::aws:policy/ReadOnlyAccess"],
        },
      });
      return oc;
    }

    function addQueue() {
      const queue = new sst.aws.Queue("MyQueue");
      queue.subscribe("functions/queue/index.subscriber");

      new sst.aws.Function("MyQueuePublisher", {
        handler: "functions/queue/index.publisher",
        link: [queue],
        url: true,
      });
      ret.queue = queue.url;

      return queue;
    }

    function addEfs() {
      const efs = new sst.aws.Efs("MyEfs", { vpc });
      ret.efs = efs.id;
      ret.efsAccessPoint = efs.nodes.accessPoint.id;

      const app = new sst.aws.Function("MyEfsApp", {
        handler: "functions/efs/index.handler",
        volume: { efs },
        url: true,
        vpc,
      });
      ret.efsApp = app.url;

      return efs;
    }

    function addEmail() {
      const topic = new sst.aws.SnsTopic("MyTopic");
      topic.subscribe(
        "MyTopicSubscriber",
        "functions/email/index.notification"
      );

      const email = new sst.aws.Email("MyEmail", {
        sender: "wangfanjie@gmail.com",
        events: [
          {
            name: "notif",
            types: ["delivery"],
            topic: topic.arn,
          },
        ],
      });

      const sender = new sst.aws.Function("MyApi", {
        handler: "functions/email/index.sender",
        link: [email],
        url: true,
      });

      ret.emailSend = sender.url;
      ret.email = email.sender;
      ret.emailConfig = email.configSet;
      return ret;
    }

    function addApiV1() {
      const api = new sst.aws.ApiGatewayV1("MyApiV1");
      api.route("GET /", {
        handler: "functions/apiv2/index.handler",
        link: [bucket],
      });
      api.deploy();
      return api;
    }

    function addApiV2() {
      const api = new sst.aws.ApiGatewayV2("MyApiV2", {
        link: [bucket],
      });
      const authorizer = api.addAuthorizer({
        name: "MyAuthorizer",
        lambda: {
          function: "functions/apiv2/index.authorizer",
          identitySources: [],
        },
      });
      api.route(
        "GET /",
        {
          handler: "functions/apiv2/index.handler",
        },
        {
          auth: { lambda: authorizer.id },
        }
      );
      return api;
    }

    function addApiWebsocket() {
      const api = new sst.aws.ApiGatewayWebSocket("MyApiWebsocket", {});
      const authorizer = api.addAuthorizer("MyAuthorizer", {
        lambda: {
          function: "functions/apiws/index.authorizer",
          identitySources: ["route.request.querystring.Authorization"],
        },
      });
      api.route("$connect", "functions/apiws/index.connect", {
        auth: { lambda: authorizer.id },
      });
      api.route("$disconnect", "functions/apiws/index.disconnect");
      api.route("$default", {
        handler: "functions/apiws/index.catchAll",
        link: [api],
      });
      api.route("sendmessage", "functions/apiws/index.sendMessage");

      return {
        managementEndpoint: api.managementEndpoint,
      };
      return api;
    }

    function addRouter() {
      //const rr7 = new sst.aws.React("MyRouterSite", {
      //  path: "sites/react-router-7-ssr",
      //  cdn: false,
      //});
      //const solid = new sst.aws.SolidStart("MyRouterSolidSite", {
      //  path: "sites/solid-start",
      //  link: [bucket],
      //  cdn: false,
      //});
      //const nuxt = new sst.aws.Nuxt("MyRouterNuxtSite", {
      //  path: "sites/nuxt",
      //  link: [bucket],
      //  cdn: false,
      //});
      //const svelte = new sst.aws.SvelteKit("MyRouterSvelteSite", {
      //  path: "sites/svelte-kit",
      //  link: [bucket],
      //  cdn: false,
      //});
      //const analog = new sst.aws.Analog("MyRouterAnalogSite", {
      //  path: "sites/analog",
      //  link: [bucket],
      //  cdn: false,
      //});
      //const remix = new sst.aws.Remix("MyRouterRemixSite", {
      //  path: "sites/remix",
      //  link: [bucket],
      //  cdn: false,
      //});

      const router = new sst.aws.Router("MyRouter", {
        domain: {
          name: "router.playground.sst.sh",
          aliases: ["*.router.playground.sst.sh"],
        },
        //routes: {
        //  "/*": app.url,
        //},
      });
      //router.route("/api", app.url, {
      //rewrite: {
      //  regex: "^/api/(.*)$",
      //  to: "/$1",
      //},
      //connectionTimeout: "1 second",
      //});
      //router.routeSite("/rr7", rr7);
      //router.routeSite("/astro5", astro5);
      //router.routeSite("/solid", solid);
      //router.routeSite("/nuxt", nuxt);
      //router.routeSite("/svelte", svelte);
      //router.routeSite("/tan", tanstackStart);
      //router.routeSite("/analog", analog);
      //router.routeSite("/remix", remix);
      //router.routeSite("/vite", vite);
      //router.routeSite("/tan", staticSite);

      new sst.aws.Function("MyRouterApp", {
        handler: "functions/router/index.handler",
        url: {
          router: {
            instance: router,
            domain: "api.router.playground.sst.sh/",
          },
        },
      });

      //const vite = new sst.aws.StaticSite("MyRouterVite", {
      //  path: "sites/vite",
      //  route: {
      //    router,
      //    path: "/vite",
      //  },
      //  build: {
      //    command: "npm run build",
      //    output: "dist",
      //  },
      //});

      //new sst.aws.Nextjs("MyRouterNextjs", {
      //  route: {
      //    router,
      //    path: "/next",
      //  },
      //  path: "sites/nextjs",
      //  link: [bucket],
      //  server: {
      //    timeout: "50 seconds",
      //  },
      //});

      new sst.aws.Astro("MyRouterAstro", {
        path: "sites/astro5",
        router: {
          instance: router,
          path: "/astro5",
        },
      });

      //const tanstackStart = new sst.aws.TanStackStart("MyRouterTanStack", {
      //  path: "sites/tanstack-start",
      //  route: {
      //    router,
      //  },
      //});

      return router;
    }

    function addSsrSite() {
      return new sst.aws.Nextjs("MyNextjsSite", {
        domain: "ssr.playground.sst.sh",
        path: "sites/nextjs",
        //path: "sites/astro4",
        //path: "sites/astro5",
        //path: "sites/astro5-static",
        //path: "sites/react-router-7-ssr",
        //path: "sites/react-router-7-csr",
        //path: "sites/tanstack-start",

        // multi-region
        //regions: ["us-east-1", "us-west-1"],
        link: [bucket],
        //assets: {
        //  purge: true,
        //},
      });
    }

    function addStaticSite() {
      new sst.aws.StaticSite("MyStaticSite", {
        domain: "static.playground.sst.sh",
        path: "sites/vite",
        build: {
          command: "npm run build",
          output: "dist",
        },
      });
    }

    function addFunction() {
      const app = new sst.aws.Function("MyApp", {
        handler: "functions/handler-example/index.handler",
        link: [bucket],
        url: true,
      });
      ret.app = app.url;
      return app;
    }

    function addCluster() {
      return new sst.aws.Cluster("MyCluster", {
        vpc,
      });
    }

    function addService() {
      const service = new sst.aws.Service("MyService", {
        cluster,
        loadBalancer: {
          rules: [
            {
              listen: "80/http",
              //container: "app",
            },
            //{ listen: "80/http", container: "web" },
            //{ listen: "8080/http", container: "sidecar" },
          ],
        },
        image: {
          context: "images/web",
        },
        //containers: [
        //  {
        //    name: "web",
        //    image: {
        //      context: "images/web",
        //    },
        //    cpu: "0.125 vCPU",
        //    memory: "0.25 GB",
        //  },
        //  {
        //    name: "sidecar",
        //    image: {
        //      context: "images/sidecar",
        //    },
        //    cpu: "0.125 vCPU",
        //    memory: "0.25 GB",
        //  },
        //],
        link: [bucket],
      });
      ret.service = service.service;
      return service;
    }

    function addTask() {
      const task = new sst.aws.Task("MyTask", {
        cluster,
        image: {
          context: "images/task",
        },
        link: [bucket],
      });

      new sst.aws.Function("MyTaskApp", {
        handler: "functions/task/index.handler",
        url: true,
        vpc,
        link: [task],
      });

      //new sst.aws.Cron("MyTaskCron", {
      //  schedule: "rate(1 minute)",
      //  task,
      //});

      return task;
    }

    function addAuroraPostgres() {
      const postgres = new sst.aws.Aurora("MyPostgres", {
        engine: "postgres",
        vpc,
      });
      new sst.aws.Function("MyPostgresApp", {
        handler: "functions/postgres/index.handler",
        url: true,
        link: [postgres],
        vpc,
      });
      ret.pgHost = postgres.host;
      ret.pgPort = $interpolate`${postgres.port}`;
      ret.pgUsername = postgres.username;
      ret.pgPassword = postgres.password;
      ret.pgDatabase = postgres.database;
      return postgres;
    }

    function addPostgres() {
      const postgres = new sst.aws.Postgres("MyPostgres", {
        vpc,
      });
      new sst.aws.Function("MyPostgresApp", {
        handler: "functions/postgres/index.handler",
        url: true,
        vpc,
        link: [postgres],
      });
      ret.pgHost = postgres.host;
      ret.pgPort = $interpolate`${postgres.port}`;
      ret.pgUsername = postgres.username;
      ret.pgPassword = postgres.password;
      return postgres;
    }

    function addRedis() {
      const redis = new sst.aws.Redis("MyRedis", {
        vpc,
        parameters: {
          "maxmemory-policy": "noeviction",
        },
        //cluster: false,
      });
      ret.redisHost = redis.host;
      const app = new sst.aws.Function("MyRedisApp", {
        handler: "functions/redis/cluster-index.handler",
        //handler: "functions/redis/instance-index.handler",
        url: true,
        vpc,
        link: [redis],
      });
      return redis;
    }

    function addCron() {
      const cron = new sst.aws.Cron("MyCron", {
        schedule: "rate(1 minute)",
        function: {
          handler: "functions/cron/index.handler",
          link: [bucket],
        },
        event: { foo: "bar" },
      });
      ret.cron = cron.nodes.function.name;
      return cron;
    }

    function addTopic() {
      const topic = new sst.aws.SnsTopic("MyTopic");
      topic.subscribe("MyTopicSubscriber", "functions/topic/index.subscriber");

      new sst.aws.Function("MyTopicPublisher", {
        handler: "functions/topic/index.publisher",
        link: [topic],
        url: true,
      });

      return topic;
    }

    function addBus() {
      const bus = new sst.aws.Bus("MyBus");
      bus.subscribe("functions/bus/index.subscriber", {
        pattern: {
          source: ["app.myevent"],
        },
      });
      bus.subscribeQueue("test", queue);

      new sst.aws.Function("MyBusPublisher", {
        handler: "functions/bus/index.publisher",
        link: [bus],
        url: true,
      });

      return bus;
    }
  },
});
