import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

// In dev, Vite forwards /api to the Go server so the browser sees one origin (no CORS needed).
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: { "/api": "http://localhost:8080" },
  },
  test: { environment: "node" },
});
