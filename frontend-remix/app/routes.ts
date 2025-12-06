import { type RouteConfig, index, route } from "@react-router/dev/routes";

export default [
  index("routes/home.tsx"),
  route("login", "routes/login.tsx"),
  route("auth/callback", "routes/auth.callback.tsx"),
  route("dashboard", "routes/dashboard.tsx"),
  route("create-instance", "routes/create-instance.tsx"),
] satisfies RouteConfig;
