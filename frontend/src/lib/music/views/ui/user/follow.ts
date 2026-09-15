import { useMutation, useQuery, useQueryClient } from "@tanstack/preact-query";
import {
	followUser,
	getFollowedUsers,
	unfollowUser,
} from "@/lib/music/app/follows";

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

	return {
		users: query.data ?? [],
		isLoading: query.isLoading,
		follow,
		unfollow,
	};
}
