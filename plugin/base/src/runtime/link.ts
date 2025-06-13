import { ComponentResource, runtime } from "@pulumi/pulumi";
import { VisibleError } from "../error";
import { Link, Linkable } from "../linkable";
import { Output, resolve, Input, output } from "../util";
import { app } from "../app";

export namespace link {
  export class Ref extends ComponentResource {
    constructor(target: string, type: string, properties: any, include?: any) {
      super(
        "sst:sst:LinkRef",
        target + "LinkRef",
        {
          properties,
          include,
        },
        {},
      );
      this.registerOutputs({
        target: target,
        include,
        properties: {
          type: type.replaceAll(":", "."),
          ...properties,
        },
      });
    }
  }

  export function reset() {
    const links = new Set<string>();
    // Ensure component names are unique
    runtime.registerStackTransformation((args) => {
      const isLinkable =
        args.type.startsWith("sst:") || Link.wrappedResources.has(args.type);
      if (isLinkable && !args.opts.parent) {
        const lcname = args.name.toLowerCase();

        // "App" is reserved and cannot be used as a component name.
        if (lcname === "app") {
          throw new VisibleError(
            `Component name "${args.name}" is reserved. Please choose a different name for your "${args.type}" component.`,
          );
        }

        // Ensure linkable resources have unique names. This includes all SST components
        // and non-SST components that are linkable.
        if (links.has(lcname)) {
          throw new VisibleError(`Component name ${args.name} is not unique.`);
        }
        links.add(lcname);
      }
      return {
        opts: args.opts,
        props: args.props,
      };
    });

    // Create link refs
    runtime.registerStackTransformation((args) => {
      const resource = args.resource;
      process.nextTick(() => {
        if (link.isLinkable(resource) && !args.opts.parent) {
          try {
            const link = resource.getSSTLink();
            new Ref(args.name, args.type, link.properties, link.include);
          } catch (e) {}
        }
      });
      return {
        opts: args.opts,
        props: args.props,
      };
    });
  }

  export function isLinkable(obj: any): obj is Linkable {
    return "getSSTLink" in obj;
  }

  export function build(links: any[]) {
    return links
      .map((link) => {
        if (!link)
          throw new VisibleError(
            "An undefined link was passed into a `link` array.",
          );
        return link;
      })
      .filter((l) => isLinkable(l))
      .map((l: Linkable) => {
        const link = l.getSSTLink();
        return resolve([l.urn, link]).apply(([urn, link]) => ({
          name: urn.split("::").at(-1)!,
          properties: {
            ...link.properties,
            type: urn.split("::").at(-2),
          },
        }));
      });
  }

  export function getProperties(links?: Input<any[]>) {
    const linkProperties = output(links ?? []).apply((links) =>
      links
        .map((link) => {
          if (!link)
            throw new VisibleError(
              "An undefined link was passed into a `link` array.",
            );
          return link;
        })
        .filter((l) => isLinkable(l))
        .map((l: Linkable) => ({
          urn: l.urn,
          properties: l.getSSTLink().properties,
        })),
    );

    return output(linkProperties).apply((e) =>
      Object.fromEntries(
        e.map(({ urn, properties }) => {
          const name = urn.split("::").at(-1)!;
          const data = {
            ...properties,
            type: urn.split("::").at(-2),
          };
          return [name, data];
        }),
      ),
    );
  }

  export function propertiesToEnv(
    properties: ReturnType<typeof getProperties>,
  ) {
    return output(properties).apply((properties) => {
      const env = Object.fromEntries(
        Object.entries(properties).map(([key, value]) => {
          return [`SST_RESOURCE_${key}`, JSON.stringify(value)];
        }),
      );
      env["SST_RESOURCE_App"] = JSON.stringify({
        name: app.name,
        stage: app.stage,
      });
      return env;
    });
  }

  export function getInclude<T>(
    type: string,
    input?: Input<any[]>,
  ): Output<T[]> {
    if (!input) return output([]);
    return output(input).apply((links) => {
      return links.filter(isLinkable).flatMap((l: Linkable) => {
        const link = l.getSSTLink();
        return (link.include || []).filter((i) => i.type === type) as T[];
      });
    });
  }
}
