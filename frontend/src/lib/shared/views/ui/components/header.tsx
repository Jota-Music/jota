import { useSignal, useSignalEffect } from "@preact/signals";
import { useQueryClient } from "@tanstack/preact-query";
import { System } from "@wailsio/runtime";
import {
	ArrowLeft,
	ChevronDown,
	House,
	Search,
	Settings,
	Turntable,
	Undo2,
	X,
} from "lucide-preact";
import { useCallback } from "preact/hooks";
import { Link, useLocation } from "wouter-preact";
import { spotifyConnected } from "@/lib/auth/views/stores/session";
import { parseSpotifyLink } from "@/lib/music/app/spotify-link";
import { removal, undoRemoval } from "@/lib/music/views/stores/removal";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import { WindowControlsBar } from "@/lib/shared/views/ui/components/window-controls-bar";
import { SpotifyIcon } from "@/lib/shared/views/ui/icons/spotify";
import YoutubeIcon from "@/lib/shared/views/ui/icons/youtube";
import { role, showSync, status } from "@/lib/sync/views/stores";

type Source = "spotify" | "youtube";
type SpotifyType = "user" | "track" | "album" | "playlist" | "artist";

const spotifyTypeOptions = [
	{ value: "user", label: "search.type.user" },
	{ value: "track", label: "search.type.track" },
	{ value: "album", label: "search.type.album" },
	{ value: "playlist", label: "search.type.playlist" },
	{ value: "artist", label: "search.type.artist" },
] as const;

const sourceKey = "search_source";
const typeKey = "search_type";

function loadSource(): Source {
	return localStorage.getItem(sourceKey) === "youtube" ? "youtube" : "spotify";
}

function loadType(): SpotifyType {
	const raw = localStorage.getItem(typeKey);
	return spotifyTypeOptions.some((option) => option.value === raw)
		? (raw as SpotifyType)
		: "user";
}

