import fs from "fs";
import path from "path";
import {
  all,
  ComponentResourceOptions,
  interpolate,
  output,
  Output,
  secret,
} from "@pulumi/pulumi";
import { cloudwatch, ecr, ecs, getRegionOutput, iam } from "@pulumi/aws";
import { ImageArgs } from "@pulumi/docker-build";
import { Component, Transform, transform } from "../component.js";
import { Input } from "../input.js";
import { VisibleError } from "../error.js";
import { Link } from "../link.js";
import { toSeconds } from "../duration.js";
import { toNumber } from "../cpu.js";
import { toGBs, toMBs } from "../size.js";
import { RETENTION } from "./logging.js";
import { bootstrap } from "./helpers/bootstrap.js";
import { imageBuilder } from "./helpers/container-builder.js";
import { normalizeContainers } from "./fargate.js";

export const managedGpuManufacturers = ["nvidia"] as const;
export const ManagedGpuAcceleratorName = {
  A100: "a100",
  A10G: "a10g",
  H100: "h100",
  K520: "k520",
  K80: "k80",
  M60: "m60",
  T4: "t4",
  T4G: "t4g",
  V100: "v100",
} as const;

export type ManagedGpuAcceleratorName =
  (typeof ManagedGpuAcceleratorName)[keyof typeof ManagedGpuAcceleratorName];

type ManagedContainers = ReturnType<typeof normalizeContainers>;

type ManagedRoleInput = Input<string>;

type ManagedGpuCount =
  | Input<number>
  | Input<{
      min: Input<number>;
      max?: Input<number>;
    }>;

export interface ManagedServiceCapacityArgs {
  cpu?: Input<{
    min: Input<number>;
    max?: Input<number>;
  }>;
  memory?: Input<{
    min: Input<`${number} GB`>;
    max?: Input<`${number} GB`>;
  }>;
  gpu?: Input<{
    count?: ManagedGpuCount;
    manufacturer?: Input<(typeof managedGpuManufacturers)[number]>;
    /**
     * The NVIDIA GPU model to require.
     *
     * Supported values: `"a100"`, `"a10g"`, `"h100"`, `"k520"`, `"k80"`,
     * `"m60"`, `"t4"`, `"t4g"`, and `"v100"`.
     */
    name?: Input<ManagedGpuAcceleratorName | Input<ManagedGpuAcceleratorName>[]>;
  }>;
  infrastructureRole: ManagedRoleInput;
  instanceRole?: ManagedRoleInput;
  instanceProfile?: Input<string>;
  storage?: Input<`${number} GB`>;
}

type ServiceSizingArgs = {
  cpu?: Input<`${number} vCPU`>;
  memory?: Input<`${number} GB`>;
};

type ManagedTaskDefinitionArgs = {
  cluster: {
    nodes: {
      cluster: {
        name: Output<string>;
      };
    };
  };
  link?: any;
  transform?: {
    image?: Transform<ImageArgs>;
    taskDefinition?: Transform<ecs.TaskDefinitionArgs>;
    logGroup?: Transform<cloudwatch.LogGroupArgs>;
  };
};

type ManagedCapacityProviderArgs = {
  transform?: {
    capacityProvider?: Transform<ecs.CapacityProviderArgs>;
    instanceProfile?: Transform<iam.InstanceProfileArgs>;
  };
};

type ManagedVpcArgs = {
  containerSubnets: Input<Input<string>[]>;
  securityGroups: Input<Input<string>[]>;
};

type NormalizedManagedCapacity = {
  taskCpu: string;
  taskMemory: string;
  hostCpu: {
    min: number;
    max?: number;
  };
  hostMemory: {
    min: number;
    max?: number;
  };
  hostStorage?: number;
  gpu?: {
    count: {
      min: number;
      max?: number;
    };
    manufacturer: (typeof managedGpuManufacturers)[number];
    names?: ManagedGpuAcceleratorName[];
  };
};

