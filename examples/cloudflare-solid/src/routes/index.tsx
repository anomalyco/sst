import { createAsync } from "@solidjs/router";

async function loadMessage() {
  "use server";

  return {
    message: process.env.MESSAGE ?? "Hello from SolidStart",
  };
}

export const route = {
  load: () => loadMessage(),
};

export default function Home() {
  const data = createAsync(() => loadMessage());

  return (
    <main>
      <h1>Hello world!</h1>
      <p>{data()?.message}</p>
    </main>
  );
}
