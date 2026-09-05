import { useQuery } from "@tanstack/preact-query";
import { Check, Trash2, Upload, X } from "lucide-preact";
import { useCallback, useState } from "preact/hooks";
import { useLocation } from "wouter-preact";
import { authPhase } from "@/lib/auth/views/stores/session";
import useMeta from "@/lib/shared/views/hooks/use-meta";
import ClearLayout from "@/lib/shared/views/ui/layouts/clear";

async function getCookiesStatus(): Promise<{ configured: boolean }> {
	const res = await fetch("/api/user/cookies", {
		credentials: "include",
	});
	if (!res.ok) throw new Error("Failed to fetch cookies status");
	return res.json();
}

async function uploadCookies(file: File): Promise<void> {
	const form = new FormData();
	form.set("cookies", file);
	const res = await fetch("/api/user/cookies", {
		method: "POST",
		credentials: "include",
		body: form,
	});
	if (!res.ok) throw new Error("Upload failed");
}

async function deleteCookies(): Promise<void> {
	const res = await fetch("/api/user/cookies", {
		method: "DELETE",
		credentials: "include",
	});
	if (!res.ok) throw new Error("Delete failed");
}

function SettingsPage() {
	useMeta("Jota | Settings", "Configure your Jota settings");
	const [, setLocation] = useLocation();
	const phase = authPhase.value;

	const [file, setFile] = useState<File | null>(null);
	const [uploading, setUploading] = useState(false);
	const [deleting, setDeleting] = useState(false);
	const [error, setError] = useState("");

	const { data, isLoading, refetch } = useQuery({
		queryKey: ["cookies-status"],
		queryFn: getCookiesStatus,
		retry: false,
	});

	const onUpload = useCallback(async () => {
		if (!file) return;
		setUploading(true);
		setError("");
		try {
			await uploadCookies(file);
			setFile(null);
			await refetch();
		} catch {
			setError("Failed to upload cookies file");
		} finally {
			setUploading(false);
		}
	}, [file, refetch]);

	const onDelete = useCallback(async () => {
		setDeleting(true);
		setError("");
		try {
			await deleteCookies();
			await refetch();
		} catch {
			setError("Failed to delete cookies file");
		} finally {
			setDeleting(false);
		}
	}, [refetch]);

	if (phase === "loading") return null;
	if (phase === "guest") {
		setLocation("/register", { replace: true });
		return null;
	}

	return (
		<ClearLayout class="gap-6">
			<h1 class="text-lg font-bold text-zinc-100">Settings</h1>

			<section class="space-y-3">
				<h2 class="text-sm font-semibold text-zinc-300">YouTube Cookies</h2>
				<p class="text-xs text-zinc-500">
					Upload a Netscape-format cookies file to authenticate yt-dlp for
					restricted content.
				</p>

				{isLoading ? (
					<span class="text-xs text-zinc-600">Checking status...</span>
				) : (
					<div class="flex items-center gap-2 text-xs">
						{data?.configured ? (
							<>
								<Check class="size-3 text-emerald-400" />
								<span class="text-emerald-400">Cookies configured</span>
							</>
						) : (
							<>
								<X class="size-3 text-zinc-500" />
								<span class="text-zinc-500">No cookies configured</span>
							</>
						)}
					</div>
				)}

				<label class="flex items-center gap-2 rounded-lg border border-zinc-700 px-3 py-2 text-sm text-zinc-300 cursor-pointer hover:border-zinc-500 transition-colors">
					<Upload class="size-4 shrink-0" />
					<span class="flex-1 truncate">
						{file ? file.name : "Choose cookies file..."}
					</span>
					<input
						type="file"
						accept=".txt"
						class="hidden"
						onChange={(e) => {
							const f = (e.target as HTMLInputElement).files?.[0];
							setFile(f ?? null);
						}}
					/>
				</label>

				<div class="flex gap-2">
					<button
						type="button"
						disabled={!file || uploading}
						onClick={onUpload}
						class="flex items-center gap-1.5 rounded-lg bg-emerald-700 px-4 py-2 text-sm font-bold text-white hover:bg-emerald-600 disabled:opacity-40 disabled:cursor-not-allowed transition-colors cursor-pointer"
					>
						<Upload class="size-4" />
						{uploading ? "Uploading..." : "Upload"}
					</button>

					{data?.configured && (
						<button
							type="button"
							disabled={deleting}
							onClick={onDelete}
							class="flex items-center gap-1.5 rounded-lg bg-red-900/60 px-4 py-2 text-sm font-bold text-red-200 hover:bg-red-800/60 disabled:opacity-40 disabled:cursor-not-allowed transition-colors cursor-pointer"
						>
							<Trash2 class="size-4" />
							{deleting ? "Removing..." : "Remove"}
						</button>
					)}
				</div>

				{error && <p class="text-xs text-red-400">{error}</p>}
			</section>
		</ClearLayout>
	);
}

export default SettingsPage;
