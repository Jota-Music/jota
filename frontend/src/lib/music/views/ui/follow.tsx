import { Heart } from "lucide-preact";
import { saveLabel, useSaved } from "@/lib/music/views/ui/playlist/saved";
import { follows, useFollows } from "@/lib/music/views/ui/user/follow";
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
		"flex h-10 w-10 md:h-8 md:w-8 cursor-pointer items-center justify-center rounded-md transition disabled:opacity-50",
} as const;

type Variant = keyof typeof base;

// The heart has one look wherever a follow is shown: red and filled when it is
// on, grey and hollow when it is off. The tone sits on the wrapper, so a button
// can still shade it on hover.
export const heartOn = "text-red-400";
const heartOff = "text-zinc-400";

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
			<Heart class={cn("size-4 md:size-3.5", on && "fill-current")} />
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
	const { users, follow, unfollow } = useFollows();

	const on = follows(users, id);
	const label = on ? t("music.unfollow") : t("music.follow");

	return (
		<HeartButton
			on={on}
			label={label}
			variant={variant}
			disabled={follow.isPending || unfollow.isPending}
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