export function normalizeManagedCapacity(
  name: string,
  args: ManagedServiceCapacityArgs,
  serviceSizing: ServiceSizingArgs,
) {
  return all([serviceSizing.cpu, serviceSizing.memory]).apply(
    ([serviceCpu, serviceMemory]) => {
      const managed = args as ManagedServiceCapacityArgs & {
        infrastructureRole?: string;
        instanceRole?: string;
        instanceProfile?: string;
        storage?: `${number} GB`;
        cpu?: { min: number; max?: number };
        memory?: { min: `${number} GB`; max?: `${number} GB` };
        gpu?: {
          count?: number | { min: number; max?: number };
          manufacturer?: (typeof managedGpuManufacturers)[number];
          name?: ManagedGpuAcceleratorName | ManagedGpuAcceleratorName[];
        };
      };

      if (!managed.infrastructureRole) {
        throw new VisibleError(
          `Missing \"capacity.managed.infrastructureRole\" for the \"${name}\" Service.`,
        );
      }

      if (!managed.instanceRole && !managed.instanceProfile) {
        throw new VisibleError(
          `You must provide either \"capacity.managed.instanceRole\" or \"capacity.managed.instanceProfile\" for the \"${name}\" Service.`,
        );
      }

      if (managed.instanceRole && managed.instanceProfile) {
        throw new VisibleError(
          `Do not provide both \"capacity.managed.instanceRole\" and \"capacity.managed.instanceProfile\" for the \"${name}\" Service.`,
        );
      }

      const hostCpu = normalizeHostCpu(managed.cpu, serviceCpu);
      const hostMemory = normalizeHostMemory(managed.memory, serviceMemory);
      const gpu = normalizeGpu(managed.gpu);
      const hostStorage = normalizeStorage(managed.storage);

      return {
        taskCpu: serviceCpu
          ? toNumber(serviceCpu).toString()
          : Math.round(hostCpu.min * 1024).toString(),
        taskMemory: serviceMemory
          ? toMBs(serviceMemory).toString()
          : hostMemory.min.toString(),
        hostCpu,
        hostMemory,
        hostStorage,
        gpu,
      } satisfies NormalizedManagedCapacity;
    },
  );

  function normalizeHostCpu(
    cpu: { min: number; max?: number } | undefined,
    fallback?: `${number} vCPU`,
  ) {
    if (cpu) {
      validateRange("capacity.managed.cpu", cpu.min, cpu.max);
      return { min: cpu.min, max: cpu.max };
    }
    if (fallback) {
      const min = parseFloat(fallback.split(" ")[0]);
      return { min, max: min };
    }
    throw new VisibleError(
      `You must provide either \"capacity.managed.cpu\" or top-level \"cpu\" for managed instances.`,
    );
  }

  function normalizeHostMemory(
    memory: { min: `${number} GB`; max?: `${number} GB` } | undefined,
    fallback?: `${number} GB`,
  ) {
    if (memory) {
      const min = toMBs(memory.min);
      const max = memory.max ? toMBs(memory.max) : undefined;
      validateRange("capacity.managed.memory", min, max);
      return { min, max };
    }
    if (fallback) {
      const min = toMBs(fallback);
      return { min, max: min };
    }
    throw new VisibleError(
      `You must provide either \"capacity.managed.memory\" or top-level \"memory\" for managed instances.`,
    );
  }

  function normalizeGpu(
    gpu:
      | {
          count?: number | { min: number; max?: number };
          manufacturer?: (typeof managedGpuManufacturers)[number];
          name?: ManagedGpuAcceleratorName | ManagedGpuAcceleratorName[];
        }
      | undefined,
  ) {
    if (!gpu) return undefined;
    const manufacturer = gpu.manufacturer ?? "nvidia";
    if (!managedGpuManufacturers.includes(manufacturer)) {
      throw new VisibleError(
        `Unsupported GPU manufacturer \"${manufacturer}\". The supported values are ${managedGpuManufacturers.join(", ")}.`,
      );
    }

    const count =
      typeof gpu.count === "number"
        ? { min: gpu.count, max: gpu.count }
        : gpu.count
          ? { min: gpu.count.min, max: gpu.count.max }
          : { min: 1, max: 1 };

    validateRange("capacity.managed.gpu.count", count.min, count.max);

    return {
      count,
      manufacturer,
      names: normalizeGpuNames(gpu.name),
    };
  }

  function normalizeGpuNames(
    name: ManagedGpuAcceleratorName | ManagedGpuAcceleratorName[] | undefined,
  ) {
    if (!name) return undefined;
    const names = Array.isArray(name) ? name : [name];
    const supported = Object.values(ManagedGpuAcceleratorName);
    const invalid = names.filter((name) => !supported.includes(name));
    if (invalid.length > 0) {
      throw new VisibleError(
        `Unsupported GPU accelerator name ${invalid.map((name) => `"${name}"`).join(", ")}. The supported NVIDIA values are ${supported
          .map((name) => `"${name}"`)
          .join(", ")}.`,
      );
    }
    return names;
  }

  function normalizeStorage(storage?: `${number} GB`) {
    if (!storage) return undefined;
    const value = toGBs(storage);
    if (value <= 0) {
      throw new VisibleError(
        `Invalid \"capacity.managed.storage\" value \"${storage}\". It must be greater than 0 GB.`,
      );
    }
    return value;
  }

  function validateRange(label: string, min: number, max?: number) {
    if (min <= 0) {
      throw new VisibleError(`\"${label}.min\" must be greater than 0.`);
    }
    if (max !== undefined && max < min) {
      throw new VisibleError(
        `\"${label}.max\" must be greater than or equal to \"${label}.min\".`,
      );
    }
  }
}

