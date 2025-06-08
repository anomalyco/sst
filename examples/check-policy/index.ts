import { S3Client, GetObjectCommand } from "@aws-sdk/client-s3";

const s3Client = new S3Client({});

export async function handler() {
  try {
    const command = new GetObjectCommand({
      Bucket: "example-bucket",
      Key: "example-key",
    });

    await s3Client.send(command);

    return {
      statusCode: 200,
      body: JSON.stringify({ message: "Successfully accessed S3" }),
    };
  } catch (error) {
    return {
      statusCode: 500,
      body: JSON.stringify({ message: "Error accessing S3", error }),
    };
  }
}