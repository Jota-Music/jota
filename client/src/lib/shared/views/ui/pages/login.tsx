import { Loader } from "lucide-preact";
import { useState } from "preact/hooks";
import { loginSpotify } from "@/lib/auth/views/stores/session";
import useMeta from "@/lib/shared/views/hooks/use-meta";

function LoginPage() {
	useMeta("Jota | Log in", "Log in with Spotify to start listening");

	const [busy, setBusy] = useState(false);
	const [error, setError] = useState("");

	const handleLogin = async () => {
		setBusy(true);
		setError("");

		const authURL = await loginSpotify();
		if (authURL) {
			window.location.href = authURL;
			return;
		}

		setBusy(false);
		setError("Could not start Spotify login");
	};

	return (
		<div class="h-dvh flex items-center justify-center bg-stone-950">
			<div class="flex flex-col items-center gap-6 px-6 text-center">
				<div class="space-y-1">
					<h1 class="text-2xl font-bold text-white">Jota</h1>
					<p class="text-sm text-zinc-400">
						Log in with Spotify to start listening
					</p>
				</div>

				<button
					type="button"
					disabled={busy}
					onClick={() => void handleLogin()}
					class="flex items-center gap-2 rounded-full bg-[#1DB954] px-6 py-2.5 text-sm font-bold text-black hover:opacity-80 transition-opacity cursor-pointer disabled:opacity-50"
				>
					{busy ? (
						<span class="flex items-center gap-2">
							<Loader size={14} class="animate-spin" />
							Opening Spotify...
						</span>
					) : (
						<span>Log in with Spotify</span>
					)}
				</button>

				{error && <p class="text-xs text-red-400">{error}</p>}
			</div>
		</div>
	);
}

export default LoginPage;
