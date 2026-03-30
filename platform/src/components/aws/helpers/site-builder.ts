import { all, output, CustomResourceOptions } from "@pulumi/pulumi";
import { Semaphore } from "../../../util/semaphore";
import { local } from "@pulumi/command";

const limiter = new Semaphore(
  parseInt(process.env.SST_BUILD_CONCURRENCY_SITE || "1"),
);

export function siteBuilder(
  name: string,
  args: local.CommandArgs,
  opts?: CustomResourceOptions,
) {
  const ready = output(limiter.acquire(name));
  // Wait for the all args values to be resolved before acquiring the semaphore
  return all([args, ready]).apply(([args]) => {

    let waitOn;

    const command = new local.Command(name, args, opts);
    waitOn = command.urn;

    // When running `sst diff` or `sst refresh`, `local.Command`'s `create` and `update` are not called.
    // So we also run `local.runOutput` to get the output of the command.
    if ($cli.command === "diff" || $cli.command === "refresh") {
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
