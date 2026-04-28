import { Resource } from "sst/resource";
import Form from "@/components/form";
import { getSignedUrl } from "@aws-sdk/s3-request-presigner";
import { S3Client, PutObjectCommand } from "@aws-sdk/client-s3";
import styles from "./page.module.css";

export const dynamic = "force-dynamic";

export default async function Home() {
  const command = new PutObjectCommand({
    Key: crypto.randomUUID(),
    Bucket: Resource.MyBucket.name,
  });

  const url = await getSignedUrl(new S3Client({}), command);
  const verification = {
    app: Resource.App,
    environment: {
      API_URL: process.env.API_URL,
    },
    secret: Resource.MySecret.value,
  };

  return (
    <div className={styles.page}>
      <main className={styles.main}>
        <section className={styles.card}>
          <h1>AWS Next.js</h1>
          <h2>Runtime values</h2>
          <pre>{JSON.stringify(verification, null, 2)}</pre>
        </section>
        <Form url={url} />
      </main>
    </div>
  );
}
