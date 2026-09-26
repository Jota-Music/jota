import { useMutation, useQuery, useQueryClient } from "@tanstack/preact-query";
import { Heart } from "lucide-preact";
import {
	followUser,
	getFollowedUsers,
	unfollowUser,
} from "@/lib/music/app/follows";
import { t } from "@/lib/shared/i18n";
import type { Action } from "@/lib/shared/views/ui/components/context-menu";

// The store dedupes case-insensitively, so the read has to match.
export const follows = (users: string[], id: string) =>
	users.some((u) => u.toLowerCase() === id.toLowerCase());

export function useFollows(account: string) {
	const queryClient = useQueryClient();
	const key = ["followed-users", account];

	const query = useQuery({
		queryKey: key,
		queryFn: () => getFollowedUsers(account),
		enabled: !!account,
	});

	const invalidate = () => queryClient.invalidateQueries({ queryKey: key });

	const follow = useMutation({
		mutationFn: (user: string) => followUser(account, user),
		onSuccess: invalidate,
	});

	const unfollow = useMutation({
		mutationFn: (user: string) => unfollowUser(account, user),
		onSuccess: invalidate,
	});

	const users = query.data ?? [];
	const toggle = (id: string): Action => {
		const on = follows(users, id);
		return {
			icon: Heart,
			label: on ? t("music.unfollow") : t("music.follow"),
			run: () => (on ? unfollow : follow).mutate(id),
		};
	};

	return {
		users,
		isLoading: query.isLoading,
		follow,
		unfollow,
		toggle,
	};
}
