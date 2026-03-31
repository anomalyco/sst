import type { Route } from "./+types/server-rendered";

export async function clientLoader() {
  return {
    message: "Hello from the clientLoader()",
  };
}

export default function MyRouteComponent({ loaderData }: Route.ComponentProps) {
  return (
    <div>
      <h1>This page is client rendered</h1>
      <p>{loaderData.message}</p>
    </div>
  );
}
