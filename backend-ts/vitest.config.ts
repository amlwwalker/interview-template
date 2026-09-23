import { configDefaults, defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    // Node, not jsdom — this is a server.
    environment: "node",
    globals: false,

    // Integration tests need a real Postgres, so they are excluded here and
    // run by their own config. Same split as the Go build tag.
    exclude: [...configDefaults.exclude, "**/*.integration.test.ts"],

    coverage: {
      provider: "v8",
      reporter: ["text", "html"],
      exclude: ["src/index.ts", "**/*.config.*", "dist/**"],
    },
  },
});
