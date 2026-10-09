import { Outlet, Head } from "@openauthjs/openauth/ui/custom/layout";
// Don't import global css (or other static assets) here!!! Use the main.tsx to do that

// This is an hono component, don't put react imports here.
// Everything here only runs on the server.
// The Head component is required




export default function Layout() {
  return (
    <html lang="en" className="h-full bg-slate-950">
      <Head>
        <link rel="icon" type="image/x-icon" href="/vite.svg" />
        <title>Hello</title>
      </Head>
      <body className="h-full text-slate-100 antialiased selection:bg-indigo-500 selection:text-white">
        <Outlet />
      </body>
    </html>
  );
}
