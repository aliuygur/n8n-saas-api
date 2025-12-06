import type { Config } from "@react-router/dev/config";

export default {
  // SSR mode - server-side rendering enabled
  ssr: true,
  // Cloudflare Workers configuration
  buildDirectory: "./build",
  serverBuildFile: "index.js",
  serverModuleFormat: "esm",
} satisfies Config;
