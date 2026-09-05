import preact from "@preact/preset-vite";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";
import { VitePWA } from "vite-plugin-pwa";

// https://vite.dev/config/
export default defineConfig({
	plugins: [
		tailwindcss(),
		preact(),
		VitePWA({
			registerType: "autoUpdate",
			manifest: {
				display: "standalone",
				display_override: ["window-controls-overlay"],
				lang: "es-ES",
				name: "Jota",
				short_name: "Jota",
				description: "Jota a simple self-hosted music service",
				theme_color: "#FF6DD6",
				background_color: "#d4d4d4",
				icons: [
					{
						src: "pwa-64x64.png",
						sizes: "64x64",
						type: "image/png",
					},
					{
						src: "pwa-192x192.png",
						sizes: "192x192",
						type: "image/png",
						purpose: "any",
					},
					{
						src: "pwa-512x512.png",
						sizes: "512x512",
						type: "image/png",
						purpose: "maskable",
					},
				],
			},
		}),
	],
	resolve: {
		tsconfigPaths: true,
	},
	envDir: "../",
	server: {
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
