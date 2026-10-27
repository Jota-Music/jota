import { ExportBackup, ImportBackup } from "@bindings/app";
import { useSignal } from "@preact/signals";
import { useMutation, useQueryClient } from "@tanstack/preact-query";
import { Events } from "@wailsio/runtime";
import { Download, Upload } from "lucide-preact";
import { useEffect, useRef } from "preact/hooks";
import { t } from "@/lib/shared/i18n";
import { addError, notify } from "@/lib/shared/views/stores/errors";

type Progress = {
	done: number;
	total: number;
	name?: string;
	failed: number;
};

export function BackupFile() {
	const queryClient = useQueryClient();
	const fileRef = useRef<HTMLInputElement>(null);
	const progress = useSignal<Progress | null>(null);

	useEffect(() => {
		return Events.On("playlists:export", (ev) => {
			const data = ev.data as Progress | undefined;
			if (data) progress.value = data;
		});
	}, []);

	const exportMutation = useMutation({
		mutationFn: ExportBackup,
		onMutate: () => {
			progress.value = { done: 0, total: 0, failed: 0 };
		},
		onSuccess: () => {
			progress.value = null;
			notify(t("settings.playlists.exported"));
		},
		onError: (error) => {
			progress.value = null;
			addError(error, "playlists");
		},
	});

	const importMutation = useMutation({
		mutationFn: async (file: File) => {
			const text = await file.text();
			return ImportBackup(text);
		},
		onSuccess: (stats) => {
			queryClient.invalidateQueries({ queryKey: ["playlists"] });
			queryClient.invalidateQueries({ queryKey: ["playlist"] });
			notify(
				t("settings.playlists.imported", {
					playlists: stats.playlists,
					follows: stats.follows,
					saved: stats.saved,
					order: stats.order,
					settings: stats.settings,
				}),
			);
		},
		onError: (error) => addError(error, "playlists"),
	});

	const pickFile = () => fileRef.current?.click();

	const onFileChange = (event: Event) => {
		const input = event.target as HTMLInputElement;
		const file = input.files?.[0];
		input.value = "";
		if (!file) return;
		importMutation.mutate(file);
	};

	const busy = exportMutation.isPending || importMutation.isPending;
	const running = progress.value;

	return (
		<div class="space-y-4">
			<div class="flex items-center gap-2">
				<Download size={16} class="text-zinc-400" />
				<h3 class="text-xs font-semibold text-zinc-200">
					{t("settings.playlists.title")}
				</h3>
			</div>
			<p class="text-xs text-zinc-500">{t("settings.playlists.description")}</p>

			<div class="grid grid-cols-2 gap-2 sm:flex sm:flex-wrap">
				<button
					type="button"
					disabled={busy}
					onClick={() => exportMutation.mutate()}
					class="flex w-full cursor-pointer items-center justify-center gap-1.5 rounded-lg bg-(--dominant-color) px-3 py-2 text-xs font-semibold text-(--binary-color) transition-opacity hover:opacity-85 disabled:cursor-not-allowed disabled:opacity-50 sm:w-auto"
				>
					<Download size={12} />
					{t("settings.playlists.export")}
				</button>
				<button
					type="button"
					disabled={busy}
					onClick={pickFile}
					class="flex w-full cursor-pointer items-center justify-center gap-1.5 rounded-lg border border-zinc-700 bg-zinc-950 px-3 py-2 text-xs font-semibold text-zinc-100 transition-colors hover:border-zinc-500 disabled:cursor-not-allowed disabled:opacity-50 sm:w-auto"
				>
					<Upload size={12} />
					{t("settings.playlists.import")}
				</button>
			</div>

			{running && (
				<div class="space-y-1">
					<div class="h-1.5 overflow-hidden rounded-full bg-zinc-800">
						{running.total === 0 ? (
							<div class="h-full w-1/3 animate-pulse rounded-full bg-(--dominant-color)" />
						) : (
							<div
								class="h-full rounded-full bg-(--dominant-color) transition-[width]"
								style={{
									width: `${Math.round((running.done / running.total) * 100)}%`,
								}}
							/>
						)}
					</div>
					<p class="truncate text-xs text-zinc-500">
						{t("settings.playlists.exporting", {
							done: running.done,
							total: running.total,
						})}
						{running.name ? ` · ${running.name}` : ""}
					</p>
				</div>
			)}

			<input
				ref={fileRef}
				type="file"
				accept="application/json,.json"
				class="hidden"
				onChange={onFileChange}
			/>
		</div>
	);
}
