//import { Resource } from "sst";
import { createAsync } from "@solidjs/router";

async function loadBucket() {
  "use server";
  return {
    //bucket: Resource.MyBucket.name,
    bucket: "test",
    foo: process.env.FOO,
  };
}

export const route = {
  load: () => loadBucket(),
};

export default function Home() {
  const bucket = createAsync(() => loadBucket());

  return (
    <main>
      <h1>Hello world!</h1>
      <p>Bucket: {bucket()?.bucket}</p>
      <p>Foo: {bucket()?.foo}</p>
    </main>
  );
}
