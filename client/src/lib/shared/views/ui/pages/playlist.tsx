import "@/lib/music/views/stores/audio";

import { useParams } from "wouter-preact";
import PlaylistPlain from "@/lib/music/views/ui/playlist/playlist";
import useMeta from "@/lib/shared/views/hooks/use-meta";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";

export function PlaylistPage() {
	const { id } = useParams<{ id: string }>();
	useMeta("Jota | Playlist", "Browse and listen to playlists on Jota");
	return (
		<DefaultLayout className="gap-4">
			<PlaylistPlain id={id} />
		</DefaultLayout>
	);
}
