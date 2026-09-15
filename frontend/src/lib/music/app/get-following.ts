import { GetFollowing } from "@bindings/app";
import type { Follow } from "@/lib/music/model";

export default async function getFollowing(): Promise<Follow[]> {
	const follows = (await GetFollowing()) as unknown as Follow[] | null;
	return follows ?? [];
}
