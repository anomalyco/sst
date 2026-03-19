import { Resource } from "sst";
import Form from "@/components/form";
import { getSignedUrl } from "@aws-sdk/s3-request-presigner";
import { S3Client, PutObjectCommand } from "@aws-sdk/client-s3";
import Link from "next/link";
import styles from "./page.module.css";

export const dynamic = "force-dynamic";

export default async function Home() {
  const command = new PutObjectCommand({
    Key: crypto.randomUUID(),
    // @ts-ignore
    Bucket: Resource.MyBucket.name,
  });
  const url = await getSignedUrl(new S3Client({}), command);

  return (
    <div className={styles.page}>
      <main className={styles.main}>
        <Form url={url} />
        <div style={{ marginTop: "20px" }}>
          <Link href="/image-test" style={{ textDecoration: "underline" }}>
            Image Optimization Test
          </Link>
        </div>
      </main>
      <h1>Test Links</h1>
      <ul>
        <li>
          <a href="./.well-known/assetlinks.json">
            .well-known/assetlinks.json
          </a>
        </li>
        <li>
          <a href="./.well-known/apple-app-site-association">
            .well-known/apple-app-site-association
          </a>
        </li>
      </ul>
    </div>
  );
}
