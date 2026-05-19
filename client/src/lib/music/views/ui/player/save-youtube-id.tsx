import { useSignal } from "@preact/signals";
import { Check, Edit2, X } from "lucide-preact";
import { useEffect, useRef, useState } from "preact/hooks";
import { updateYoutubeId } from "@/lib/music/app/get-audio";
import type { Song } from "@/lib/music/model";
import YoutubeIcon from "@/lib/shared/views/ui/icons/youtube";

export default function SaveYoutubeId({
	song,
}: {
	song: Song;
	isPlaying: boolean;
}) {
	const [isEditing, setIsEditing] = useState(false);
	const currentYoutubeId = useSignal(song.youtubeId ?? "");
	const $input = useRef<HTMLInputElement>(null);

	const onStartEdit = () => {
		currentYoutubeId.value = song.youtubeId ?? "";
		setIsEditing(true);
	};

	useEffect(() => {
		if (isEditing) {
			$input.current?.focus();
			$input.current?.select();
		}
	}, [isEditing]);

	async function handlSubmit(e: Event) {
		e.preventDefault();
		const formData = new FormData(e.target as HTMLFormElement);
		const youtube = formData.get("youtube") as string;

		if (song.youtubeId === youtube) {
			setIsEditing(false);
			return;
		}

		await updateYoutubeId(song.id, youtube);
		song.youtubeId = youtube;
		currentYoutubeId.value = youtube;

		setIsEditing(false);
	}

	return (
		<div class="mt-2 flex gap-2 items-center">
			<YoutubeIcon class="size-4 text-white/20" />

			{isEditing ? (
				<form class="flex gap-2 items-center h-8" onSubmit={handlSubmit}>
					<input
						ref={$input}
						type="text"
						value={currentYoutubeId.value}
						name="youtube"
						placeholder="YouTube ID"
						class="w-full rounded-md border border-zinc-800 bg-zinc-950 px-3 py-1 text-sm text-white outline-none focus:border-zinc-600"
						onBlur={(e) => {
							if (e.relatedTarget && (e.target as HTMLElement).closest("form")?.contains(e.relatedTarget as HTMLElement)) {
								return;
							}
							setIsEditing(false);
						}}
						onInput={(e) => {
							currentYoutubeId.value = (e.target as HTMLInputElement).value;
						}}
						onKeyDown={(e) => {
							if (e.key === "Escape") {
								setIsEditing(false);
							}
						}}
					/>

					<button
						type="submit"
						class="cursor-pointer p-1 rounded hover:brightness-200 transition disabled:opacity-50 disabled:cursor-not-allowed"
						title="Guardar"
					>
						<Check size={16} />
					</button>

					<button
						type="button"
						class="cursor-pointer p-1 rounded hover:brightness-200 transition disabled:opacity-50 disabled:cursor-not-allowed"
						title="Cancelar"
						onClick={() => setIsEditing(false)}
					>
						<X size={16} />
					</button>
				</form>
			) : (
				<button
					type="button"
					class="flex gap-2 items-center group h-8"
					onClick={onStartEdit}
				>
					<p class="text-white text-xs opacity-50 flex-1 truncate flex items-center gap-2">
						{song.youtubeId ? (
							<button
								type="button"
								class="cursor-pointer font-mono bg-white/10 px-1.5 py-0.5 rounded"
							>
								{song.youtubeId}
							</button>
						) : (
							<span class="cursor-pointer font-mono bg-white/10 px-1.5 py-0.5 rounded">
								...
							</span>
						)}
					</p>

					{song.youtubeId && (
						<span
							class="cursor-pointer p-1 rounded hover:brightness-200 transition opacity-0 group-hover:opacity-100"
							title="Editar YouTube ID"
						>
							<Edit2 size={16} />
						</span>
					)}
				</button>
			)}
		</div>
	);
}
