import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "node",
    globals: false,
    include: ["src/**/*.integration.test.ts"],

    // These share one database, so running files in parallel would have them
    // truncating each other's rows mid-test.
    fileParallelism: false,
    testTimeout: 15_000,
  },
});
