import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  publicDir: "assets",
  server: {
    proxy: {
      // Proxy /api/* to the Go backend during development.
      // In production the Go server serves the built frontend from ./frontend/dist/
      // so they share the same origin and no proxy is needed.
      "/api": {
        target: "http://localhost:9090",
        changeOrigin: true,
      },
    },
  },
});
