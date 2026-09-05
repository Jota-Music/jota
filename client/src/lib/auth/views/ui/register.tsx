import { useState } from "preact/hooks";
import { useLocation } from "wouter-preact";
import { authPhase, logIn, logOut } from "@/lib/auth/views/stores/session";
import { post } from "@/lib/shared/api";

type Mode = "login" | "register";

function Auth() {
	const [, setLocation] = useLocation();
	const [mode, setMode] = useState<Mode>("login");
	const [loading, setLoading] = useState(false);
	const [error, setError] = useState<string | null>(null);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		setError(null);
		setLoading(true);

		const formData = new FormData(e.currentTarget as HTMLFormElement);
		const name = formData.get("name");
		const pass = formData.get("pass");

		const username = typeof name === "string" ? name : "";
		const password = typeof pass === "string" ? pass : "";

		try {
			if (mode === "register") {
				await post("/auth/register", {
					name: username,
					pass: password,
				});

				setMode("login");
			} else {
				const res = await post<{ user: string }>("/auth/log-in", {
					name: username,
					pass: password,
				});

				logIn(res.user);
				setLocation("/");
			}
		} catch (err: unknown) {
			if (err instanceof Error) {
				setError(err.message);
			} else {
				setError("Error");
			}
		} finally {
			setLoading(false);
		}
	}

	if (authPhase.value === "signedIn") {
		return (
			<div class="flex items-center justify-center px-4 h-full">
				<div class="w-full max-w-xs">
					<button
						type="button"
						onClick={() => void logOut()}
						class="w-full py-2 text-sm rounded-lg font-bold bg-(--dominant-color) text-(--binary-color) hover:brightness-75 cursor-pointer transition"
					>
						Log out
					</button>
				</div>
			</div>
		);
	}

	return (
		<div class="flex items-center justify-center px-4 h-full">
			<div class="w-full max-w-xs space-y-6">
				<div class="flex gap-6 text-sm text-zinc-600">
					<button
						type="button"
						onClick={() => setMode("login")}
						class={
							"cursor-pointer " +
							(mode === "login"
								? "text-zinc-200"
								: "hover:text-zinc-400 transition")
						}
					>
						Login
					</button>

					<button
						type="button"
						onClick={() => setMode("register")}
						class={
							"cursor-pointer " +
							(mode === "register"
								? "text-zinc-200"
								: "hover:text-zinc-400 transition")
						}
					>
						Register
					</button>
				</div>

				<div class="relative">
					<div class="absolute inset-0 flex items-center">
						<div class="w-full border-t border-zinc-800" />
					</div>
					<div class="relative flex justify-center text-xs">
						<span class="bg-zinc-950 px-2 text-zinc-600">or</span>
					</div>
				</div>

				<form onSubmit={handleSubmit} class="space-y-4">
					<input
						type="text"
						name="name"
						placeholder="Username"
						required
						class="w-full rounded-md border border-zinc-800 bg-zinc-950 px-3 py-1.5 text-sm text-white outline-none focus:border-zinc-600 transition"
					/>

					<input
						type="password"
						name="pass"
						placeholder="Password"
						required
						class="w-full rounded-md border border-zinc-800 bg-zinc-950 px-3 py-1.5 text-sm text-white outline-none focus:border-zinc-600"
					/>

					<button
						type="submit"
						disabled={loading}
						class="w-full py-2 text-sm rounded-lg font-bold bg-(--dominant-color) text-(--binary-color) hover:brightness-75 cursor-pointer transition disabled:opacity-40 disabled:cursor-not-allowed"
					>
						{loading ? "..." : mode === "login" ? "Sign in" : "Create account"}
					</button>

					{error && <p class="text-xs text-zinc-500">{error}</p>}
				</form>
			</div>
		</div>
	);
}

export default Auth;