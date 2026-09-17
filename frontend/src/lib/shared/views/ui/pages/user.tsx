import { useQuery, useQueryClient } from "@tanstack/preact-query";
import { RefreshCw } from "lucide-preact";
import { useParams } from "wouter-preact";
import getUserPlaylists, {
	revalidateUserPlaylists,
} from "@/lib/music/app/get-user-playlists";
import { type Item, Shelf } from "@/lib/music/views/ui/shelf";
import { UserHeader } from "@/lib/music/views/ui/user/header";
import { t } from "@/lib/shared/i18n";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";

export function UserPage() {
	const { user } = useParams<{ user: string }>();

	const queryClient = useQueryClient();

	const { data, isLoading, isError } = useQuery({
		queryKey: ["user-playlists", user],
		queryFn: () => getUserPlaylists(user ?? ""),
		enabled: !!user,
	});

	async function handleRefresh() {
		if (user) await revalidateUserPlaylists(user).catch(() => {});
		await queryClient.invalidateQueries({ queryKey: ["user-playlists", user] });
	}

	const playlists = data ?? [];

	if (isError) {
		return (
			<DefaultLayout>
				<div class="flex flex-col items-start gap-3 p-6 text-sm">
					<p class="text-red-400">
						{t("pages.user.failed", { user: user ?? "" })}
					</p>
					<button
						type="button"
						onClick={handleRefresh}
						title={t("common.retry")}
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
				<UserHeader username={user} />

				<Shelf
					items={playlists.map(
						(p): Item => ({
							id: p.id,
							name: p.name,
							cover: p.cover,
							subtitle: p.owner,
						}),
					)}
					to={(id) => `/playlist/${id}`}
					viewKey="user_view"
					isLoading={isLoading}
					emptyMessage={t("pages.user.noPlaylists")}
				/>
			</div>
		</DefaultLayout>
	);
}
