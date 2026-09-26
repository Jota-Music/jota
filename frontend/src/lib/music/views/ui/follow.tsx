import { Heart } from "lucide-preact";
import { spotifyUser } from "@/lib/auth/views/stores/session";
import { saveLabel, useSaved } from "@/lib/music/views/ui/playlist/saved";
import { useFollows } from "@/lib/music/views/ui/user/follow";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";

// One heart style for every followable thing. cn only joins classes, so the
// state colours must never overlap with the variant's own: the winner would be
// the CSS source order, not this order. The cover sits on artwork and needs a
// scrim; toolbar sits on the page background, next to the other plain actions.
const base = {
	cover:
		"flex h-8 w-8 cursor-pointer items-center justify-center rounded-md bg-black/70 transition hover:bg-black/90 disabled:opacity-50",
	toolbar:
		"flex h-8 w-8 cursor-pointer items-center justify-center rounded-md border border-zinc-800 bg-zinc-950 transition hover:text-white disabled:opacity-50",
} as const;

type Variant = keyof typeof base;

const idle = "text-zinc-400";
const following = "text-red-400 hover:text-red-300";

function HeartButton({
	on,
	label,
	variant,
	disabled,
	onClick,
}: {
	on: boolean;
	label: string;
	variant: Variant;
	disabled?: boolean;
	onClick: () => void;
}) {
	return (
		<button
			type="button"
			title={label}
			aria-label={label}
			aria-pressed={on}
			disabled={disabled}
			onClick={(e) => {
				e.preventDefault();
				e.stopPropagation();
				onClick();
			}}
			class={cn(base[variant], on ? following : idle)}
		>
			<Heart size={14} class={on ? "fill-current" : ""} />
		</button>
	);
}

// FollowButton follows a user, a YouTube channel or a Spotify artist. The key is
// already prefixed, so one button covers every entity the local follows store
// can hold. It is local only: Spotify's own following list is read-only.
export function FollowButton({
	id,
	variant = "toolbar",
}: {
	id: string;
	variant?: Variant;
}) {
	const account = spotifyUser.value ?? "";
	const { users, follow, unfollow } = useFollows(account);

	const on = users.some((u) => u.toLowerCase() === id.toLowerCase());
	const label = on ? t("music.unfollow") : t("music.follow");

	return (
		<HeartButton
			on={on}
			label={label}
			variant={variant}
			disabled={!account || follow.isPending || unfollow.isPending}
			onClick={() => (on ? unfollow : follow).mutate(id)}
		/>
	);
}

// SaveButton keeps an external playlist in the library, from either source. It
// pairs with the context menu entry built from the same useSaved hook.
export function SaveButton({
	id,
	variant = "cover",
}: {
	id: string;
	variant?: Variant;
}) {
	const { isSaved, follow } = useSaved();
	const saved = isSaved(id);

	return (
		<HeartButton
			on={saved}
			label={saveLabel(saved)}
			variant={variant}
			onClick={() => follow(id, !saved)}
		/>
	);
}
