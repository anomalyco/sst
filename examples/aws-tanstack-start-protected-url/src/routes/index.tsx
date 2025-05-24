import { Resource } from 'sst'
import Form from '~/components/Form'
import { createServerFn } from '@tanstack/react-start'
import { createFileRoute } from '@tanstack/react-router'
import { getSignedUrl } from '@aws-sdk/s3-request-presigner'
import { S3Client, PutObjectCommand } from '@aws-sdk/client-s3'

export const getPresignedUrl = createServerFn().handler(async () => {
  const command = new PutObjectCommand({
    Key: crypto.randomUUID(),
    Bucket: Resource.MyBucket.name,
  })
  return await getSignedUrl(new S3Client({}), command)
})

export const Route = createFileRoute('/')({
  component: RouteComponent,
  loader: async () => {
    return { url: await getPresignedUrl() }
  },
})

function RouteComponent() {
  const { url } = Route.useLoaderData()
  return (
    <main>
      <Form url={url} />
    </main>
  )
}
