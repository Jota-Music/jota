import { System } from "@wailsio/runtime";
import {
	ArrowLeft,
	ChevronDown,
	House,
	Search,
	Settings,
	Users,
} from "lucide-preact";
import { useCallback, useEffect, useState } from "preact/hooks";
import { Link, useLocation } from "wouter-preact";
import { cn } from "@/lib/shared/utils/tw";
import { WindowControlsBar } from "@/lib/shared/views/ui/components/window-controls-bar";
import { SpotifyIcon } from "@/lib/shared/views/ui/icons/spotify";
import YoutubeIcon from "@/lib/shared/views/ui/icons/youtube";
import { showSync } from "@/lib/sync/views/stores";

type Source = "spotify" | "youtube";
type SpotifyType = "user" | "track" | "album" | "playlist" | "artist";

const spotifyTypeOptions = [
	{ value: "user", label: "User" },
	{ value: "track", label: "Track" },
	{ value: "album", label: "Album" },
	{ value: "playlist", label: "Playlist" },
	{ value: "artist", label: "Artist" },
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

export function Header({
	class: _class,
	className,
	onDragStart,
}: {
	class?: string;
	className?: string;
	onDragStart?: (e: MouseEvent) => void;
}) {
	const [location, setLocation] = useLocation();
	const desktop = System.IsDesktop();
	const [openSearch, setOpenSearch] = useState(false);
	const [searchDraft, setSearchDraft] = useState("");
	const [source, setSource] = useState<Source>(loadSource);
	const [searchType, setSearchType] = useState<SpotifyType>(loadType);

	useEffect(() => {
		localStorage.setItem(sourceKey, source);
	}, [source]);

	useEffect(() => {
		localStorage.setItem(typeKey, searchType);
	}, [searchType]);

	const onSearchSubmit = useCallback(
		(e: Event) => {
			e.preventDefault();
			const query = searchDraft.trim();
			if (!query) return;
			const encoded = encodeURIComponent(query);
			setLocation(
				source === "youtube"
					? `/search/youtube/${encoded}`
					: `/search/${searchType}/${encoded}`,
			);
		},
		[searchDraft, searchType, source, setLocation],
	);

	const goBack = useCallback(() => {
		if (window.history.length > 1) {
			window.history.back();
		} else {
			setLocation("/");
		}
	}, [setLocation]);

	function handleDragMouseDown(e: MouseEvent) {
		const target = e.target as HTMLElement;
		if (
			target.closest(
				"button, a, input, select, textarea, [role='slider'], [data-no-drag]",
			)
		)
			return;
		onDragStart?.(e);
	}

	const placeholder =
		source === "youtube"
			? "Search in YouTube (videos or playlists)"
			: searchType === "user"
				? "Spotify username..."
				: `Spotify ${searchType.charAt(0).toUpperCase() + searchType.slice(1)} ID or URI...`;

	return (
		// biome-ignore lint/a11y/noStaticElementInteractions: pointer-only window drag surface
		<header
			style="--wails-draggable: drag"
			onMouseDown={handleDragMouseDown}
			class={cn(
				"sticky top-0 z-40 md:z-100 border-b border-zinc-800 bg-zinc-950/80 backdrop-blur",
				_class,
				className,
			)}
		>
			<div class="mx-auto flex items-center justify-between text-sm text-zinc-300 h-10 px-4">
				<div class="flex items-center gap-3 pl-1">
					{location !== "/" && (
						<button
							type="button"
							onClick={goBack}
							class="flex size-8 items-center justify-center text-zinc-400 hover:text-zinc-100 cursor-pointer"
						>
							<ArrowLeft class="size-5 md:size-4" />
						</button>
					)}

					<Link
						href="/settings"
						title="Settings"
						class="flex size-8 items-center justify-center text-zinc-400 hover:text-zinc-100"
					>
						<Settings class="size-5 md:size-4" />
					</Link>

					<Link
						href="/"
						class="flex size-8 items-center justify-center text-zinc-400 hover:text-zinc-100"
					>
						<House class="size-5 md:size-4" />
					</Link>
				</div>

				<div class="flex items-center gap-3">
					<button
						type="button"
						onClick={() => setOpenSearch((v) => !v)}
						class="flex size-8 items-center justify-center text-zinc-400 hover:text-zinc-100 cursor-pointer"
					>
						<Search class="size-5 md:size-4" strokeWidth={2.5} />
					</button>

					<button
						onClick={() => (showSync.value = !showSync.value)}
						type="button"
						title="Listen together"
						class="flex size-8 items-center justify-center cursor-pointer"
					>
						<Users class="size-5 md:size-4 text-zinc-400 hover:text-zinc-100" />
					</button>
				</div>

				<div class="flex items-center gap-3 pr-1 h-full">
					{desktop && <div class="h-full w-26" aria-hidden />}
					<WindowControlsBar />
				</div>
			</div>

			<div
				class={cn(
					"mx-auto max-w-2xl overflow-hidden transition-all duration-200 px-4 md:px-0",
					openSearch ? "max-h-40 opacity-100 py-2" : "max-h-0 opacity-0 py-0",
				)}
			>
				<form onSubmit={onSearchSubmit}>
					{/* Mobile: input on its own row, controls + submit below */}
					<div class="flex flex-col gap-2 md:hidden">
						<div class="flex h-11 items-stretch overflow-hidden rounded-lg border border-zinc-800 bg-zinc-950 transition-all focus-within:border-zinc-600 focus-within:ring-1 focus-within:ring-zinc-600">
							<SearchInput
								value={searchDraft}
								placeholder={placeholder}
								onInput={setSearchDraft}
							/>
						</div>
						<div class="flex h-11 items-stretch overflow-hidden rounded-lg border border-zinc-800 bg-zinc-950">
							<SourceToggle source={source} onSelect={setSource} />
							{source === "spotify" && (
								<TypeSelect
									value={searchType}
									onChange={setSearchType}
									stretch
								/>
							)}
							<button
								type="submit"
								title="Search"
								class="ml-auto flex w-11 shrink-0 cursor-pointer items-center justify-center text-(--binary-color) bg-(--dominant-color) transition-opacity hover:opacity-75"
							>
								<Search class="size-5" strokeWidth={2.5} />
							</button>
						</div>
					</div>

					{/* Desktop: single pill + submit (unchanged) */}
					<div class="hidden gap-2 md:grid md:grid-cols-[1fr_max-content]">
						<div class="flex h-10 items-stretch overflow-hidden rounded-lg border border-zinc-800 bg-zinc-950 transition-all focus-within:border-zinc-600 focus-within:ring-1 focus-within:ring-zinc-600">
							<SourceToggle source={source} onSelect={setSource} />
							{source === "spotify" && (
								<TypeSelect value={searchType} onChange={setSearchType} />
							)}
							<SearchInput
								value={searchDraft}
								placeholder={placeholder}
								onInput={setSearchDraft}
							/>
						</div>
						<button
							class="flex h-10 w-10 shrink-0 cursor-pointer items-center justify-center rounded-lg px-3 py-2 text-(--binary-color) bg-(--dominant-color) transition-opacity hover:opacity-75"
							type="submit"
							title="Search"
						>
							<Search class="size-4" strokeWidth={2.5} />
						</button>
					</div>
				</form>
			</div>
		</header>
	);
}

function SourceToggle({
	source,
	onSelect,
}: {
	source: Source;
	onSelect: (source: Source) => void;
}) {
	return (
		<div class="flex shrink-0 items-stretch border-r border-zinc-800">
			<button
				type="button"
				title="Search in Spotify"
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
			<button
				type="button"
				title="Search in YouTube"
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
					"h-full cursor-pointer appearance-none bg-transparent pl-2 pr-6 text-sm text-white outline-none md:pl-3 md:pr-7",
					stretch && "w-full",
				)}
			>
				{spotifyTypeOptions.map((opt) => (
					<option key={opt.value} value={opt.value} class="bg-zinc-950">
						{opt.label}
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
}: {
	value: string;
	placeholder: string;
	onInput: (value: string) => void;
}) {
	return (
		<div class="relative flex min-w-0 flex-1 items-center">
			<input
				value={value}
				placeholder={placeholder}
				onInput={(e) => onInput((e.target as HTMLInputElement).value)}
				class="h-full w-full min-w-0 bg-transparent pl-3 pr-3 text-sm text-white outline-none placeholder:text-zinc-600 md:pl-4"
			/>
		</div>
	);
}
