import { Cron } from "../../src/components/cloudflare/cron";
import { Queue } from "../../src/components/cloudflare/queue";
import { Worker } from "../../src/components/cloudflare/worker";

// This file is a compile-time fixture covered by `bun run typecheck:platform`.
declare const queue: Queue;
declare const worker: Worker;
declare const bucket: unknown;

queue.subscribe(worker);
queue.subscribe("consumer.ts");
queue.subscribe({ handler: "consumer.ts", link: [bucket] });

new Cron("CronWorker", { worker, schedules: ["*/5 * * * *"] });
new Cron("CronJobWorker", { job: worker, schedules: ["*/5 * * * *"] });
new Cron("CronString", { worker: "cron.ts", schedules: ["* * * * *"] });
new Cron("CronArgs", {
  worker: { handler: "cron.ts", link: [bucket] },
  schedules: ["* * * * *"],
});
