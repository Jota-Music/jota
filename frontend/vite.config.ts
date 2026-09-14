import preact from "@preact/preset-vite";
import tailwindcss from "@tailwindcss/vite";
import { fileURLToPath, URL } from "node:url";
import { defineConfig } from "vite";
import wails from "@wailsio/runtime/plugins/vite";

// https://vite.dev/config/
export default defineConfig({
	plugins: [wails("./bindings"), tailwindcss(), preact()],
	resolve: {
		tsconfigPaths: true,
		alias: {
			"@bindings/": fileURLToPath(
				new URL("bindings/github.com/Jota-Music/jota/internal/app/", import.meta.url),
			),
		},
	},
	envDir: "../",
	clearScreen: false,
	server: {
		host: "127.0.0.1",
		proxy: {
			"/api": {
				target: `http://localhost:${process.env.PORT || "3001"}`,
				changeOrigin: true,
			},
			"/ws": {
				target: `ws://localhost:${process.env.PORT || "3001"}`,
				ws: true,
			},
		},
	},
});
