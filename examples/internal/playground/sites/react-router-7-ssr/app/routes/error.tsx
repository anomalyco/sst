import type { Route } from "./+types/error";

export async function loader() {
  throw new Error("This is a test error");
  return {
    message: "Hello from the loader()",
  };
}

export default function MyRouteComponent({ loaderData }: Route.ComponentProps) {
  return (
    <div>
      <h1>This page is server rendered</h1>
      <p>{loaderData.message}</p>
    </div>
  );
}
