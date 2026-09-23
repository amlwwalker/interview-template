import "@testing-library/jest-dom/vitest";

import { cleanup } from "@testing-library/react";
import { afterEach } from "vitest";

// Testing Library does not auto-clean when globals are enabled in every
// Vitest version, and a leaked DOM between tests produces confusing
// "found multiple elements" failures. Explicit is cheaper than surprising.
afterEach(() => {
  cleanup();
});
