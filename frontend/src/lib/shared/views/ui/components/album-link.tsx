import type { JSX } from "preact";
import { Link } from "wouter-preact";
import type { Album } from "@/lib/music/model";

type Props = {
	album: Album;
	class?: string;
};

export function AlbumLink({ album, class: className }: Props) {
	if (!album.id || !album.title) {
		return <span class={className}>{album.title}</span>;
	}
	return (
		<Link
			href={`/album/${album.id}`}
			class={`${className ?? ""} hover:underline`}
			onClick={(e: JSX.TargetedMouseEvent<HTMLAnchorElement>) => {
				e.stopPropagation();
			}}
		>
			{album.title}
		</Link>
	);
}

export default AlbumLink;
