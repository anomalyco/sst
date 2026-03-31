import { type RouteConfig, index, route } from "@react-router/dev/routes";

export default [
  index("routes/home.tsx"),
  route("server-rendered", "./routes/server-rendered.tsx"),
  route("client-rendered", "./routes/client-rendered.tsx"),
  route("prerendered", "./routes/prerendered.tsx"),
  route("plus+sign-in-path", "./routes/plus-sign-in-path.tsx"),
  route("error", "./routes/error.tsx"),
] satisfies RouteConfig;
