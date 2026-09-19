import { useQuery } from "@tanstack/preact-query";
import { Heart, User as UserIcon } from "lucide-preact";
import { Link } from "wouter-preact";
import { spotifyUser } from "@/lib/auth/views/stores/session";
import type { UserSource } from "@/lib/music/app/get-user-playlists";
import getUserProfile from "@/lib/music/app/get-user-profile";
import { useFollows } from "@/lib/music/views/ui/user/follow";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
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
	const account = spotifyUser.value ?? "";

	const profile = useQuery({
		queryKey: ["user-profile", identifier],
		queryFn: () => getUserProfile(identifier),
		enabled: spotify && !!identifier,
	});

	const { users, follow, unfollow } = useFollows(account);

	const isFollowing = users.some(
		(u) => u.toLowerCase() === identifier.toLowerCase(),
	);

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
				<img
					src={imageUrl}
					alt={name}
					class="h-16 w-16 shrink-0 rounded-full object-cover"
				/>
			) : (
				<div class="flex h-16 w-16 shrink-0 items-center justify-center rounded-full bg-zinc-900 text-zinc-500">
					<UserIcon size={26} />
				</div>
			)}

			<div class="flex min-w-0 flex-col">
				<h2 class="truncate text-xl font-semibold leading-tight">{name}</h2>
				{showHandle && <p class="truncate text-sm text-zinc-500">{handle}</p>}
			</div>

			{spotify && (
				<>
					<Link
						href={`/search/youtube/${encodeURIComponent(name)}`}
						title={t("pages.user.findOnYouTube")}
						class="flex h-8 w-8 cursor-pointer items-center justify-center rounded-md border border-zinc-800 bg-zinc-950 text-zinc-400 transition-colors hover:text-white"
					>
						<YoutubeIcon class="size-4" />
					</Link>
					<button
						type="button"
						onClick={() => (isFollowing ? unfollow : follow).mutate(identifier)}
						disabled={!account || follow.isPending || unfollow.isPending}
						aria-label={
							isFollowing ? t("pages.user.unfollow") : t("pages.user.follow")
						}
						class={cn(
							"flex h-8 w-8 cursor-pointer items-center justify-center rounded-md border border-zinc-800 bg-zinc-950 hover:text-white disabled:opacity-50",
							isFollowing ? "text-red-400" : "text-zinc-400",
						)}
					>
						<Heart size={14} class={isFollowing ? "fill-current" : ""} />
					</button>
				</>
			)}
		</header>
	);
}
