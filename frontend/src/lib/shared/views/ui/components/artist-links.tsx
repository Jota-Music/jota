import type { JSX } from "preact";
import { Link } from "wouter-preact";
import type { Artist } from "@/lib/music/model";

type Props = {
	artists: Artist[];
	class?: string;
};

function ArtistLink({
	artist,
	class: className,
}: {
	artist: Artist;
	class?: string;
}) {
	if (!artist.id) {
		return <span class={className}>{artist.name}</span>;
	}
	return (
		<Link
			href={`/artist/${artist.id}`}
			class={`${className ?? ""} hover:underline`}
			onClick={(e: JSX.TargetedMouseEvent<HTMLAnchorElement>) => {
				e.stopPropagation();
			}}
		>
			{artist.name}
		</Link>
	);
}

export function ArtistLinks({ artists, class: className }: Props) {
	return (
		<span class={className}>
			{artists.map((artist, i) => (
				<span key={`${artist.name}-${i}`}>
					{i > 0 && ", "}
					<ArtistLink artist={artist} />
				</span>
			))}
		</span>
	);
}

export default ArtistLinks;
