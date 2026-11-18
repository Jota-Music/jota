import { useQueries, useQuery, useQueryClient } from "@tanstack/preact-query";
import { Mic2, User as UserIcon } from "lucide-preact";
import { spotifyReady } from "@/lib/auth/views/stores/session";
import { artistFollow, parseFollow, userFollow } from "@/lib/music/app/follows";
import { getArtist } from "@/lib/music/app/get-artist";
import { getChannelInfo } from "@/lib/music/app/get-channel";
import getFollowing from "@/lib/music/app/get-following";
import getFriends from "@/lib/music/app/get-friends";
import getUserProfile from "@/lib/music/app/get-user-profile";
import { FollowButton } from "@/lib/music/views/ui/follow";
import {
	FollowingHint,
	type IconType,
	type Item,
	Shelf,
	type Source,
} from "@/lib/music/views/ui/shelf";
import { useFollows } from "@/lib/music/views/ui/user/follow";
import { getOrder, saveOrder } from "@/lib/shared/app/order";
import { applyOrder, move } from "@/lib/shared/app/reorder";
import YoutubeIcon from "@/lib/shared/views/ui/icons/youtube";

type Entry = {
	id: string;
	name: string;
	cover?: string;
	icon: IconType;
	user?: string;
	channel?: string;
	artist?: string;
	// follow is the key this entry has in the local follows store, so its menu
	// can offer the same unfollow the hover button does. Only local follows
	// have one: the Spotify list is read-only and cannot be unwritten.
	follow?: string;
	source?: Source;
};

function link(id: string): string {
	const target = parseFollow(id);
	if (target.channel) return `/youtube/user/${target.channel}`;
	if (target.artist) return `/artist/${target.artist}`;
	return `/spotify/user/${target.user ?? ""}`;
}

// Following merges the account's own follows with the friends stored locally,
// so the shelf is populated on login without the user adding anything.
export function FollowingShelf({ account }: { account: string }) {
	const queryClient = useQueryClient();
	const { users: followed, isLoading: followedLoading, toggle } = useFollows();

	// Same rule as the library's sources: a query that failed holds no data, so
	// a remount would ask Spotify again on every route change. The session
	// events are what say when it can answer.
	const friends = useQuery({
		queryKey: ["friends", account],
		queryFn: getFriends,
		enabled: spotifyReady.value && !!account,
		retry: false,
		retryOnMount: false,
	});

	const following = useQuery({
		queryKey: ["following", account],
		queryFn: getFollowing,
		enabled: spotifyReady.value && !!account,
		retry: false,
		retryOnMount: false,
	});

	const orderQuery = useQuery({
		queryKey: ["shelf-order", "following"],
		queryFn: () => getOrder("following", ""),
		enabled: spotifyReady.value,
	});

	// Spotify follows and friends are read-only, so they get no follow key
	// and no heart; only local follows can be unwritten.
	const entries: Entry[] = [];
	const seen = new Set<string>();

	function addUser(user: string) {
		const key = userFollow(user);
		if (seen.has(key.toLowerCase())) return;
		seen.add(key.toLowerCase());
		entries.push({
			id: key,
			name: user,
			icon: UserIcon,
			user,
		});
	}

	for (const user of followed) {
		const target = parseFollow(user);
		if (seen.has(target.key.toLowerCase())) continue;
		seen.add(target.key.toLowerCase());
		entries.push({
			id: target.key,
			name: target.artist ?? target.channel ?? target.user ?? "",
			icon: target.channel ? YoutubeIcon : target.artist ? Mic2 : UserIcon,
			follow: user,
			...(target.channel
				? { channel: target.channel, source: "youtube" as const }
				: {}),
			...(target.artist ? { artist: target.artist } : {}),
			...(target.user ? { user: target.user } : {}),
		});
	}

	for (const follow of following.data ?? []) {
		const key =
			follow.kind === "artist"
				? artistFollow(follow.id)
				: userFollow(follow.id);
		if (seen.has(key.toLowerCase())) continue;
		seen.add(key.toLowerCase());
		entries.push({
			id: key,
			name: follow.name || follow.id,
			cover: follow.imageUrl,
			icon: follow.kind === "artist" ? Mic2 : UserIcon,
			user: follow.kind === "user" ? follow.id : undefined,
			source: "spotify",
		});
	}

	for (const user of friends.data ?? []) addUser(user);

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

	const channelTargets = entries.filter(
		(entry) => entry.channel && !entry.cover,
	);
	const infos = useQueries({
		queries: channelTargets.map((entry) => ({
			queryKey: ["channel-info", entry.channel],
			queryFn: () => getChannelInfo(entry.channel ?? ""),
		})),
	});
	const infoByChannel = new Map(
		channelTargets.map((entry, index) => [
			(entry.channel ?? "").toLowerCase(),
			infos[index]?.data,
		]),
	);

	const artistTargets = entries.filter((entry) => entry.artist && !entry.cover);
	const artists = useQueries({
		queries: artistTargets.map((entry) => ({
			queryKey: ["artist", entry.artist],
			queryFn: () => getArtist(entry.artist ?? ""),
		})),
	});
	const artistById = new Map(
		artistTargets.map((entry, index) => [
			(entry.artist ?? "").toLowerCase(),
			artists[index]?.data,
		]),
	);

	// The table maps a shelf id to the key the follows store holds, so the menu
	// and the hover button unfollow the same record.
	const followById = new Map(
		entries.flatMap((entry) =>
			entry.follow ? [[entry.id.toLowerCase(), entry.follow] as const] : [],
		),
	);

	const unsorted: Item[] = entries.map((entry) => {
		const profile = entry.user
			? profileByUser.get(entry.user.toLowerCase())
			: undefined;
		const info = entry.channel
			? infoByChannel.get(entry.channel.toLowerCase())
			: undefined;
		const artist = entry.artist
			? artistById.get(entry.artist.toLowerCase())
			: undefined;
		return {
			id: entry.id,
			name: profile?.displayName || info?.name || artist?.name || entry.name,
			cover:
				entry.cover ?? profile?.imageUrl ?? info?.avatar ?? artist?.imageUrl,
			icon: entry.icon,
			menu: entry.follow ? [toggle(entry.follow)] : [],
			source: entry.source,
		};
	});

	const items = applyOrder(unsorted, orderQuery.data ?? []);

	const reorder = (fromId: string, toId: string) => {
		const next = move(
			items.map((item) => item.id),
			fromId,
			toId,
		);
		queryClient.setQueryData(["shelf-order", "following"], next);
		void saveOrder("following", "", next);
	};

	// Same rule as the library: the shelf paints what it has and no loader sits
	// over it, so the empty state waits until every source has answered.
	const pending =
		!spotifyReady.value ||
		followedLoading ||
		friends.isLoading ||
		following.isLoading ||
		orderQuery.isLoading;

	return (
		<Shelf
			items={items}
			to={link}
			viewKey="following_view"
			isLoading={pending && items.length === 0}
			emptyMessage={pending ? null : <FollowingHint />}
			showLocal={false}
			onReorder={reorder}
			// Everything on this shelf is a follow, so the hover control is the
			// heart itself instead of a generic remove button.
			leading={(id) => {
				const key = followById.get(id.toLowerCase());
				return key ? <FollowButton id={key} variant="cover" /> : null;
			}}
		/>
	);
}
