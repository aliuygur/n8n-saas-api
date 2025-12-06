import { createRequestHandler } from "@react-router/cloudflare";
// @ts-ignore
import * as build from "./build/server/index.js";

const handleRequest = createRequestHandler({
  build,
  mode: "production",
});

export default {
  async fetch(request, env, ctx) {
    try {
      return await handleRequest({
        request,
        env,
        waitUntil: ctx.waitUntil.bind(ctx),
        passThroughOnException: ctx.passThroughOnException.bind(ctx),
      });
    } catch (error) {
      console.error("Worker error:", error);
      return new Response(`Internal Server Error: ${error.message}\n${error.stack}`, { status: 500 });
    }
  },
};
