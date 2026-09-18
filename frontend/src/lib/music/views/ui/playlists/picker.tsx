import { useMutation, useQuery, useQueryClient } from "@tanstack/preact-query";
import { Check, Loader, Plus } from "lucide-preact";
import { useEffect, useState } from "preact/hooks";
import {
	addSongsToPlaylist,
	createPlaylist,
	getPlaylists,
} from "@/lib/music/app/playlists";
import type { Song } from "@/lib/music/model";
import {
	clearSelection,
	closePicker,
	pickerSongs,
} from "@/lib/music/views/stores/selection";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import { addError } from "@/lib/shared/views/stores/errors";
import { Modal } from "@/lib/shared/views/ui/components/modal";

export function PlaylistPicker() {
	const pending = pickerSongs.value;
	const queryClient = useQueryClient();
	const [open, setOpen] = useState(false);
	const [songs, setSongs] = useState<Song[]>([]);
	const [name, setName] = useState("");
	const [targets, setTargets] = useState<Set<string>>(new Set());

	useEffect(() => {
		if (!pending) return;
		setSongs(pending);
		setOpen(true);
	}, [pending]);

	const { data: playlists = [] } = useQuery({
		queryKey: ["playlists"],
		queryFn: getPlaylists,
	});

	const close = () => {
		setOpen(false);
		closePicker();
		setName("");
		setTargets(new Set());
	};

	const submit = useMutation({
		mutationFn: async () => {
			const refs = songs.filter((s) => !s.broken).map((s) => s.id);
			if (refs.length === 0) throw new Error("no songs selected");

			const ids = new Set(targets);
			const trimmed = name.trim();
			if (trimmed) {
				const created = await createPlaylist(trimmed);
				ids.add(created.id);
			}
			if (ids.size === 0) throw new Error("no playlists selected");

			for (const id of ids) {
				await addSongsToPlaylist(id, refs);
			}
		},
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["playlists"] });
			queryClient.invalidateQueries({ queryKey: ["playlist"] });
			clearSelection();
			close();
		},
		onError: (error) => addError(error, "playlist"),
	});

	const toggleTarget = (id: string) => {
		setTargets((current) => {
			const next = new Set(current);
			if (next.has(id)) next.delete(id);
			else next.add(id);
			return next;
		});
	};

	const canSubmit =
		(name.trim() !== "" || targets.size > 0) && songs.length > 0;

	return (
		<Modal open={open} close={close} labelledBy="playlist-picker-title">
			<div class="flex min-h-0 flex-col">
				<header class="shrink-0 border-b border-zinc-800 px-4 py-3">
					<h2
						id="playlist-picker-title"
						class="text-sm font-semibold text-white"
					>
						{t("music.custom.pickerTitle")}
					</h2>
					<p class="mt-0.5 text-xs text-zinc-500">
						{t("music.custom.selected", { count: songs.length })}
					</p>
				</header>

				<div class="min-h-0 flex-1 overflow-y-auto p-2">
					{playlists.length === 0 ? (
						<p class="px-2 py-3 text-xs text-zinc-500">
							{t("music.custom.noPlaylists")}
						</p>
					) : (
						<ul class="flex flex-col">
							{playlists.map((playlist) => {
								const selected = targets.has(playlist.id);
								return (
									<li key={playlist.id}>
										<button
											type="button"
											onClick={() => toggleTarget(playlist.id)}
											class={cn(
												"flex w-full cursor-pointer items-center gap-3 rounded-md px-3 py-2 text-left transition",
												selected ? "bg-zinc-800/70" : "hover:bg-zinc-900",
											)}
										>
											<span
												class={cn(
													"flex size-5 shrink-0 items-center justify-center rounded border",
													selected
														? "border-(--dominant-color) bg-(--dominant-color) text-(--binary-color)"
														: "border-zinc-600",
												)}
											>
												{selected && <Check size={14} strokeWidth={3} />}
											</span>
											<span class="min-w-0 flex-1 truncate text-sm text-zinc-200">
												{playlist.name}
											</span>
										</button>
									</li>
								);
							})}
						</ul>
					)}
				</div>

				<footer class="shrink-0 border-t border-zinc-800 p-3">
					<div class="flex items-center gap-2">
						<Plus size={16} class="shrink-0 text-zinc-500" />
						<input
							type="text"
							value={name}
							onInput={(e) => setName((e.target as HTMLInputElement).value)}
							placeholder={t("music.custom.namePlaceholder")}
							class="h-10 min-w-0 flex-1 rounded-md border border-zinc-800 bg-zinc-950 px-3 text-sm text-white outline-none focus:border-zinc-600"
						/>
						<button
							type="button"
							disabled={!canSubmit || submit.isPending}
							onClick={() => submit.mutate()}
							class="flex h-10 shrink-0 cursor-pointer items-center gap-2 rounded-md bg-(--dominant-color) px-4 text-sm font-medium text-(--binary-color) transition-opacity hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-40"
						>
							{submit.isPending && <Loader size={14} class="animate-spin" />}
							{t("music.custom.add")}
						</button>
					</div>
				</footer>
			</div>
		</Modal>
	);
}
