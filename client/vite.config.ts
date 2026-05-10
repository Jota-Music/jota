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
    allowedHosts: [
      "localhost",
      "127.0.0.1",
      "jota.loca.lt",
      "https://jota.loca.lt",
    ],
  },
});
