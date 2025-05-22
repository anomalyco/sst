import * as sst from "sst-plugin";
import { Semaphore } from "sst-plugin/internal/semaphore";
import { local } from "@pulumi/command";

const limiter = new Semaphore(
  parseInt(process.env.SST_BUILD_CONCURRENCY_SITE || "1"),
);

export function siteBuilder(
  name: string,
  args: local.CommandArgs,
  opts?: sst.ComponentOptions,
) {
  // Wait for the all args values to be resolved before acquiring the semaphore
  return sst.resolve([args]).apply(async ([args]) => {
    await limiter.acquire(name);

    let waitOn;

    const command = new local.Command(name, args, opts);
    waitOn = command.urn;

    // When running `sst diff`, `local.Command`'s `create` and `update` are not called.
    // So we will also run `local.runOutput` to get the output of the command.
    if (sst.command === "diff") {
      waitOn = local.runOutput(
        {
          command: args.create!,
          dir: args.dir,
          environment: args.environment,
        },
        opts,
      ).stdout;
    }

    return waitOn.apply(() => {
      limiter.release();
      return command;
    });
  });
}
