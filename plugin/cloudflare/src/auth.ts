import * as sst from "sst-plugin";
import { PrivateKey } from "@pulumi/tls";
import { WorkerArgs, Worker } from "./worker.js";
import { CloudflareComponent } from "./component.js";

export interface AuthArgs {
  authenticator: WorkerArgs;
}

export class Auth extends CloudflareComponent implements sst.Linkable {
  private readonly _key: PrivateKey;
  private readonly _authenticator: sst.Output<Worker>;

  constructor(name: string, args: AuthArgs, opts?: sst.ComponentOptions) {
    super(__pulumiType, name, args, opts);

    this._key = new PrivateKey(`${name}Keypair`, {
      algorithm: "RSA",
    });

    this._authenticator = sst.output(args.authenticator).apply((args) => {
      return new Worker(`${name}Authenticator`, {
        ...args,
        url: true,
        environment: {
          ...args.environment,
          AUTH_PRIVATE_KEY: sst.secret(this.key.privateKeyPemPkcs8),
          AUTH_PUBLIC_KEY: sst.secret(this.key.publicKeyPem),
        },
      });
    });
  }

  public get key() {
    return this._key;
  }

  public get authenticator() {
    return this._authenticator;
  }

  public get url() {
    return this._authenticator.url!;
  }

  /** @internal */
  public getSSTLink() {
    return {
      properties: {
        url: this._authenticator.url,
        publicKey: sst.secret(this.key.publicKeyPem),
      },
    };
  }
}

const __pulumiType = "sst:cloudflare:Auth";
// @ts-expect-error
Auth.__pulumiType = __pulumiType;
