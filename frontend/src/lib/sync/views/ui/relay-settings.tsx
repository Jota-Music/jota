import { TriangleAlert, Turntable } from "lucide-preact";
import { useEffect, useState } from "preact/hooks";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import { PasswordInput } from "@/lib/shared/views/ui/components/password-input";
import * as transport from "@/lib/sync/app/transport";
import * as store from "@/lib/sync/views/stores";

const inputClass =
	"w-full rounded-lg border border-zinc-800 bg-zinc-950 p-3 text-sm text-zinc-100 outline-none focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600 transition-all";

export function RelaySettings() {
	const url = store.relayUrl.value;
	const [relay, setRelay] = useState<"idle" | "checking" | "ok" | "error">(
		"idle",
	);

	useEffect(() => {
		const trimmed = url.trim();
		if (trimmed === "") {
			setRelay("idle");
			store.tokenRequired.value = false;
			return;
		}
		let alive = true;
		setRelay("checking");
		const timer = setTimeout(() => {
			transport
				.check(trimmed)
				.then((required) => {
					if (!alive) return;
					setRelay("ok");
					store.tokenRequired.value = required;
				})
				.catch(() => alive && setRelay("error"));
		}, 500);
		return () => {
			alive = false;
			clearTimeout(timer);
		};
	}, [url]);

	return (
		<section class="space-y-4 rounded-2xl border border-zinc-800 bg-zinc-900/40 p-4">
			<div class="flex items-center gap-2">
				<Turntable size={18} class="text-zinc-400" />
				<h2 class="text-sm font-semibold text-zinc-200">
					{t("sync.relay.title")}
				</h2>
			</div>
			<p class="text-xs text-zinc-500">{t("sync.relay.body")}</p>

			<div class="space-y-1.5">
				<div class="flex items-center justify-between">
					<label for="sync-relay" class="text-xs font-medium text-zinc-400">
						{t("sync.relay.label")}
					</label>
					<RelayStatus state={relay} />
				</div>
				<input
					id="sync-relay"
					class={inputClass}
					placeholder={t("sync.relay.placeholder")}
					value={url}
					onInput={(e) => (store.relayUrl.value = e.currentTarget.value)}
				/>
			</div>

			<div
				class="grid transition-all duration-200 ease-out"
				style={{
					gridTemplateRows: store.tokenRequired.value ? "1fr" : "0fr",
				}}
			>
				<div class="min-h-0 overflow-hidden">
					<div class="space-y-1.5 pt-1.5">
						<label for="sync-token" class="text-xs font-medium text-zinc-400">
							{t("sync.relay.token")}
						</label>
						<PasswordInput
							id="sync-token"
							class={inputClass}
							label={t("sync.relay.token")}
							value={store.token.value}
							onValue={(v) => (store.token.value = v)}
						/>
						<p class="flex items-center gap-1.5 text-xs text-yellow-400">
							<TriangleAlert size={14} class="shrink-0" />
							{t("sync.relay.tokenRequired")}
						</p>
					</div>
				</div>
			</div>
		</section>
	);
}

function RelayStatus({
	state,
}: {
	state: "idle" | "checking" | "ok" | "error";
}) {
	const status = {
		idle: null,
		checking: {
			label: t("sync.relay.checking"),
			text: "text-zinc-400",
			dot: "bg-zinc-500 animate-pulse",
		},
		ok: {
			label: t("sync.relay.reachable"),
			text: "text-green-400",
			dot: "bg-green-400",
		},
		error: {
			label: t("sync.relay.unreachable"),
			text: "text-red-400",
			dot: "bg-red-400",
		},
	}[state];

	if (!status) {
		return <span class="h-4 w-24" aria-hidden />;
	}

	return (
		<span class={cn("flex items-center gap-1.5 text-xs", status.text)}>
			<span class={cn("size-1.5 rounded-full", status.dot)} />
			{status.label}
		</span>
	);
}
