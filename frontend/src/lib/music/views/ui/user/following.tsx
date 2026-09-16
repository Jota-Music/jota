import { useQueries, useQuery } from "@tanstack/preact-query";
import { Mic2, User as UserIcon } from "lucide-preact";
import getFollowing from "@/lib/music/app/get-following";
import getFriends from "@/lib/music/app/get-friends";
import getUserProfile from "@/lib/music/app/get-user-profile";
import { type IconType, type Item, Shelf } from "@/lib/music/views/ui/shelf";
import { useFollows } from "@/lib/music/views/ui/user/follow";

type Entry = {
	id: string;
	name: string;
	cover?: string;
	removable: boolean;
	icon: IconType;
	user?: string;
};

function link(id: string): string {
	return id.startsWith("artist:")
		? `/artist/${id.slice("artist:".length)}`
		: `/${id.slice("user:".length)}`;
}

// Following merges the account's own follows with the friends stored locally,
// so the shelf is populated on login without the user adding anything.
export function FollowingShelf({ account }: { account: string }) {
	const {
		users: followed,
		isLoading: followedLoading,
		unfollow,
	} = useFollows(account);

	const friends = useQuery({
		queryKey: ["friends", account],
		queryFn: getFriends,
		enabled: !!account,
	});

	const following = useQuery({
		queryKey: ["following", account],
		queryFn: getFollowing,
		enabled: !!account,
	});

	// Local follows are removable; Spotify follows and friends are read-only.
	const entries: Entry[] = [];
	const seen = new Set<string>();

	function addUser(user: string, removable: boolean) {
		const key = `user:${user}`;
		if (seen.has(key.toLowerCase())) return;
		seen.add(key.toLowerCase());
		entries.push({
			id: key,
			name: user,
			removable,
			icon: UserIcon,
			user,
		});
	}

	for (const user of followed) addUser(user, true);

	for (const follow of following.data ?? []) {
		const key = `${follow.kind}:${follow.id}`;
		if (seen.has(key.toLowerCase())) continue;
		seen.add(key.toLowerCase());
		entries.push({
			id: key,
			name: follow.name || follow.id,
			cover: follow.imageUrl,
			removable: false,
			icon: follow.kind === "artist" ? Mic2 : UserIcon,
			user: follow.kind === "user" ? follow.id : undefined,
		});
	}

	for (const user of friends.data ?? []) addUser(user, false);

	const profileTargets = entries.filter((entry) => entry.user && !entry.cover);
	const profiles = useQueries({
		queries: profileTargets.map((entry) => ({
			queryKey: ["user-profile", entry.user],
			queryFn: () => getUserProfile(entry.user ?? ""),
		})),
	});
	const profileByUser = new Map(
		profileTargets.map((entry, index) => [
			(entry.user ?? "").toLowerCase(),
			profiles[index]?.data,
		]),
	);

	const items: Item[] = entries.map((entry) => {
		const profile = entry.user
			? profileByUser.get(entry.user.toLowerCase())
			: undefined;
		return {
			id: entry.id,
			name: profile?.displayName || entry.name,
			cover: entry.cover ?? profile?.imageUrl,
			removable: entry.removable,
			icon: entry.icon,
		};
	});

	return (
		<Shelf
			items={items}
			to={link}
			isLoading={followedLoading || friends.isLoading || following.isLoading}
			emptyMessage="No friends or followed accounts yet. Open a user profile and tap the heart to follow it."
			onRemove={(id) => {
				if (id.startsWith("user:")) unfollow.mutate(id.slice("user:".length));
			}}
		/>
	);
}
