/**
 * Simulates a browser multipart upload via presigned URLs.
 * Verifies that the ETag header is accessible through CORS.
 *
 * Usage: node test-multipart.mjs <bucket-name>
 */
import {
  S3Client,
  CreateMultipartUploadCommand,
  CompleteMultipartUploadCommand,
  AbortMultipartUploadCommand,
} from "@aws-sdk/client-s3";
import { getSignedUrl } from "@aws-sdk/s3-request-presigner";
import { UploadPartCommand } from "@aws-sdk/client-s3";

const BUCKET = process.argv[2];
if (!BUCKET) {
  console.error("Usage: node test-multipart.mjs <bucket-name>");
  process.exit(1);
}

const KEY = "test-multipart-upload.bin";
const ORIGIN = "https://example.com";
// S3 minimum part size is 5MB (except last part)
const PART_SIZE = 5 * 1024 * 1024;
const NUM_PARTS = 2;

const s3 = new S3Client();

async function run() {
  // 1. Initiate multipart upload
  const { UploadId } = await s3.send(
    new CreateMultipartUploadCommand({ Bucket: BUCKET, Key: KEY })
  );
  console.log("UploadId:", UploadId);

  const parts = [];

  try {
    for (let partNum = 1; partNum <= NUM_PARTS; partNum++) {
      // 2. Generate presigned URL for this part
      const presignedUrl = await getSignedUrl(
        s3,
        new UploadPartCommand({
          Bucket: BUCKET,
          Key: KEY,
          UploadId,
          PartNumber: partNum,
        }),
        { expiresIn: 3600 }
      );

      // 3. Upload part via fetch with Origin header (simulating browser CORS)
      const body = Buffer.alloc(PART_SIZE, `part${partNum}`);
      const res = await fetch(presignedUrl, {
        method: "PUT",
        body,
        headers: { Origin: ORIGIN },
      });

      // 4. Check CORS headers
      const exposeHeaders = res.headers.get("access-control-expose-headers");
      const etag = res.headers.get("etag");

      console.log(`\n--- Part ${partNum} ---`);
      console.log("Status:", res.status);
      console.log("Access-Control-Expose-Headers:", exposeHeaders);
      console.log("ETag:", etag);

      if (!etag) {
        throw new Error(
          `Part ${partNum}: ETag header is null — browser would fail here`
        );
      }

      if (!exposeHeaders?.includes("ETag")) {
        console.warn(
          `⚠ Access-Control-Expose-Headers does not include ETag — browser JS cannot read it`
        );
      } else {
        console.log("✓ ETag is exposed via CORS");
      }

      parts.push({ ETag: etag, PartNumber: partNum });
    }

    // 5. Complete multipart upload using ETags from each part
    const result = await s3.send(
      new CompleteMultipartUploadCommand({
        Bucket: BUCKET,
        Key: KEY,
        UploadId,
        MultipartUpload: { Parts: parts },
      })
    );
    console.log("\n✓ Multipart upload completed successfully");
    console.log("Location:", result.Location);
  } catch (err) {
    console.error("\n✗ Failed:", err.message);
    await s3.send(
      new AbortMultipartUploadCommand({ Bucket: BUCKET, Key: KEY, UploadId })
    );
    process.exit(1);
  }
}

run();
