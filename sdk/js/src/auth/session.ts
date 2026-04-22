import { SignJWT, importPKCS8, importSPKI, jwtVerify } from "jose";
import { Resource } from "../resource/index.js";
import process from "node:process";

export type SessionBuilder = ReturnType<typeof createSessionBuilder>;

export function createSessionBuilder<
  SessionTypes extends Record<string, any> = {},
>() {
  type SessionValue =
    | {
        [type in keyof SessionTypes]: {
          type: type;
          properties: SessionTypes[type];
        };
      }[keyof SessionTypes]
    | {
        type: "public";
        properties: {};
      };

  return {
    async verify(token: string): Promise<SessionValue> {
      const auth = Object.values(Resource).find((value) => value.publicKey);
      if (!auth) {
        throw new Error(
          "No auth resource found. Make sure to link the auth resource to this function.",
        );
      }
      const publicKey = auth.publicKey;
      const result = await jwtVerify(
        token,
        await importSPKI(publicKey, "RS512"),
      );
      return result.payload as any;
    },
    async create(session: SessionValue) {
      const privateKey = await importPKCS8(
        // @ts-expect-error
        process.env.AUTH_PRIVATE_KEY || Resource.AUTH_PRIVATE_KEY,
        "RS512",
      );
      const token = await new SignJWT(session)
        .setProtectedHeader({ alg: "RS512" })
        .setExpirationTime("1yr")
        .sign(privateKey);
      return token;
    },
    $type: {} as SessionTypes,
    $typeValues: {} as SessionValue,
  };
}
