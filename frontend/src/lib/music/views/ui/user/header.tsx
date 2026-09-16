import { useQuery } from "@tanstack/preact-query";
import { Heart, User as UserIcon } from "lucide-preact";
import { spotifyUser } from "@/lib/auth/views/stores/session";
import getUserProfile from "@/lib/music/app/get-user-profile";
import { useFollows } from "@/lib/music/views/ui/user/follow";

export function UserHeader({ username }: { username: string }) {
	const account = spotifyUser.value ?? "";

	const profile = useQuery({
		queryKey: ["user-profile", username],
		queryFn: () => getUserProfile(username),
		enabled: !!username,
	});

	const { users, follow, unfollow } = useFollows(account);

	const isFollowing = users.some(
		(u) => u.toLowerCase() === username.toLowerCase(),
	);

	const name = profile.data?.displayName || username;

	return (
		<header class="flex shrink-0 items-center gap-4">
			{profile.data?.imageUrl ? (
				<img
					src={profile.data.imageUrl}
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
				{name !== username && (
					<p class="truncate text-sm text-zinc-500">@{username}</p>
				)}
			</div>

			<button
				type="button"
				onClick={() => (isFollowing ? unfollow : follow).mutate(username)}
				disabled={!account || follow.isPending || unfollow.isPending}
				aria-label={isFollowing ? "Unfollow user" : "Follow user"}
				class={`flex h-8 w-8 cursor-pointer items-center justify-center rounded-md border border-zinc-800 bg-zinc-950 hover:text-white disabled:opacity-50 ${
					isFollowing ? "text-red-400" : "text-zinc-400"
				}`}
			>
				<Heart size={14} class={isFollowing ? "fill-current" : ""} />
			</button>
		</header>
	);
}
