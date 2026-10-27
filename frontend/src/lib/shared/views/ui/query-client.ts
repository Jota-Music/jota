import { QueryClient } from "@tanstack/preact-query";

export const queryClient = new QueryClient({
	defaultOptions: {
		queries: {
			staleTime: 5 * 60 * 1000,
			gcTime: 5 * 60 * 1000,
			refetchOnWindowFocus: false,
		},
	},
});
