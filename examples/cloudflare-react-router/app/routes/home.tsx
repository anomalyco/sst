import { Resource } from "sst/resource";
import type { Route } from "./+types/home";
import { Welcome } from "../welcome/welcome";

export function meta({}: Route.MetaArgs) {
  return [
    { title: "React Router on Cloudflare with SST" },
    { name: "description", content: "Deployed with sst.cloudflare.ReactRouter" },
  ];
}

export async function loader() {
  const count = await Resource.MyKv.get("counter");
  return { count: count ? parseInt(count, 10) : 0 };
}

export async function action() {
  const current = await Resource.MyKv.get("counter");
  const next = current ? parseInt(current, 10) + 1 : 1;
  await Resource.MyKv.put("counter", next.toString());
  return { count: next };
}

export default function Home({ loaderData, actionData }: Route.ComponentProps) {
  const count = actionData?.count ?? loaderData.count;
  return <Welcome count={count} />;
}
