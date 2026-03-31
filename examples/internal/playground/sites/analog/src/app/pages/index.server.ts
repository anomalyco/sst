import { Resource } from 'sst';
import { PageServerLoad } from '@analogjs/router';
import { getSignedUrl } from '@aws-sdk/s3-request-presigner';
import { S3Client, PutObjectCommand } from '@aws-sdk/client-s3';

export const load = async ({ }: PageServerLoad) => {
  const command = new PutObjectCommand({
    Key: crypto.randomUUID(),
    // @ts-ignore: Generated on deploy
    Bucket: Resource.MyBucket.name,
  });

  const url = await getSignedUrl(new S3Client({}), command);

  return {
    url
  };
};
