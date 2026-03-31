import type { Route } from "./+types/server-rendered";
import { v4 } from "uuid";

export async function loader() {
  return {
    message: "Hello from the loader() with uuid: " + v4(),
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
