import { s3 } from "@pulumi/aws";
import * as sst from "sst-plugin";
import { Transform } from "sst-plugin/internal/transform";
import { FunctionArgs, Function } from "./function.js";
import { PrivateKey } from "@pulumi/tls";

export interface AuthArgs {
  authenticator: FunctionArgs;
  transform?: {
    bucketPolicy?: Transform<s3.BucketPolicyArgs>;
  };
}

export class Auth extends sst.Component implements sst.Linkable {
  private readonly _key: PrivateKey;
  private readonly _authenticator: sst.Output<Function>;

  constructor(name: string, args: AuthArgs, opts?: sst.ComponentOptions) {
    super(__pulumiType, name, args, opts);

    this._key = new PrivateKey(`${name}Keypair`, {
      algorithm: "RSA",
    });

    this._authenticator = sst.output(args.authenticator).apply((args) => {
      return new Function(`${name}Authenticator`, {
        url: true,
        ...args,
        environment: {
          ...args.environment,
          AUTH_PRIVATE_KEY: sst.secret(this.key.privateKeyPemPkcs8),
          AUTH_PUBLIC_KEY: sst.secret(this.key.publicKeyPem),
        },
        _skipHint: true,
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
        publicKey: sst.secret(this.key.publicKeyPem),
      },
    };
  }
}

const __pulumiType = "sst:aws:Auth";
// @ts-expect-error
Auth.__pulumiType = __pulumiType;
