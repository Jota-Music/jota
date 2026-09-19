import { useSignal } from "@preact/signals";
import { createPlaylist } from "@/lib/music/app/playlists";
import type { PlaylistSummary } from "@/lib/music/model";
import { t } from "@/lib/shared/i18n";
import { addError } from "@/lib/shared/views/stores/errors";
import { Modal, ModalHeader } from "@/lib/shared/views/ui/components/modal";

export function CreatePlaylistModal({
	open,
	close,
	onCreated,
}: {
	open: boolean;
	close: () => void;
	onCreated: (playlist: PlaylistSummary) => void;
}) {
	const name = useSignal("");
	const saving = useSignal(false);

	const closeAndReset = () => {
		name.value = "";
		close();
	};

	const submit = async () => {
		saving.value = true;
		try {
			const playlist = await createPlaylist(name.value);
			onCreated(playlist);
			name.value = "";
			close();
		} catch (error) {
			addError(error, "playlist");
		} finally {
			saving.value = false;
		}
	};

	return (
		<Modal
			open={open}
			close={closeAndReset}
			labelledBy="create-playlist-title"
			hideClose
		>
			<div class="flex flex-col">
				<ModalHeader close={closeAndReset}>
					<h2
						id="create-playlist-title"
						class="text-sm font-semibold text-white"
					>
						{t("music.custom.create")}
					</h2>
				</ModalHeader>
				<form
					onSubmit={(e) => {
						e.preventDefault();
						void submit();
					}}
					class="flex items-center gap-2 p-4"
				>
					<input
						type="text"
						value={name.value}
						onInput={(e) => {
							name.value = (e.target as HTMLInputElement).value;
						}}
						placeholder={t("music.custom.namePlaceholder")}
						class="h-10 min-w-0 flex-1 rounded-md border border-zinc-800 bg-zinc-950 px-3 text-sm text-white outline-none focus:border-zinc-600"
					/>
					<button
						type="submit"
						disabled={saving.value}
						class="flex h-10 shrink-0 cursor-pointer items-center rounded-md bg-(--dominant-color) px-4 text-sm font-medium text-(--binary-color) transition-opacity hover:opacity-80 disabled:opacity-40"
					>
						{t("music.custom.submit")}
					</button>
				</form>
			</div>
		</Modal>
	);
}
