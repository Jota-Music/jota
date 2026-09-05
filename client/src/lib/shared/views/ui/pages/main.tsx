import { signal } from "@preact/signals";
import { useQuery, useQueryClient } from "@tanstack/preact-query";
import { Loader, X, Search, ChevronDown } from "lucide-preact";
import { useLayoutEffect } from "preact/hooks";
import { Link, useLocation, useRoute } from "wouter-preact";
import {
	spotifyConnected,
	spotifyUser,
	syncSpotifyStatus,
} from "@/lib/auth/views/stores/session";
import getUserPlaylists, {
	type PlaylistSummary,
} from "@/lib/music/app/get-user-playlists";
import { homeRoomEnforced, joinRoomById } from "@/lib/shared/api/room";
import useMeta from "@/lib/shared/views/hooks/use-meta";
import { followedUsers } from "@/lib/shared/views/stores/follows";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";

const inputValue = signal("");
const searchType = signal<"user" | "track" | "album" | "playlist" | "artist">("user");

const searchTypeOptions = [
	{ value: "user", label: "User" },
	{ value: "track", label: "Track" },
	{ value: "album", label: "Album" },
	{ value: "playlist", label: "Playlist" },
	{ value: "artist", label: "Artist" },
] as const;

function FollowedUserSection({ user }: { user: string }) {
	const queryClient = useQueryClient();
	const { data, isLoading, isError } = useQuery({
		queryKey: ["user-playlists", user],
		queryFn: () => getUserPlaylists(user),
	});

	if (isError) {
		return (
			<section class="flex flex-col gap-3">
				<div class="flex items-center gap-2">
					<span class="text-base font-semibold text-zinc-500">@{user}</span>
					<button
						type="button"
						onClick={() => {
							queryClient.removeQueries({ queryKey: ["user-playlists", user] });
							queryClient.fetchQuery({
								queryKey: ["user-playlists", user],
								queryFn: () => getUserPlaylists(user, true),
							});
						}}
						class="text-xs text-red-400 hover:text-red-300 transition-colors cursor-pointer"
					>
						Retry
					</button>
				</div>
				<p class="text-xs text-red-400">Failed to load playlists</p>
			</section>
		);
	}

	const playlists = data ?? [];

	return (
		<section class="flex flex-col gap-3">
			<div class="flex items-center gap-2">
				<Link
					href={`/${user}`}
					class="text-base font-semibold text-zinc-300 hover:text-white transition-colors"
				>
					@{user}
				</Link>
			</div>

			{isLoading ? (
				<div class="flex items-center gap-2 text-sm text-zinc-500">
					<Loader size={14} class="animate-spin" />
					Loading playlists...
				</div>
			) : playlists.length === 0 ? (
				<p class="text-sm text-zinc-500">No playlists found.</p>
			) : (
				<div class="grid grid-cols-2 lg:grid-cols-3 gap-3 md:gap-4">
					{playlists.map((p: PlaylistSummary) => (
						<Link key={p.id} href={`/playlist/${p.id}`}>
							<div class="rounded-md overflow-hidden cursor-pointer group">
								<div class="flex flex-col gap-2">
									<div class="aspect-square w-full overflow-hidden rounded-md">
										<img
											src={"mosaic" in p ? p.mosaic : p.cover}
											alt={p.name}
											class="w-full h-full object-cover group-hover:opacity-80 transition-opacity"
										/>
									</div>
									<h3 class="font-medium truncate text-sm text-zinc-300 group-hover:text-white transition-colors">
										{p.name}
									</h3>
								</div>
							</div>
						</Link>
					))}
				</div>
			)}
		</section>
	);
}

function MyPlaylistsSection() {
	const queryClient = useQueryClient();
	const spotifyHandle = spotifyUser.value ?? "default";
	const { data, isLoading, isError } = useQuery({
		queryKey: ["user-playlists", spotifyHandle],
		queryFn: () => getUserPlaylists(spotifyHandle),
	});

	if (isError) {
		return (
			<section class="flex flex-col gap-3">
				<div class="flex items-center gap-2">
					<span class="text-base font-semibold text-zinc-200">
						{spotifyHandle === "default"
							? "Playlists"
							: `@${spotifyHandle}'s playlists`}
					</span>
					<button
						type="button"
						onClick={() => {
							queryClient.removeQueries({
								queryKey: ["user-playlists", spotifyHandle],
							});
							queryClient.fetchQuery({
								queryKey: ["user-playlists", spotifyHandle],
								queryFn: () => getUserPlaylists(spotifyHandle, true),
							});
						}}
						class="text-xs text-red-400 hover:text-red-300 transition-colors cursor-pointer"
					>
						Retry
					</button>
				</div>
				<p class="text-xs text-red-400">Failed to load playlists</p>
			</section>
		);
	}

	const playlists = data ?? [];

	return (
		<section class="flex flex-col gap-3">
			<div class="flex items-center justify-between">
				<span class="text-base font-semibold text-zinc-200">
					{spotifyHandle === "default"
						? "Playlists"
						: `@${spotifyHandle}'s playlists`}
				</span>
				<button
					type="button"
					onClick={() => {
						queryClient.removeQueries({
							queryKey: ["user-playlists", spotifyHandle],
						});
						queryClient.fetchQuery({
							queryKey: ["user-playlists", spotifyHandle],
							queryFn: () => getUserPlaylists(spotifyHandle, true),
						});
					}}
					class="text-xs text-zinc-500 hover:text-zinc-300 transition-colors cursor-pointer"
				>
					Refresh
				</button>
			</div>

			{isLoading ? (
				<div class="flex items-center gap-2 text-sm text-zinc-500">
					<Loader size={14} class="animate-spin" />
					Loading playlists...
				</div>
			) : playlists.length === 0 ? (
				<p class="text-sm text-zinc-500">No playlists found.</p>
			) : (
				<div class="grid grid-cols-2 lg:grid-cols-3 gap-3 md:gap-4">
					{playlists.map((p: PlaylistSummary) => (
						<Link key={p.id} href={`/playlist/${p.id}`}>
							<div class="rounded-md overflow-hidden cursor-pointer group">
								<div class="flex flex-col gap-2">
									<div class="aspect-square w-full overflow-hidden rounded-md">
										<img
											src={"mosaic" in p ? p.mosaic : p.cover}
											alt={p.name}
											class="w-full h-full object-cover group-hover:opacity-80 transition-opacity"
										/>
									</div>
									<h3 class="font-medium truncate text-sm text-zinc-300 group-hover:text-white transition-colors">
										{p.name}
									</h3>
								</div>
							</div>
						</Link>
					))}
				</div>
			)}
		</section>
	);
}

