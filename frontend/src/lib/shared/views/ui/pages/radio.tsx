import { useParams } from "wouter-preact";
import PlaylistPlain from "@/lib/music/views/ui/playlist/playlist";

export function RadioPage() {
	const { id } = useParams<{ id: string }>();
	const uri = `spotify:radio:track:${id}`;

	// The playlist view is shared verbatim: search, ordering, selection, play.
	return <PlaylistPlain id={uri} />;
}