export function Header() {
	const [location, setLocation] = useLocation();
	const queryClient = useQueryClient();
	const desktop = System.IsDesktop();
	const openSearch = useSignal(false);
	const searchDraft = useSignal("");
	const source = useSignal<Source>(loadSource());
	const searchType = useSignal<SpotifyType>(loadType());

	const liveSource: Source = spotifyConnected.value ? source.value : "youtube";
	const inRoom = role.value !== "off";
	const connected = status.value === "open";

	useSignalEffect(() => {
		localStorage.setItem(sourceKey, source.value);
	});

	useSignalEffect(() => {
		localStorage.setItem(typeKey, searchType.value);
	});

	const onSearchSubmit = useCallback(
		(e: Event) => {
			e.preventDefault();
			const query = searchDraft.value.trim();
			if (!query) return;

			// A pasted Spotify link/URI decides its own type, so the selected
			// search type only applies to plain text queries.
			const live = spotifyConnected.value ? source.value : "youtube";
			const ref = live === "spotify" ? parseSpotifyLink(query) : null;
			const encoded = encodeURIComponent(ref ? ref.id : query);

			setLocation(
				live === "youtube"
					? `/search/youtube/${encoded}`
					: ref
						? `/search/${ref.type}/${encoded}`
						: `/search/${searchType.value}/${encoded}`,
			);
		},
		[setLocation],
	);

	const goBack = useCallback(() => {
		if (window.history.length > 1) {
			window.history.back();
		} else {
			setLocation("/");
		}
	}, [setLocation]);

	const placeholder =
		liveSource === "youtube"
			? t("search.youtubePlaceholder")
			: searchType.value === "user"
				? t("search.userPlaceholder")
				: t("search.spotifyPlaceholder", {
						type: t(`search.type.${searchType.value}`),
					});

	return (
		<header
			style="--wails-draggable: drag"
			class="sticky top-0 z-40 md:z-100 border-b border-zinc-800 bg-zinc-950"
		>
			<div class="mx-auto flex items-center justify-between text-sm text-zinc-300 h-10 pr-4">
				<div class="flex h-full items-center gap-1 pl-1">
					<button
						type="button"
						onClick={goBack}
						disabled={location === "/"}
						title={t("nav.back")}
						class={cn(
							"flex aspect-square h-full items-center justify-center cursor-pointer transition-colors",
							location === "/"
								? "text-zinc-400 disabled:opacity-30"
								: "text-zinc-400 hover:text-zinc-100",
							location !== "/" && "hover:text-zinc-100",
						)}
					>
						<ArrowLeft class="size-5 md:size-4" />
					</button>

					<Link
						href="/"
						class={cn(
							"flex aspect-square h-full items-center justify-center transition-colors",
							location === "/"
								? "text-(--dominant-color)"
								: "text-zinc-400 hover:text-zinc-100",
						)}
					>
						<House class="size-5 md:size-4" />
					</Link>

					<Link
						href="/settings"
						title={t("nav.settings")}
						class={cn(
							"flex aspect-square h-full items-center justify-center transition-colors",
							location.startsWith("/settings")
								? "text-(--dominant-color)"
								: "text-zinc-400 hover:text-zinc-100",
						)}
					>
						<Settings class="size-5 md:size-4" />
					</Link>

					{removal.value && (
						<button
							type="button"
							title={t("music.custom.undo")}
							aria-label={t("music.custom.undo")}
							onClick={() => void undoRemoval(queryClient)}
							class="flex aspect-square h-full items-center justify-center cursor-pointer text-zinc-400 transition-colors hover:text-zinc-100"
						>
							<Undo2 class="size-5 md:size-4" />
						</button>
					)}
				</div>

				<div class="flex items-center gap-3">
					<button
						type="button"
						onClick={() => {
							openSearch.value = !openSearch.value;
						}}
						class="flex size-8 items-center justify-center text-zinc-400 hover:text-zinc-100 cursor-pointer"
					>
						<Search class="size-5 md:size-4" strokeWidth={2.5} />
					</button>

					<button
						onClick={() => (showSync.value = !showSync.value)}
						type="button"
						title={t("nav.rooms")}
						class="flex size-8 items-center justify-center cursor-pointer"
					>
						<Turntable
							class={cn(
								"size-5 md:size-4",
								connected
									? "text-green-400"
									: inRoom
										? "text-yellow-400"
										: "text-zinc-400 hover:text-zinc-100",
							)}
						/>
					</button>
				</div>

				<div class="flex items-center gap-3 pr-1 h-full">
					{desktop && <div class="h-full w-26" aria-hidden />}
					<WindowControlsBar />
				</div>
			</div>

			<div
				class={cn(
					"mx-auto grid max-w-2xl overflow-hidden px-4 transition-[grid-template-rows,opacity] duration-200 md:px-0",
					openSearch.value
						? "grid-rows-[1fr] opacity-100"
						: "grid-rows-[0fr] opacity-0",
				)}
			>
				<form onSubmit={onSearchSubmit} class="min-h-0 overflow-hidden">
					<div class="py-2">
						{/* Mobile: source/type on top row, input + submit below */}
						<div class="flex flex-col gap-1.5 md:hidden">
							<div class="flex h-11 w-max items-stretch overflow-hidden rounded-lg border border-zinc-800 bg-zinc-950">
								<SourceToggle
									source={liveSource}
									showSpotify={spotifyConnected.value}
									onSelect={(v) => {
										source.value = v;
									}}
								/>
								{liveSource === "spotify" && (
									<TypeSelect
										value={searchType.value}
										onChange={(v) => {
											searchType.value = v;
										}}
										stretch
									/>
								)}
							</div>
							<div class="grid grid-cols-[1fr_max-content] gap-1.5">
								<div class="flex h-11 items-stretch overflow-hidden rounded-lg border border-zinc-800 bg-zinc-950 transition-all focus-within:border-zinc-600 focus-within:ring-1 focus-within:ring-zinc-600">
									<SearchInput
										value={searchDraft.value}
										placeholder={placeholder}
										onInput={(v) => {
											searchDraft.value = v;
										}}
										clearable
									/>
								</div>
								<button
									type="submit"
									title={t("search.title")}
									class="flex h-11 w-11 shrink-0 cursor-pointer items-center justify-center rounded-lg text-(--binary-color) bg-(--dominant-color) transition-opacity hover:opacity-75"
								>
									<Search class="size-5" strokeWidth={2.5} />
								</button>
							</div>
						</div>

						{/* Desktop: single pill + submit (unchanged) */}
						<div class="hidden gap-2 md:grid md:grid-cols-[1fr_max-content]">
							<div class="flex h-10 items-stretch overflow-hidden rounded-lg border border-zinc-800 bg-zinc-950 transition-all focus-within:border-zinc-600 focus-within:ring-1 focus-within:ring-zinc-600">
								<SourceToggle
									source={liveSource}
									showSpotify={spotifyConnected.value}
									onSelect={(v) => {
										source.value = v;
									}}
								/>
								{liveSource === "spotify" && (
									<TypeSelect
										value={searchType.value}
										onChange={(v) => {
											searchType.value = v;
										}}
									/>
								)}
								<SearchInput
									value={searchDraft.value}
									placeholder={placeholder}
									onInput={(v) => {
										searchDraft.value = v;
									}}
									clearable
								/>
							</div>
							<button
								class="flex h-10 w-10 shrink-0 cursor-pointer items-center justify-center rounded-lg px-3 py-2 text-(--binary-color) bg-(--dominant-color) transition-opacity hover:opacity-75"
								type="submit"
								title={t("search.title")}
							>
								<Search class="size-4" strokeWidth={2.5} />
							</button>
						</div>
					</div>
				</form>
			</div>
		</header>
	);
}

