import { signal } from "@preact/signals";
import { useQuery, useQueryClient } from "@tanstack/preact-query";
import { RefreshCw } from "lucide-preact";
import { useParams } from "wouter-preact";
import getUserPlaylists from "@/lib/music/app/get-user-playlists";
import { type Item, Shelf } from "@/lib/music/views/ui/shelf";
import { UserHeader } from "@/lib/music/views/ui/user/header";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";

export function UserPage() {
	const { user } = useParams<{ user: string }>();

	const queryClient = useQueryClient();
	const refreshing = signal(false);

	const { data, isLoading, isError } = useQuery({
		queryKey: ["user-playlists", user],
		queryFn: () => getUserPlaylists(user ?? ""),
		enabled: !!user,
	});

	async function handleRefresh() {
		refreshing.value = true;
		queryClient.removeQueries({ queryKey: ["user-playlists", user] });
		await queryClient.fetchQuery({
			queryKey: ["user-playlists", user],
			queryFn: () => getUserPlaylists(user ?? ""),
		});
		refreshing.value = false;
	}

	const playlists = data ?? [];

	if (isError) {
		return (
			<DefaultLayout>
				<div class="flex flex-col items-start gap-3 p-6 text-sm">
					<p class="text-red-400">Failed to load playlists for @{user}</p>
					<button
						type="button"
						onClick={handleRefresh}
						title="Retry"
						class="flex h-8 w-8 cursor-pointer items-center justify-center rounded-md border border-zinc-800 bg-zinc-950 text-zinc-400 hover:text-white transition-colors"
					>
						<RefreshCw size={14} />
					</button>
				</div>
			</DefaultLayout>
		);
	}

	return (
		<DefaultLayout class="gap-6">
			<div class="flex flex-col gap-6 min-h-0 flex-1">
				<UserHeader
					username={user ?? ""}
					actions={
						<button
							type="button"
							onClick={handleRefresh}
							disabled={refreshing.value}
							aria-label="Refresh playlists"
							class="flex h-8 w-8 cursor-pointer items-center justify-center rounded-md border border-zinc-800 bg-zinc-950 text-zinc-400 hover:text-white disabled:opacity-50"
						>
							<RefreshCw
								size={14}
								class={refreshing.value ? "animate-spin" : ""}
							/>
						</button>
					}
				/>

				<Shelf
					items={playlists.map(
						(p): Item => ({
							id: p.id,
							name: p.name,
							cover: p.cover ?? p.mosaic,
						}),
					)}
					to={(id) => `/playlist/${id}`}
					isLoading={isLoading}
					emptyMessage="No playlists found."
				/>
			</div>
		</DefaultLayout>
	);
}
