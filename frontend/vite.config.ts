/// <reference types="vitest" />
import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

export default defineConfig({
  plugins: [react()],

  server: {
    port: 5173,
    strictPort: true, // fail loudly rather than silently moving to 5174,
    // which would then be blocked by the backend's CORS allowlist.

    // Deliberately NOT proxying /api to the Go server.
    //
    // A dev proxy would make the browser see one origin and CORS would never
    // engage — convenient, but it hides the very thing this template is meant
    // to demonstrate, and it papers over a misconfiguration that would only
    // resurface in production. If you ever want it:
    //
    // proxy: { "/api": { target: "http://localhost:8080", changeOrigin: true } }
  },

  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test/setup.ts"],
    css: false,
    coverage: {
      provider: "v8",
      reporter: ["text", "html"],
      // Config and entrypoints have no logic worth asserting on.
      exclude: ["src/main.tsx", "src/test/**", "**/*.d.ts", "**/*.config.*"],
    },
  },
});
