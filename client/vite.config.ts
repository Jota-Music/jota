import preact from "@preact/preset-vite";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";

// https://vite.dev/config/
export default defineConfig({
  plugins: [tailwindcss(), preact()],
  resolve: {
    tsconfigPaths: true,
  },
  envDir: "../",
  server: {
    proxy: {
      "/api": {
        target: `http://localhost:${process.env.PORT || "3002"}`,
        changeOrigin: true,
      },
      "/ws": {
        target: `ws://localhost:${process.env.PORT || "3002"}`,
        ws: true,
      },
    },
  },
});
