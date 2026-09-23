import { useParams } from "wouter-preact";
import PlaylistPlain from "@/lib/music/views/ui/playlist/playlist";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";

export function RadioPage() {
	const { id } = useParams<{ id: string }>();
	const uri = `spotify:radio:track:${id}`;

	// The playlist view is shared verbatim: search, ordering, selection, play.
	return (
		<DefaultLayout className="gap-4">
			<PlaylistPlain id={uri} />
		</DefaultLayout>
	);
}
