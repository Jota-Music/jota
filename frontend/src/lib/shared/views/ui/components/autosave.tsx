import {
	ChooseSyncFolder,
	SessionPass,
	SetSessionPass,
	SetSyncAuto,
	SetSyncImport,
	SetSyncInterval,
	SyncAuto,
	SyncFolder,
	SyncImport,
	SyncInterval,
	SyncPending,
	SyncStatus,
} from "@bindings/app";
import { useSignal } from "@preact/signals";
import { useMutation, useQuery, useQueryClient } from "@tanstack/preact-query";
import {
	CloudDownload,
	CloudUpload,
	FolderSync,
	RotateCcw,
	Save,
	Smartphone,
	TriangleAlert,
} from "lucide-preact";
import { useEffect } from "preact/hooks";
import {
	autosave,
	chooseFolder,
	createDocument,
	isAndroid,
	openDocument,
	reload,
	syncName,
} from "@/lib/music/app/backup";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import { addError, notify } from "@/lib/shared/views/stores/errors";
import { PasswordInput } from "@/lib/shared/views/ui/components/password-input";
import { Sheet } from "@/lib/shared/views/ui/components/sheet";
import { Switch } from "@/lib/shared/views/ui/components/switch";

// Numeric and compact on purpose: "this minute" reads worse than "0 min" and
// the same three units work in every locale.
function ago(at: number): string {
	const minutes = Math.max(0, Math.round((Date.now() - at) / 60_000));
	if (minutes < 60) return `${minutes} min`;
	const hours = Math.round(minutes / 60);
	if (hours < 24) return `${hours} h`;
	return `${Math.round(hours / 24)} d`;
}

