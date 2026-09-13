import { Loader } from "lucide-preact";
import { useState } from "preact/hooks";
import { loginSpotifyAndWait } from "@/lib/auth/views/stores/session";
import useMeta from "@/lib/shared/views/hooks/use-meta";
import { WindowControlsBar } from "@/lib/shared/views/ui/components/window-controls-bar";

function LoginPage() {
	useMeta("Jota | Log in", "Log in with Spotify to start listening");

	const [busy, setBusy] = useState(false);
	const [error, setError] = useState("");

	const handleLogin = async () => {
		setBusy(true);
		setError("");

		const ok = await loginSpotifyAndWait();
		if (!ok) {
			setBusy(false);
			setError("Spotify login failed. Check the terminal for details.");
		}
	};

	function handleDragMouseDown(e: MouseEvent) {
		const target = e.target as HTMLElement;
		if (target.closest("button")) return;
		window.getSelection()?.removeAllRanges();
	}

	return (
		<div class="h-dvh flex flex-col bg-stone-950">
			<header
				style="--wails-draggable: drag"
				onMouseDown={handleDragMouseDown}
				class="sticky top-0 z-40 md:z-100 border-b border-zinc-800 bg-zinc-950/80 backdrop-blur"
			>
				<div class="mx-auto flex max-w-2xl items-center justify-end h-10 text-sm text-zinc-300 pr-0">
					<WindowControlsBar />
				</div>
			</header>

			<div class="flex flex-1 items-center justify-center">
				<div class="flex flex-col items-center gap-6 px-6 text-center">
					<div class="space-y-1">
						<h1 class="text-2xl font-bold text-white">Jota</h1>
						<p class="text-sm text-zinc-400">
							Log in with Spotify to start listening
						</p>
					</div>

					{busy ? (
						<div class="flex flex-col items-center gap-3">
							<Loader size={24} class="text-zinc-500 animate-spin" />
							<p class="text-sm text-zinc-500">
								Complete Spotify login in the browser window that opened, then
								return here.
							</p>
						</div>
					) : (
						<button
							type="button"
							onClick={() => void handleLogin()}
							class="flex items-center gap-2 rounded-full bg-[#1DB954] px-6 py-2.5 text-sm font-bold text-black hover:opacity-80 transition-opacity cursor-pointer"
						>
							Log in with Spotify
						</button>
					)}

					{error && <p class="text-xs text-red-400">{error}</p>}
				</div>
			</div>
		</div>
	);
}

export default LoginPage;
