import { useState } from "react";
import type { User } from "@/lib/auth/views/model/user";
import {
    authPhase,
    logIn,
    logOut
} from "@/lib/auth/views/stores/session";
import { post } from "@/lib/shared/api";

type Mode = "login" | "register";

function Auth() {
	const [mode, setMode] = useState<Mode>("login");
	const [loading, setLoading] = useState(false);
	const [error, setError] = useState<string | null>(null);

	async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
		e.preventDefault();
		setError(null);
		setLoading(true);

		const formData = new FormData(e.currentTarget);
		const name = formData.get("name");
		const pass = formData.get("pass");

		const username = typeof name === "string" ? name : "";
		const password = typeof pass === "string" ? pass : "";

		try {
			if (mode === "register") {
				await post<User>("/auth/register", {
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
			<div className="flex items-center justify-center px-4 h-full">
				<div className="w-full max-w-xs">
					<button
						type="button"
						onClick={() => void logOut()}
						className="w-full py-2 text-sm rounded-lg font-bold bg-(--dominant-color) text-(--binary-color) hover:brightness-75 cursor-pointer transition"
					>
						Log out
					</button>
				</div>
			</div>
		);
	}

	return (
		<div className="flex items-center justify-center px-4 h-full">
			<div className="w-full max-w-xs space-y-6">
				<div className="flex gap-6 text-sm text-zinc-600">
					<button
						type="button"
						onClick={() => setMode("login")}
						className={
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
						className={
							"cursor-pointer " +
							(mode === "register"
								? "text-zinc-200"
								: "hover:text-zinc-400 transition")
						}
					>
						Register
					</button>
				</div>

				{/* Form */}
				<form onSubmit={handleSubmit} className="space-y-4">
					<input
						type="text"
						name="name"
						placeholder="Username"
						required
						className="w-full rounded-md border border-zinc-800 bg-zinc-950 px-3 py-1.5 text-sm text-white outline-none focus:border-zinc-600 transition"
					/>

					<input
						type="password"
						name="pass"
						placeholder="Password"
						required
						className="w-full rounded-md border border-zinc-800 bg-zinc-950 px-3 py-1.5 text-sm text-white outline-none focus:border-zinc-600"
					/>

					<button
						type="submit"
						disabled={loading}
						className="w-full py-2 text-sm rounded-lg font-bold bg-(--dominant-color) text-(--binary-color) hover:brightness-75 cursor-pointer transition disabled:opacity-40 disabled:cursor-not-allowed"
					>
						{loading ? "..." : mode === "login" ? "Sign in" : "Create account"}
					</button>

					{error && <p className="text-xs text-zinc-500">{error}</p>}
				</form>
			</div>
		</div>
	);
}

export default Auth;
