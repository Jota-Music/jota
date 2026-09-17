import { GetUserProfile } from "@bindings/app";
import type { UserProfile } from "@/lib/music/model";

export default async function getUserProfile(
	username: string,
): Promise<UserProfile> {
	return GetUserProfile(username);
}
