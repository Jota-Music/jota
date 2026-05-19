import { signal } from "@preact/signals";
import { useQuery, useQueryClient } from "@tanstack/preact-query";
import { Loader, X } from "lucide-preact";
import { Link, useLocation } from "wouter-preact";
import getUserPlaylists, {
	type PlaylistSummary,
} from "@/lib/music/app/get-user-playlists";
import { followedUsers } from "@/lib/shared/views/stores/follows";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";
import useMeta from "@/lib/shared/views/hooks/use-meta";

const inputValue = signal("");

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
							queryClient.fetchQuery({ queryKey: ["user-playlists", user], queryFn: () => getUserPlaylists(user, true) });
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

export function MainPage() {
	const [, setLocation] = useLocation();
	useMeta(
		"Jota | Free music self-hosted service",
		"Jota is a free, self-hosted music service. Stream your music anywhere, anytime.",
	);

	const handleSubmit = (e: Event) => {
		e.preventDefault();
		const value = inputValue.value.trim();
		if (value) {
			setLocation(`/${value}`);
		}
	};

	return (
		<DefaultLayout class="gap-6 h-full">
			<div class="flex flex-col gap-6 h-full">
				<header>
					<h2 class="text-xl font-semibold leading-tight">Jota</h2>
					<p class="text-sm opacity-70 mt-1">
						Free music self-hosted service
					</p>
				</header>

				<form onSubmit={handleSubmit} class="flex gap-2">
					<div class="relative flex-1">
						<input
							type="text"
							placeholder="Enter a username..."
							value={inputValue.value}
							onInput={(e) => {
								inputValue.value = (e.target as HTMLInputElement).value;
							}}
							class="h-10 w-full rounded-md border border-zinc-800 bg-zinc-950 pr-9 pl-3 text-sm text-white outline-none focus:border-zinc-600"
						/>
						{inputValue.value && (
							<button
								type="button"
								onClick={() => {
									inputValue.value = "";
								}}
								class="absolute right-2 top-1/2 -translate-y-1/2 flex h-5 w-5 cursor-pointer items-center justify-center rounded text-zinc-500 hover:text-white hover:bg-zinc-800"
								aria-label="Clear search"
							>
								<X size={14} />
							</button>
						)}
					</div>
					<button
						type="submit"
						class="rounded-lg px-4 py-2 font-bold text-sm text-(--binary-color) bg-(--dominant-color) hover:opacity-75 transition-opacity cursor-pointer"
					>
						Go
					</button>
				</form>

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