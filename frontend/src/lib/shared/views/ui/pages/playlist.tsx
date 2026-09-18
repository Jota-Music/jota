import "@/lib/music/views/stores/audio";

import { useParams } from "wouter-preact";
import { RequireSpotify } from "@/lib/auth/views/ui/spotify-connect";
import { isCustom } from "@/lib/music/app/playlists";
import PlaylistPlain from "@/lib/music/views/ui/playlist/playlist";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";

export function PlaylistPage() {
	const { id } = useParams<{ id: string }>();

	const content = (
		<DefaultLayout className="gap-4">
			<PlaylistPlain id={id} />
		</DefaultLayout>
	);

	if (id?.startsWith("youtube:") || isCustom(id)) return content;
	return <RequireSpotify>{content}</RequireSpotify>;
}
