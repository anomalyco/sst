import { Resource } from "sst";
import { DynamoDBClient } from "@aws-sdk/client-dynamodb";
import { GetCommand, PutCommand, DynamoDBDocumentClient } from "@aws-sdk/lib-dynamodb";
import { Form, useNavigation } from "react-router";
import type { Route } from "./+types/data.$pk";

// Initialize the DynamoDB Document Client
const client = new DynamoDBClient({});
const docClient = DynamoDBDocumentClient.from(client);

export function meta({}: Route.MetaArgs) {
  return [
    { title: "Path Data App" },
    { name: "description", content: "Read and write data using path as PK" },
  ];
}

export async function loader({ request }: Route.LoaderArgs) {
  const url = new URL(request.url);
  const pk = url.pathname;

  const { Item } = await docClient.send(
    new GetCommand({
      TableName: Resource.MyDynamo.name,
      Key: { pk },
    })
  );

  return { content: Item?.content || null, pk };
}

export async function action({ request }: Route.ActionArgs) {
  const url = new URL(request.url);
  const pk = url.pathname;
  
  // Parse the submitted form data
  const formData = await request.formData();
  const content = formData.get("content") as string;

  await docClient.send(
    new PutCommand({
      TableName: Resource.MyDynamo.name,
      Item: {
        pk,
        content,
      },
    })
  );

  return { success: true };
}

// 3. Frontend component
export default function Home({ loaderData, actionData }: Route.ComponentProps) {
  const { content, pk } = loaderData;
  const navigation = useNavigation();
  const isSubmitting = navigation.state === "submitting";

  return (
    <div className="flex h-screen items-center justify-center">
      <div className="flex flex-col items-center gap-8">
        <div className="text-center">
          <h1 className="leading text-2xl font-bold text-gray-800 dark:text-gray-100">
            Path: {pk}
          </h1>
          <p className="text-gray-500 mt-2">
            Current Data: {content ? (
              <span className="font-mono bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded">
                {content}
              </span>
            ) : (
              "No data written yet."
            )}
          </p>
        </div>
        <Form method="post" className="flex flex-row gap-4 w-full max-w-sm">
          <input
            name="content"
            type="text"
            required
            placeholder="Enter new data..."
            className="block w-full text-sm text-slate-800 border border-slate-300 rounded-full px-4 py-2"
          />
          <button
            type="submit"
            disabled={isSubmitting}
            className="bg-violet-500 hover:bg-violet-700 text-white text-sm
            font-semibold py-2 px-6 rounded-full disabled:opacity-50"
          >
            {isSubmitting ? "Saving..." : "Save"}
          </button>
        </Form>
        
        {actionData?.success && (
          <p className="text-sm text-green-500 font-semibold">
            Data successfully saved!
          </p>
        )}
      </div>
    </div>
  );
}