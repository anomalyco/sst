/**
 * AWS-managed CloudFront policies and configuration values.
 */
export const cloudfront = {
  cachePolicy: {
    /**
     * Disables caching. This policy is useful for dynamic content and APIs.
     *
     * @see https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/using-managed-cache-policies.html
     */
    cachingDisabled: "4135ea2d-6df8-44a3-9df3-4b5a84be39ad",
    /**
     * Optimizes cache efficiency by minimizing values included in the cache key.
     *
     * @see https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/using-managed-cache-policies.html
     */
    cachingOptimized: "658327ea-f89d-4fab-a63d-7e88639e58f6",
  },
} as const;
