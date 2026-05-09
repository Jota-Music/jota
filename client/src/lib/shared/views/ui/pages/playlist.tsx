import "@/lib/music/views/stores/audio";

import { useParams } from "wouter-preact";
import PlaylistPlain from "@/lib/music/views/ui/playlist/playlist";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";

export function PlaylistPage() {
	const { id } = useParams<{ id: string }>();
	return (
		<DefaultLayout className="gap-4">
			<PlaylistPlain id={id} />
		</DefaultLayout>
	);
}
