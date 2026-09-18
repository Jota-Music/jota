import { Disc3, Pause, Play } from "lucide-preact";
import type { ComponentChildren } from "preact";
import { Link } from "wouter-preact";
import { t } from "@/lib/shared/i18n";
import { Mosaic } from "@/lib/shared/views/ui/components/mosaic";

function LinkLine({ link }: { link: { to: string; label: string } }) {
	return (
		<Link
			to={link.to}
			class="mt-1 block truncate text-sm opacity-70 hover:opacity-100 hover:underline"
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
	onPlay,
	playing,
	actions,
}: {
	cover?: string;
	covers?: string[];
	title: string;
	subtitle?: string;
	link?: { to: string; label: string };
	onPlay?: () => void;
	playing?: boolean;
	actions?: ComponentChildren;
}) {
	const right =
		onPlay || actions ? (
			<div class="ml-auto flex shrink-0 items-center gap-2">
				{onPlay && (
					<button
						type="button"
						title={t("music.playAll")}
						aria-label={t("music.playAll")}
						onClick={onPlay}
						class="flex h-10 w-10 shrink-0 cursor-pointer items-center justify-center rounded-full bg-(--dominant-color) text-(--binary-color) transition"
					>
						{playing ? (
							<Pause class="size-5 fill-current" />
						) : (
							<Play class="size-5 fill-current" />
						)}
					</button>
				)}
				{actions}
			</div>
		) : null;

	return (
		<header class="flex shrink-0 items-center gap-4">
			{covers?.length ? (
				<div class="h-16 w-16 shrink-0 overflow-hidden rounded-md">
					<Mosaic covers={covers} placeholder={<Disc3 size={16} />} />
				</div>
			) : cover ? (
				<img
					src={cover}
					alt=""
					class="h-16 w-16 shrink-0 rounded-md object-cover"
				/>
			) : (
				<div class="flex h-16 w-16 shrink-0 items-center justify-center rounded-md bg-zinc-900 text-zinc-500">
					<Disc3 size={28} />
				</div>
			)}
			<div class="min-w-0">
				<h2 class="truncate text-xl font-semibold leading-tight">{title}</h2>
				{subtitle && <p class="text-sm opacity-70 mt-1">{subtitle}</p>}
				{link && <LinkLine link={link} />}
			</div>
			{right}
		</header>
	);
}
