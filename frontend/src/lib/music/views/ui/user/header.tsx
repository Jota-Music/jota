import { useQuery } from "@tanstack/preact-query";
import { User as UserIcon } from "lucide-preact";
import { Link } from "wouter-preact";
import type { UserSource } from "@/lib/music/app/get-user-playlists";
import getUserProfile from "@/lib/music/app/get-user-profile";
import { FollowButton } from "@/lib/music/views/ui/follow";
import { t } from "@/lib/shared/i18n";
import { cover } from "@/lib/shared/utils/cover";
import { Image } from "@/lib/shared/views/ui/components/image";
import YoutubeIcon from "@/lib/shared/views/ui/icons/youtube";

export function UserHeader({
	source,
	identifier,
	name: fallbackName,
	imageUrl: fallbackImage,
}: {
	source: UserSource;
	identifier: string;
	name?: string;
	imageUrl?: string;
}) {
	const spotify = source === "spotify";

	const profile = useQuery({
		queryKey: ["user-profile", identifier],
		queryFn: () => getUserProfile(identifier),
		enabled: spotify && !!identifier,
	});

	const followKey = spotify ? identifier : `youtube:${identifier}`;

	const name = profile.data?.displayName || fallbackName || identifier;
	const imageUrl = profile.data?.imageUrl || fallbackImage;
	const handle = identifier.startsWith("@") ? identifier : `@${identifier}`;
	// A raw channel ID or URL is not a handle, so don't show it as one.
	const showHandle = spotify
		? name !== identifier
		: identifier.startsWith("@") && name !== handle;

	return (
		<header class="flex shrink-0 items-center gap-4">
			{imageUrl ? (
				<Image
					src={cover(imageUrl, 128)}
					alt={name}
					class="h-16 w-16 shrink-0 rounded-full object-cover"
				/>
			) : (
				<div class="flex h-16 w-16 shrink-0 items-center justify-center rounded-full bg-zinc-900 text-zinc-500">
					<UserIcon size={26} />
				</div>
			)}

			{/* flex-1 takes the leftover width so the actions stay flush right
			    and the name still truncates instead of pushing them off. */}
			<div class="flex min-w-0 flex-1 flex-col">
				<h2 class="truncate text-xl font-semibold leading-tight">{name}</h2>
				{showHandle && <p class="truncate text-sm text-zinc-500">{handle}</p>}
			</div>

			{spotify && (
				<Link
					href={`/search/youtube/${encodeURIComponent(name)}`}
					title={t("pages.user.findOnYouTube")}
					class="flex h-8 w-8 cursor-pointer items-center justify-center rounded-md border border-zinc-800 bg-zinc-950 text-zinc-400 transition-colors hover:text-white"
				>
					<YoutubeIcon class="size-4" />
				</Link>
			)}
			<FollowButton id={followKey} />
		</header>
	);
}