export function MainPage() {
	const [, setLocation] = useLocation();
	const [, joinParams] = useRoute("/join/:room");
	useMeta(
		"Jota | Free music self-hosted service",
		"Jota is a free, self-hosted music service. Stream your music anywhere, anytime.",
	);

	useLayoutEffect(() => {
		void syncSpotifyStatus();
		if (joinParams?.room) {
			homeRoomEnforced.value = false;
			joinRoomById(joinParams.room);
		}
	}, [joinParams?.room]);

	const handleSubmit = (e: Event) => {
		e.preventDefault();
		const value = inputValue.value.trim();
		if (value) {
			const type = searchType.value;
			if (type === "user") {
				setLocation(`/${value}`);
			} else {
				setLocation(`/search/${type}/${value}`);
			}
		}
	};

	return (
		<DefaultLayout class="gap-6 h-full">
			<div class="flex flex-col gap-6 h-full">
				<header>
					<h2 class="text-xl font-semibold leading-tight">Jota</h2>
					<p class="text-sm opacity-70 mt-1">Free music self-hosted music service</p>
				</header>

<form onSubmit={handleSubmit} class="flex gap-2">
					<div class="flex-1 flex items-stretch h-10 rounded-lg border border-zinc-800 bg-zinc-950 overflow-hidden focus-within:border-zinc-600 focus-within:ring-1 focus-within:ring-zinc-600 transition-all">
						<div class="relative shrink-0">
							<select
								value={searchType.value}
								onChange={(e) => {
									searchType.value = (e.target as HTMLSelectElement).value as typeof searchType.value;
								}}
								class="h-full pl-3 pr-7 text-sm text-white outline-none appearance-none cursor-pointer bg-transparent border-r border-zinc-800"
							>
								{searchTypeOptions.map((opt) => (
									<option key={opt.value} value={opt.value} class="bg-zinc-950">
										{opt.label}
									</option>
								))}
							</select>
							<ChevronDown class="absolute right-2 top-1/2 -translate-y-1/2 size-3.5 text-zinc-500 pointer-events-none" />
						</div>
						<div class="relative flex-1 flex items-center">
							<Search class="absolute left-3 size-4 text-zinc-500 pointer-events-none" />
							<input
								type="text"
								placeholder={searchType.value === "user" ? "Spotify username..." : `Spotify ${searchType.value.charAt(0).toUpperCase() + searchType.value.slice(1)} ID or URI...`}
								value={inputValue.value}
								onInput={(e) => {
									inputValue.value = (e.target as HTMLInputElement).value;
								}}
								class="h-full w-full bg-transparent pl-9 pr-9 text-sm text-white outline-none placeholder:text-zinc-600"
							/>
							{inputValue.value && (
								<button
									type="button"
									onClick={() => {
										inputValue.value = "";
									}}
									class="absolute right-2 flex h-6 w-6 cursor-pointer items-center justify-center rounded text-zinc-500 hover:text-white hover:bg-zinc-800 transition-colors"
									aria-label="Clear search"
								>
									<X size={14} />
								</button>
							)}
						</div>
					</div>
					<button
						type="submit"
						class="h-10 w-10 rounded-lg px-3 py-2 text-(--binary-color) bg-(--dominant-color) hover:opacity-75 transition-opacity cursor-pointer flex items-center justify-center"
						title="Go"
					>
						<Search class="size-4" />
					</button>
				</form>

				{spotifyConnected.value && <MyPlaylistsSection />}

				{followedUsers.value.length > 0 && (
					<div class="flex flex-col gap-6">
						<h3 class="text-sm font-semibold text-zinc-400 uppercase tracking-wider">
							Following
						</h3>
						{followedUsers.value.map((user) => (
							<FollowedUserSection key={user} user={user} />
						))}
					</div>
				)}
			</div>
		</DefaultLayout>
	);
}

export default MainPage;