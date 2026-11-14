import { Loader, Pause, Play } from "lucide-preact";
import type { ComponentChildren } from "preact";
import { Link } from "wouter-preact";
import { t } from "@/lib/shared/i18n";
import { cover as coverImage } from "@/lib/shared/utils/cover";
import { CoverFallback } from "@/lib/shared/views/ui/components/cover-fallback";
import { Image } from "@/lib/shared/views/ui/components/image";
import { Mosaic } from "@/lib/shared/views/ui/components/mosaic";

function LinkLine({ link }: { link: { to: string; label: string } }) {
	return (
		<Link
			to={link.to}
			class="mt-px block truncate text-sm leading-tight opacity-70 hover:opacity-100 hover:underline"
		>
			{link.label}
		</Link>
	);
}

export function PageHeader({
	cover,
	covers,
	title,
	subtitle,
	link,
	pending = false,
	beforePlay,
	onPlay,
	playing,
	loading,
	actions,
}: {
	cover?: string;
	covers?: string[];
	title: string;
	subtitle?: string;
	link?: { to: string; label: string };
	// pending keeps the header frame and swaps its text for bars, so the rows
	// below do not jump when the response lands.
	pending?: boolean;
	// beforePlay sits left of the play button, inside the same cluster.
	beforePlay?: ComponentChildren;
	onPlay?: () => void;
	playing?: boolean;
	loading?: boolean;
	actions?: ComponentChildren;
}) {
	const right =
		beforePlay || onPlay || actions ? (
			<div class="ml-auto flex shrink-0 items-center gap-2">
				{beforePlay}
				{/* The play slot is reserved whenever the cluster exists, so the
				    header does not shift once the tracks land. */}
				<button
					type="button"
					title={t("music.playAll")}
					aria-label={t("music.playAll")}
					onClick={onPlay}
					disabled={loading || !onPlay}
					class="flex h-10 w-10 shrink-0 cursor-pointer items-center justify-center rounded-full bg-(--dominant-color) text-(--binary-color) transition disabled:cursor-default disabled:opacity-70"
				>
					{loading ? (
						<Loader class="size-5 animate-spin" />
					) : playing ? (
						<Pause class="size-5 fill-current" />
					) : (
						<Play class="size-5 fill-current" />
					)}
				</button>
				{actions}
			</div>
		) : null;

	return (
		<header class="flex shrink-0 items-center gap-4 px-3 pt-4 md:px-0 md:pt-0">
			{pending ? (
				<div class="shimmer h-16 w-16 shrink-0 rounded-md" aria-hidden />
			) : covers?.length ? (
				<div class="h-16 w-16 shrink-0 overflow-hidden rounded-md">
					<Mosaic covers={covers} size={128} placeholder={<CoverFallback />} />
				</div>
			) : (
				<Image
					src={coverImage(cover, 128)}
					alt=""
					class="h-16 w-16 shrink-0 rounded-md object-cover"
					fallback={<CoverFallback class="h-16 w-16 shrink-0 rounded-md" />}
				/>
			)}
			{/* Same three lines on every page: the title, the count and the link
			    row always hold their box, so the text starts at the cover's top
			    and no page shifts the rows below it. */}
			<div class="flex h-16 min-w-0 flex-col overflow-hidden">
				<h2 class="truncate text-xl font-semibold leading-tight">
					{pending ? (
						<span
							class="shimmer inline-block h-[1em] w-40 rounded align-middle"
							aria-hidden
						/>
					) : (
						title
					)}
				</h2>
				<p class="mt-px truncate text-sm leading-tight opacity-70">
					{pending ? (
						<span
							class="shimmer inline-block h-3 w-24 rounded align-middle"
							aria-hidden
						/>
					) : (
						subtitle || "\u00a0"
					)}
				</p>
				{pending ? (
					<span class="mt-px block truncate text-sm leading-tight">
						<span
							class="shimmer inline-block h-3 w-20 rounded align-middle"
							aria-hidden
						/>
					</span>
				) : link ? (
					<LinkLine link={link} />
				) : (
					<span class="mt-px block truncate text-sm leading-tight" aria-hidden>
						&nbsp;
					</span>
				)}
			</div>
			{right}
		</header>
	);
}