export function createManagedCapacityProvider(
  name: string,
  args: ManagedCapacityProviderArgs & {
    capacity: Input<ManagedServiceCapacityArgs>;
  },
  opts: ComponentResourceOptions,
  parent: Component,
  clusterName: Output<string>,
  vpc: ManagedVpcArgs,
  normalized: Output<NormalizedManagedCapacity>,
) {
  const infrastructureRole = iam.Role.get(
    `${name}ManagedInfrastructureRole`,
    output(args.capacity).apply((v) => v.infrastructureRole),
    {},
    { parent },
  );

  const instanceProfileArn = output(args.capacity).apply((v) => {
    if (v.instanceProfile) return v.instanceProfile;

    return new iam.InstanceProfile(
      ...transform(
        args.transform?.instanceProfile,
        `${name}ManagedInstanceProfile`,
        {
          role: output(v.instanceRole!).apply(extractRoleName),
        },
        { parent },
      ),
    ).arn;
  });

  return new ecs.CapacityProvider(
    ...transform(
      args.transform?.capacityProvider,
      `${name}ManagedCapacityProvider`,
      {
        cluster: clusterName,
        managedInstancesProvider: all([
          normalized,
          infrastructureRole.arn,
          instanceProfileArn,
          vpc.containerSubnets,
          vpc.securityGroups,
        ]).apply(
          ([normalized, infrastructureRoleArn, instanceProfileArn, subnets, securityGroups]) => ({
            infrastructureRoleArn,
            propagateTags: "CAPACITY_PROVIDER",
            instanceLaunchTemplate: {
              ec2InstanceProfileArn: instanceProfileArn,
              networkConfiguration: {
                subnets,
                securityGroups,
              },
              ...(normalized.hostStorage
                ? {
                    storageConfiguration: {
                      storageSizeGib: normalized.hostStorage,
                    },
                  }
                : {}),
              instanceRequirements: {
                vcpuCount: {
                  min: normalized.hostCpu.min,
                  max: normalized.hostCpu.max,
                },
                memoryMib: {
                  min: normalized.hostMemory.min,
                  max: normalized.hostMemory.max,
                },
                instanceGenerations: ["current"],
                ...(normalized.gpu
                  ? {
                      acceleratorTypes: ["gpu"],
                      acceleratorCount: {
                        min: normalized.gpu.count.min,
                        max: normalized.gpu.count.max,
                      },
                      acceleratorManufacturers: [normalized.gpu.manufacturer],
                      ...(normalized.gpu.names
                        ? {
                            acceleratorNames: normalized.gpu.names,
                          }
                        : {}),
                    }
                  : {}),
              },
            },
          }),
        ),
      },
      { parent },
    ),
  );
}

