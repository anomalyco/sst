import vm from "node:vm";
import { beforeAll, describe, expect, it, vi } from "vitest";
import * as pulumi from "@pulumi/pulumi";

// @ts-ignore
global.$app = { name: "app", stage: "test" };
global.$util = pulumi;

const resources = new Map<string, pulumi.runtime.MockResourceArgs>();

pulumi.runtime.setMocks(
  {
    newResource(args) {
      resources.set(args.name, args);
      return {
        id: `${args.name}_id`,
        state: {
          ...args.inputs,
          arn: `arn:aws:mock:us-east-1:123456789012:${args.name}`,
          bucketRegionalDomainName: `${args.inputs.bucket}.s3.us-east-1.amazonaws.com`,
        },
      };
    },
    call(args) {
      return args.inputs;
    },
  },
  "project",
  "stack",
  false,
);

type RedirectResponse = {
  statusCode: number;
  statusDescription: string;
  headers: { location: { value: string } };
};

describe("HttpsRedirect", () => {
  let code: string;

  beforeAll(async () => {
    const { HttpsRedirect } = await import(
      "../../src/components/aws/https-redirect"
    );
    new HttpsRedirect("Redirect", {
      targetDomain: pulumi.output(Promise.resolve("www.example.com")),
      sourceDomains: pulumi.output(["example.com", "old.example.com"]),
      cert: pulumi.output(
        "arn:aws:acm:us-east-1:123456789012:certificate/test",
      ),
    });

    await vi.waitFor(() => {
      expect(resources.has("RedirectDistribution")).toBe(true);
      expect(resources.has("RedirectBucketPublicAccessBlock")).toBe(true);
      expect(resources.has("RedirectBucketWebsite")).toBe(true);
    });
    const functionCode = resources.get("RedirectCloudfrontFunctionRequest")
      ?.inputs.code;
    if (typeof functionCode !== "string")
      throw new Error("Redirect function code was not registered");
    code = functionCode;
  });

  function redirect(
    uri: string,
    query?: string,
    host = "example.com",
  ): Promise<RedirectResponse> {
    return vm.runInNewContext(`${code}\nhandler(event)`, {
      event: {
        request: {
          uri,
          headers: { host: { value: host } },
          rawQueryString: () => query,
        },
      },
    });
  }

  it.each([
    ["/", undefined, "https://www.example.com/"],
    ["/products/item", undefined, "https://www.example.com/products/item"],
    [
      "/products",
      "utm_source=newsletter&gclid=abc",
      "https://www.example.com/products?utm_source=newsletter&gclid=abc",
    ],
    [
      "/search",
      "tag=one&tag=two&flag&empty=",
      "https://www.example.com/search?tag=one&tag=two&flag&empty=",
    ],
    [
      "/a%2Fb",
      "q=a%2Bb+c&url=%2Fpath%3Fa%3D1",
      "https://www.example.com/a%2Fb?q=a%2Bb+c&url=%2Fpath%3Fa%3D1",
    ],
    ["/products", "", "https://www.example.com/products?"],
  ])(
    "redirects %s with query %s without altering the URL",
    async (uri, query, location) => {
      expect(await redirect(uri, query)).toEqual({
        statusCode: 301,
        statusDescription: "Moved Permanently",
        headers: { location: { value: location } },
      });
    },
  );

  it("redirects every configured source to the resolved target", async () => {
    const response = await redirect("/products", undefined, "old.example.com");
    expect(response.headers.location.value).toBe(
      "https://www.example.com/products",
    );
  });

  it("keeps requests to the CloudFront hostname forbidden", async () => {
    const response = await redirect("/", undefined, "d123.cloudfront.net");
    expect(response.statusCode).toBe(403);
  });

  it("uses a private S3 REST fallback with always-signed OAC", () => {
    const distribution = resources.get("RedirectDistribution");
    expect(distribution?.inputs.aliases).toEqual([
      "example.com",
      "old.example.com",
    ]);
    expect(distribution?.inputs.viewerCertificate).toEqual({
      acmCertificateArn: "arn:aws:acm:us-east-1:123456789012:certificate/test",
      sslSupportMethod: "sni-only",
    });
    expect(distribution?.inputs.origins).toEqual([
      {
        originId: "s3Origin",
        domainName:
          resources.get("RedirectBucketBucket")?.inputs.bucket +
          ".s3.us-east-1.amazonaws.com",
        originAccessControlId:
          "RedirectS3AccessControl.sst.aws.OriginAccessControl_id",
        s3OriginConfig: { originAccessIdentity: "" },
      },
    ]);
    expect(
      resources.get("RedirectBucketPublicAccessBlock")?.inputs,
    ).toMatchObject({
      blockPublicAcls: true,
      blockPublicPolicy: true,
      ignorePublicAcls: true,
      restrictPublicBuckets: true,
    });
  });

  it("retains the old website configuration during rollout", () => {
    expect(
      resources.get("RedirectBucketWebsite")?.inputs.redirectAllRequestsTo,
    ).toEqual({ hostName: "www.example.com", protocol: "https" });
  });
});