function size(bytes: number): string {
	if (bytes < 1024) return `${bytes} B`;
	if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
	return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

// A content:// URI is unreadable in a list, so only the document name shows and
// the full value stays in the title.
function name(uri: string): string {
	const last = uri.split("/").pop() ?? uri;
	try {
		return decodeURIComponent(last) || uri;
	} catch {
		return last;
	}
}

// The authority is the only part of a content:// URI that says where the
// document actually lives, and the phone versus the cloud is what the user has
// to see at a glance.
function place(uri: string): string {
	switch (uri.split("/")[2]) {
		case "com.google.android.apps.docs.storage":
			return t("settings.autosave.place.drive");
		case "com.android.externalstorage.documents":
			return t("settings.autosave.place.phone");
		default:
			return "";
	}
}

export function Autosave() {
	const queryClient = useQueryClient();

	const failed = useSignal(false);
	const cookiePass = useSignal("");
	const sheet = useSignal(false);

	useEffect(() => {
		void SessionPass().then((p) => {
			cookiePass.value = p ?? "";
		});
	}, []);

	const { data: folder = "", isLoading: folderLoading } = useQuery({
		queryKey: ["sync-folder"],
		queryFn: SyncFolder,
	});

	const folderName = useSignal("");

	useEffect(() => {
		if (!folder) {
			folderName.value = "";
			return;
		}
		void syncName(folder).then((name) => {
			folderName.value = name;
		});
	}, [folder]);

	const { data: auto = false, isLoading: autoLoading } = useQuery({
		queryKey: ["sync-auto"],
		queryFn: SyncAuto,
	});

	const autoMutation = useMutation({
		mutationFn: SetSyncAuto,
		onSuccess: (_value, variables) =>
			queryClient.setQueryData(["sync-auto"], variables),
		onError: (error) => addError(error, "sync"),
	});

	const { data: importing = true, isLoading: importLoading } = useQuery({
		queryKey: ["sync-import"],
		queryFn: SyncImport,
	});

	const importMutation = useMutation({
		mutationFn: SetSyncImport,
		onSuccess: (_value, variables) =>
			queryClient.setQueryData(["sync-import"], variables),
		onError: (error) => addError(error, "sync"),
	});

	// The pass is only stored on blur: every keystroke would re-encrypt the
	// sessions and start a sync pass.
	const passMutation = useMutation({
		mutationFn: SetSessionPass,
		onError: (error) => addError(error, "sync"),
	});

	const { data: interval = 15, isLoading: intervalLoading } = useQuery({
		queryKey: ["sync-interval"],
		queryFn: SyncInterval,
	});

	const { data: status } = useQuery({
		queryKey: ["sync-status"],
		queryFn: SyncStatus,
	});

	// The panel must not offer a sync the last pass already answered, so this one
	// is never served from the query cache.
	const { data: pending = false } = useQuery({
		queryKey: ["sync-pending"],
		queryFn: SyncPending,
		staleTime: 0,
	});

	const intervalMutation = useMutation({
		mutationFn: SetSyncInterval,
		onSuccess: (_value, variables) =>
			queryClient.setQueryData(["sync-interval"], variables),
		onError: (error) => addError(error, "sync"),
	});

	// A save is explicit: the user asked for it, so it is not chained to the
	// pending state that only decides how the button is painted. force is the
	// emergency exit offered only after a save failed: over a copy the provider
	// serves cached it would drop what another device has not saved yet.
	const saveMutation = useMutation({
		mutationFn: (force: boolean) => autosave("manual", force),
		onSuccess: () => {
			failed.value = false;
			notify(t("settings.autosave.done"));
		},
		onError: (error) => {
			failed.value = true;
			addError(error, "sync");
		},
	});

	// A reload is explicit too, and it is the one action that lets the file undo
	// a local deletion, so its notice says what the document brought in.
	const reloadMutation = useMutation({
		mutationFn: reload,
		onSuccess: (stats) => {
			failed.value = false;
			notify(
				t("settings.playlists.imported", {
					playlists: stats.playlists,
					follows: stats.follows,
					saved: stats.saved,
					order: stats.order,
					settings: stats.settings,
				}),
			);
		},
		onError: (error) => addError(error, "sync"),
	});

	// Picking a location is the moment the first pass is expected: syncing here
	// spares a second click and fills the panel with the real numbers. A cancel
	// answers nothing and leaves everything as it was.
	function adopt(path: string) {
		if (!path) return;
		queryClient.setQueryData(["sync-folder"], path);
		saveMutation.mutate(false);
	}

	const chooseMutation = useMutation({
		mutationFn: () => (isAndroid ? chooseFolder() : ChooseSyncFolder()),
		onSuccess: adopt,
		onError: (error) => addError(error, "sync"),
	});

	const openMutation = useMutation({
		mutationFn: openDocument,
		onSuccess: adopt,
		onError: (error) => addError(error, "sync"),
	});

	const createMutation = useMutation({
		mutationFn: createDocument,
		onSuccess: adopt,
		onError: (error) => addError(error, "sync"),
	});

	// Android has three sources, so both entry points open the same sheet: the
	// folder picker alone is not the only place a location can live.
	function pick() {
		if (isAndroid) sheet.value = true;
		else chooseMutation.mutate();
	}

	const busy =
		chooseMutation.isPending ||
		openMutation.isPending ||
		createMutation.isPending ||
		saveMutation.isPending ||
		reloadMutation.isPending;

	// Three ways in, named by what they leave behind instead of by the mechanism:
	// which system picker runs underneath is the phone's business.
	const sources = [
		{
			icon: Smartphone,
			title: t("settings.autosave.source.folder"),
			hint: t("settings.autosave.source.folderHint"),
			run: () => chooseMutation.mutate(),
		},
		{
			icon: CloudDownload,
			title: t("settings.autosave.source.open"),
			hint: t("settings.autosave.source.openHint"),
			run: () => openMutation.mutate(),
		},
		{
			icon: CloudUpload,
			title: t("settings.autosave.source.create"),
			hint: t("settings.autosave.source.createHint"),
			run: () => createMutation.mutate(),
		},
	];

	return (
		<div class="space-y-4">
			<div class="flex items-center gap-2">
				<FolderSync size={16} class="text-zinc-400" />
				<h3 class="text-xs font-semibold text-zinc-200">
					{t("settings.autosave.title")}
				</h3>
			</div>
			<p class="text-xs text-zinc-500">{t("settings.autosave.description")}</p>
			<p class="flex items-start gap-1.5 text-[11px] text-zinc-500">
				<TriangleAlert size={14} class="mt-px shrink-0" />
				{t("settings.autosave.secret")}
			</p>

			<div class="space-y-2 rounded-xl border border-zinc-800 bg-zinc-950/60 p-3">
				<p class="text-xs font-semibold text-zinc-400">
					{t("settings.autosave.session.title")}
				</p>
				<PasswordInput
					value={cookiePass.value}
					onValue={(v) => {
						cookiePass.value = v;
					}}
					onBlur={() => passMutation.mutate(cookiePass.value)}
					placeholder={t("settings.autosave.session.pass")}
					class="w-full rounded-lg border border-zinc-800 bg-zinc-950 p-2 text-xs text-zinc-200 outline-none transition-all focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600"
				/>
				<p class="text-xs text-zinc-500">
					{t("settings.autosave.session.hint")}
				</p>
			</div>

			<button
				type="button"
				disabled={busy}
				onClick={pick}
				class="flex w-full cursor-pointer items-center justify-center gap-1.5 rounded-lg border border-zinc-700 bg-zinc-950 px-3 py-2 text-xs font-semibold text-zinc-100 transition-colors hover:border-zinc-500 disabled:cursor-not-allowed disabled:opacity-50"
			>
				<FolderSync size={12} />
				{folder ? t("settings.autosave.change") : t("settings.autosave.choose")}
			</button>
			<div class="space-y-2 rounded-xl bg-zinc-950 p-3 ring-1 ring-zinc-800">
				<p class="text-[11px] text-zinc-500">
					{t("settings.autosave.location")}
				</p>
				<p
					class="truncate text-xs font-medium text-zinc-300"
					title={folder || undefined}
				>
					{folderLoading ? (
						<span class="shimmer block h-3.5 w-40 rounded" />
					) : folder ? (
						<>
							{folderName.value || name(folder)}
							{place(folder) && (
								<span class="text-zinc-500"> · {place(folder)}</span>
							)}
						</>
					) : (
						t("settings.autosave.none")
					)}
				</p>
				{status && status.at > 0 && (
					<dl class="mt-2 grid grid-cols-[1fr_auto] items-baseline gap-x-4 gap-y-1 text-[11px]">
						<dt class="text-zinc-500">{t("settings.autosave.status.at")}</dt>
						<dd class="text-right tabular-nums text-zinc-300">
							{ago(status.at)}
						</dd>
						<dt class="text-zinc-500">{t("settings.autosave.status.size")}</dt>
						<dd class="text-right tabular-nums text-zinc-300">
							{size(status.bytes)}
						</dd>
						<dt class="text-zinc-500">
							{t("settings.autosave.status.playlists")}
						</dt>
						<dd class="text-right tabular-nums text-zinc-300">
							{status.remote}
						</dd>
						{status.imported > 0 && (
							<>
								<dt class="text-zinc-500">
									{t("settings.autosave.status.imported")}
								</dt>
								<dd class="text-right tabular-nums text-emerald-300">
									+{status.imported}
								</dd>
							</>
						)}
					</dl>
				)}
				{status?.sealed && (
					<p class="mt-2 text-[11px] text-amber-300">
						{t("settings.autosave.session.sealed")}
					</p>
				)}
				{status?.problem && (
					<div class="mt-2 space-y-0.5">
						<p class="text-[11px] text-amber-300">
							{t("settings.autosave.problem")}
						</p>
						<p class="break-words text-[11px] text-amber-300/70">
							{status.problem}
						</p>
					</div>
				)}
			</div>

			<button
				type="button"
				disabled={busy || !folder}
				onClick={() => saveMutation.mutate(false)}
				class={cn(
					"flex w-full cursor-pointer items-center justify-center gap-1.5 rounded-lg px-3 py-2 text-xs font-semibold transition-colors disabled:cursor-not-allowed disabled:opacity-50",
					pending
						? "bg-(--dominant-color) text-(--binary-color) hover:opacity-85"
						: "border border-zinc-700 bg-zinc-950 text-zinc-100 hover:border-zinc-500",
				)}
			>
				<Save size={12} />
				{t("settings.autosave.save")}
			</button>
			<button
				type="button"
				disabled={busy || !folder}
				onClick={() => reloadMutation.mutate()}
				class="flex w-full cursor-pointer items-center justify-center gap-1.5 rounded-lg border border-zinc-700 bg-zinc-950 px-3 py-2 text-xs font-semibold text-zinc-100 transition-colors hover:border-zinc-500 disabled:cursor-not-allowed disabled:opacity-50"
			>
				<RotateCcw size={12} />
				{t("settings.autosave.reload")}
			</button>
			{status?.stale && (
				<div class="flex items-center justify-between gap-3 rounded-xl border border-amber-500/40 bg-amber-500/10 p-3">
					<p class="text-xs text-amber-300">{t("settings.autosave.stale")}</p>
					<button
						type="button"
						disabled={busy}
						onClick={pick}
						class="shrink-0 cursor-pointer rounded-lg border border-amber-500/40 px-2 py-1 text-xs font-semibold text-amber-200 transition-colors hover:border-amber-400 disabled:cursor-not-allowed disabled:opacity-50"
					>
						{t("settings.autosave.choose")}
					</button>
				</div>
			)}

			<div class="flex items-center justify-between gap-3 rounded-xl bg-zinc-950 p-3 ring-1 ring-zinc-800">
				<p class="text-xs text-zinc-500">{t("settings.autosave.auto")}</p>
				{autoLoading ? (
					<div class="shimmer h-6 w-11 shrink-0 rounded-full" />
				) : (
					<Switch
						checked={auto}
						label={t("settings.autosave.auto")}
						disabled={autoLoading || autoMutation.isPending}
						onToggle={() => autoMutation.mutate(!auto)}
					/>
				)}
			</div>

			<div class="grid grid-cols-[1fr_max-content] items-start gap-x-3 gap-y-1 rounded-xl bg-zinc-950 p-3 ring-1 ring-zinc-800">
				<p class="text-balance text-xs text-zinc-500">
					{t("settings.autosave.import")}
				</p>
				{importLoading ? (
					<div class="shimmer h-6 w-11 shrink-0 rounded-full" />
				) : (
					<Switch
						checked={importing}
						label={t("settings.autosave.import")}
						disabled={importLoading || importMutation.isPending}
						onToggle={() => importMutation.mutate(!importing)}
					/>
				)}
				<p class="col-start-1 text-balance text-[11px] text-zinc-500">
					{t("settings.autosave.importHint")}
				</p>
			</div>

			<div class="flex items-center justify-between gap-3 rounded-xl bg-zinc-950 p-3 ring-1 ring-zinc-800">
				<p class="text-xs text-zinc-500">{t("settings.autosave.interval")}</p>
				{intervalLoading ? (
					<div class="shimmer h-6 w-16 rounded-lg" />
				) : (
					<select
						value={interval}
						disabled={intervalLoading || intervalMutation.isPending}
						onChange={(e) =>
							intervalMutation.mutate(
								Number((e.target as HTMLSelectElement).value),
							)
						}
						class="cursor-pointer appearance-none rounded-lg border border-zinc-800 bg-zinc-950 px-2 py-1 text-xs text-zinc-100 outline-none focus:border-zinc-600"
					>
						{[5, 10, 15, 30, 60].map((m) => (
							<option key={m} value={m} class="bg-zinc-950">
								{m} min
							</option>
						))}
					</select>
				)}
			</div>

			{failed.value && folder && (
				<button
					type="button"
					disabled={busy}
					onClick={() => saveMutation.mutate(true)}
					class="cursor-pointer self-start text-xs font-semibold text-amber-300 underline transition-colors hover:text-amber-200 disabled:cursor-not-allowed disabled:opacity-50"
				>
					{t("settings.autosave.overwrite")}
				</button>
			)}

			<Sheet
				open={sheet.value}
				close={() => {
					sheet.value = false;
				}}
				mobileOnly
				centered
				labelledBy="autosave-source"
			>
				<p
					id="autosave-source"
					class="px-4 pb-2 text-sm font-semibold text-zinc-200"
				>
					{t("settings.autosave.source.title")}
				</p>
				{sources.map((source) => (
					<button
						key={source.title}
						type="button"
						onClick={() => {
							sheet.value = false;
							source.run();
						}}
						class="flex w-full cursor-pointer items-start gap-3 px-4 py-3 text-left transition-colors hover:bg-zinc-800/80"
					>
						<source.icon size={16} class="mt-0.5 shrink-0 text-zinc-500" />
						<span class="min-w-0">
							<span class="block text-sm text-zinc-200">{source.title}</span>
							<span class="block text-xs text-zinc-500">{source.hint}</span>
						</span>
					</button>
				))}
			</Sheet>
		</div>
	);
}