export function createManagedTaskDefinition(
  name: string,
  args: ManagedTaskDefinitionArgs,
  opts: ComponentResourceOptions,
  parent: Component,
  containers: ManagedContainers,
  architecture: Output<"x86_64" | "arm64">,
  taskRole: iam.Role,
  executionRole: iam.Role,
  normalized: Output<NormalizedManagedCapacity>,
) {
  const clusterName = args.cluster.nodes.cluster.name;
  const region = getRegionOutput({}, opts).region;
  const bootstrapData = region.apply((region) => bootstrap.forRegion(region));
  const linkEnvs = Link.propertiesToEnv(Link.getProperties(args.link));

  const containerDefinitions = all([containers, normalized]).apply(
    ([containers, normalized]) => {
      if (normalized.gpu && containers.length > 1) {
        throw new VisibleError(
          `GPU support currently requires a single container when using managed instances.`,
        );
      }

      return containers.map((container) => ({
        name: container.name,
        image: (() => {
          if (typeof container.image === "string") return output(container.image);

          const containerImage = container.image;
          const contextPath = path.join($cli.paths.root, container.image.context);
          const dockerfile = container.image.dockerfile ?? "Dockerfile";
          const dockerfilePath = path.join(contextPath, dockerfile);
          const dockerIgnorePath = fs.existsSync(
            path.join(contextPath, `${dockerfile}.dockerignore`),
          )
            ? path.join(contextPath, `${dockerfile}.dockerignore`)
            : path.join(contextPath, ".dockerignore");

          const lines = fs.existsSync(dockerIgnorePath)
            ? fs.readFileSync(dockerIgnorePath).toString().split("\n")
            : [];
          if (!lines.find((line) => line === ".sst")) {
            fs.writeFileSync(
              dockerIgnorePath,
              [...lines, "", "# sst", ".sst"].join("\n"),
            );
          }

          const image = imageBuilder(
            ...transform(
              args.transform?.image,
              `${name}Image${container.name}`,
              {
                context: { location: contextPath },
                dockerfile: { location: dockerfilePath },
                buildArgs: containerImage.args,
                secrets: all([linkEnvs, containerImage.secrets ?? {}]).apply(
                  ([link, secrets]) => ({ ...link, ...secrets }),
                ),
                target: container.image.target,
                platforms: [container.image.platform],
                tags: [container.name, ...(container.image.tags ?? [])].map(
                  (tag) => interpolate`${bootstrapData.assetEcrUrl}:${tag}`,
                ),
                registries: [
                  ecr
                    .getAuthorizationTokenOutput(
                      {
                        registryId: bootstrapData.assetEcrRegistryId,
                      },
                      { parent },
                    )
                    .apply((authToken) => ({
                      address: authToken.proxyEndpoint,
                      password: secret(authToken.password),
                      username: authToken.userName,
                    })),
                ],
                ...(container.image.cache !== false
                  ? {
                      cacheFrom: [
                        {
                          registry: {
                            ref: interpolate`${bootstrapData.assetEcrUrl}:${container.name}-cache`,
                          },
                        },
                      ],
                      cacheTo: [
                        {
                          registry: {
                            ref: interpolate`${bootstrapData.assetEcrUrl}:${container.name}-cache`,
                            imageManifest: true,
                            ociMediaTypes: true,
                            mode: "max",
                          },
                        },
                      ],
                    }
                  : {}),
                push: true,
              },
              { parent },
            ),
          );

          return interpolate`${bootstrapData.assetEcrUrl}@${image.digest}`;
        })(),
        cpu: container.cpu ? toNumber(container.cpu) : undefined,
        memory: container.memory ? toMBs(container.memory) : undefined,
        command: container.command,
        entrypoint: container.entrypoint,
        healthCheck: container.health && {
          command: container.health.command,
          startPeriod: toSeconds(container.health.startPeriod ?? "0 seconds"),
          timeout: toSeconds(container.health.timeout ?? "5 seconds"),
          interval: toSeconds(container.health.interval ?? "30 seconds"),
          retries: container.health.retries ?? 3,
        },
        pseudoTerminal: true,
        portMappings: [{ containerPortRange: "1-65535" }],
        logConfiguration: {
          logDriver: "awslogs",
          options: {
            "awslogs-group": (() => {
              return new cloudwatch.LogGroup(
                ...transform(
                  args.transform?.logGroup,
                  `${name}LogGroup${container.name}`,
                  {
                    name: container.logging.name,
                    retentionInDays: RETENTION[container.logging.retention],
                  },
                  { parent, ignoreChanges: ["name"] },
                ),
              );
            })().name,
            "awslogs-region": region,
            "awslogs-stream-prefix": "/service",
          },
        },
        environment: linkEnvs.apply((linkEnvs) =>
          Object.entries({
            ...container.environment,
            ...linkEnvs,
          }).map(([name, value]) => ({ name, value })),
        ),
        environmentFiles: container.environmentFiles?.map((file) => ({
          type: "s3",
          value: file,
        })),
        linuxParameters: {
          initProcessEnabled: true,
        },
        mountPoints: container.volumes?.map((volume) => ({
          sourceVolume: volume.efs.accessPoint,
          containerPath: volume.path,
        })),
        secrets: Object.entries(container.ssm ?? {}).map(([name, valueFrom]) => ({
          name,
          valueFrom,
        })),
        resourceRequirements: normalized.gpu
          ? [{ type: "GPU", value: normalized.gpu.count.min.toString() }]
          : undefined,
      }));
    },
  );

  return output(
    new ecs.TaskDefinition(
      ...transform(
        args.transform?.taskDefinition,
        `${name}Task`,
        {
          family: interpolate`${clusterName}-${name}`,
          trackLatest: true,
          cpu: normalized.apply((v) => v.taskCpu),
          memory: normalized.apply((v) => v.taskMemory),
          networkMode: "awsvpc",
          requiresCompatibilities: ["MANAGED_INSTANCES"],
          runtimePlatform: {
            cpuArchitecture: architecture.apply((v) => v.toUpperCase()),
            operatingSystemFamily: "LINUX",
          },
          executionRoleArn: executionRole.arn,
          taskRoleArn: taskRole.arn,
          volumes: output(containers).apply((containers) => {
            const uniqueAccessPoints: Set<string> = new Set();
            return containers.flatMap((container) =>
              (container.volumes ?? []).flatMap((volume) => {
                if (uniqueAccessPoints.has(volume.efs.accessPoint)) return [];
                uniqueAccessPoints.add(volume.efs.accessPoint);
                return {
                  name: volume.efs.accessPoint,
                  efsVolumeConfiguration: {
                    fileSystemId: volume.efs.fileSystem,
                    transitEncryption: "ENABLED",
                    authorizationConfig: {
                      accessPointId: volume.efs.accessPoint,
                    },
                  },
                };
              }),
            );
          }),
          containerDefinitions: $jsonStringify(containerDefinitions),
        },
        { parent },
      ),
    ),
  );
}

export function isManagedCapacityInput(
  capacity: unknown,
): capacity is {
  managed: Input<ManagedServiceCapacityArgs>;
} {
  return typeof capacity === "object" && capacity !== null && "managed" in capacity;
}

function extractRoleName(role: string) {
  return role.split("/").pop()!;
}