function SourceToggle({
	source,
	showSpotify,
	onSelect,
}: {
	source: Source;
	showSpotify: boolean;
	onSelect: (source: Source) => void;
}) {
	return (
		<div class="flex shrink-0 items-stretch border-r border-zinc-800">
			{showSpotify && (
				<button
					type="button"
					title={t("search.inSpotify")}
					onClick={() => onSelect("spotify")}
					class={cn(
						"flex w-11 items-center justify-center transition-colors cursor-pointer md:w-10",
						source === "spotify"
							? "bg-zinc-800 text-[#1DB954]"
							: "text-zinc-500 hover:text-zinc-300",
					)}
				>
					<SpotifyIcon size={16} />
				</button>
			)}
			<button
				type="button"
				title={t("search.inYouTube")}
				onClick={() => onSelect("youtube")}
				class={cn(
					"flex w-11 items-center justify-center transition-colors cursor-pointer md:w-10",
					source === "youtube"
						? "bg-zinc-800 text-[#FF0033]"
						: "text-zinc-500 hover:text-zinc-300",
				)}
			>
				<YoutubeIcon class="size-4" />
			</button>
		</div>
	);
}

function TypeSelect({
	value,
	onChange,
	stretch,
}: {
	value: SpotifyType;
	onChange: (value: SpotifyType) => void;
	stretch?: boolean;
}) {
	return (
		<div
			class={cn(
				"relative shrink-0 border-r border-zinc-800",
				stretch && "flex-1",
			)}
		>
			<select
				value={value}
				onChange={(e) =>
					onChange((e.target as HTMLSelectElement).value as SpotifyType)
				}
				class={cn(
					"h-full w-full cursor-pointer appearance-none rounded-none bg-transparent pl-2 pr-6 text-sm text-white outline-none md:pl-3 md:pr-7",
					stretch && "w-full",
				)}
			>
				{spotifyTypeOptions.map((opt) => (
					<option key={opt.value} value={opt.value} class="bg-zinc-950">
						{t(opt.label)}
					</option>
				))}
			</select>
			<ChevronDown class="pointer-events-none absolute right-1.5 top-1/2 size-3 -translate-y-1/2 text-zinc-400 md:right-2" />
		</div>
	);
}

function SearchInput({
	value,
	placeholder,
	onInput,
	clearable,
}: {
	value: string;
	placeholder: string;
	onInput: (value: string) => void;
	clearable?: boolean;
}) {
	return (
		<div class="relative flex min-w-0 flex-1 items-center">
			<input
				value={value}
				placeholder={placeholder}
				onInput={(e) => onInput((e.target as HTMLInputElement).value)}
				class={cn(
					"h-full w-full min-w-0 bg-transparent pl-3 text-sm text-white outline-none placeholder:text-zinc-600 md:pl-4",
					clearable ? "pr-9" : "pr-3",
				)}
			/>
			{clearable && value && (
				<button
					type="button"
					title={t("search.clear")}
					onClick={() => onInput("")}
					class="absolute right-2 flex size-6 cursor-pointer items-center justify-center rounded text-zinc-500 transition-colors hover:bg-zinc-800 hover:text-white"
				>
					<X size={14} />
				</button>
			)}
		</div>
	);
}
