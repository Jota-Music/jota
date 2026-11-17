import type { JSX } from "preact";
import { Link } from "wouter-preact";
import { prefetch } from "@/lib/music/app/prefetch";
import type { Album } from "@/lib/music/model";
import { cn } from "@/lib/shared/utils/tw";

type Props = {
	album: Album;
	// A song taken from a playlist names it in the album slot; the playlist id
	// is what the player can open.
	playlistId?: string;
	class?: string;
};

export function AlbumLink({ album, playlistId, class: className }: Props) {
	const to = playlistId
		? `/playlist/${playlistId}`
		: album.id
			? `/album/${album.id}`
			: "";
	if (!to || !album.title) {
		return <span class={className}>{album.title}</span>;
	}
	return (
		<Link
			href={to}
			class={cn(className, "hover:underline")}
			onPointerEnter={() => prefetch(to)}
			onFocus={() => prefetch(to)}
			onClick={(e: JSX.TargetedMouseEvent<HTMLAnchorElement>) => {
				e.stopPropagation();
			}}
		>
			{album.title}
		</Link>
	);
}

export default AlbumLink;
