import { useSignal, useSignalEffect } from "@preact/signals";
import { CircleAlert, TriangleAlert, Turntable } from "lucide-preact";
import { t } from "@/lib/shared/i18n";
import { PasswordInput } from "@/lib/shared/views/ui/components/password-input";
import type { RelayCheck } from "@/lib/sync/app/transport";
import * as transport from "@/lib/sync/app/transport";
import * as store from "@/lib/sync/views/stores";
import { Session } from "@/lib/sync/views/ui/relay/session";
import {
	type State,
	Status,
	TokenStatus,
} from "@/lib/sync/views/ui/relay/status";

export function RelaySettings() {
	const url = store.relayUrl.value;
	const relay = useSignal<State>("idle");
	const verdict = useSignal<RelayCheck>({ error: "", token: "none" });

	useSignalEffect(() => {
		const target = store.relayUrl.value.trim();
		const secret = store.token.value.trim();
		if (target === "") {
			relay.value = "idle";
			verdict.value = { error: "", token: "none" };
			store.tokenRequired.value = false;
			return;
		}
		let alive = true;
		relay.value = "checking";
		const timer = setTimeout(async () => {
			const result = await transport.check(target, secret);
			if (!alive) return;
			verdict.value = result;
			relay.value = result.error === "" ? "ok" : "error";
			// Only a verdict about the token opens the field: a relay that is down
			// says nothing about it, and must not make the field disappear.
			if (result.token !== "none") store.tokenRequired.value = true;
		}, 500);
		return () => {
			alive = false;
			clearTimeout(timer);
		};
	});

	return (
		<section
			id="relay"
			class="space-y-4 rounded-2xl border border-zinc-800 bg-zinc-900/40 p-4"
		>
			<div class="flex items-center gap-2">
				<Turntable size={18} class="text-zinc-400" />
				<h2 class="text-sm font-semibold text-zinc-200">
					{t("sync.relay.title")}
				</h2>
				<Status state={relay.value} />
			</div>
			<p class="text-xs text-zinc-500">{t("sync.relay.body")}</p>

			<div class="space-y-1.5">
				<label for="sync-relay" class="text-xs font-medium text-zinc-400">
					{t("sync.relay.label")}
				</label>
				<input
					id="sync-relay"
					class="w-full rounded-lg border border-zinc-800 bg-zinc-950 p-3 text-sm text-zinc-100 outline-none focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600 transition-all disabled:opacity-50"
					placeholder={t("sync.relay.placeholder")}
					value={url}
					disabled={store.relayLocked.value}
					onInput={(e) => (store.relayUrl.value = e.currentTarget.value)}
					onBlur={transport.save}
				/>
			</div>

			{store.relayLocked.value && (
				<p class="flex items-center gap-1.5 text-xs text-zinc-500">
					<TriangleAlert size={14} class="shrink-0" />
					{t("sync.relay.pinned")}
				</p>
			)}

			{verdict.value.error !== "" && (
				<p class="flex items-start gap-1.5 text-xs text-red-400">
					<CircleAlert size={14} class="mt-px shrink-0" />
					{verdict.value.error}
				</p>
			)}

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
							class="w-full rounded-lg border border-zinc-800 bg-zinc-950 p-3 text-sm text-zinc-100 outline-none focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600 transition-all disabled:opacity-50"
							label={t("sync.relay.token")}
							value={store.token.value}
							disabled={store.relayLocked.value}
							onValue={(v) => (store.token.value = v)}
							onBlur={transport.save}
						/>
						<TokenStatus verdict={verdict.value} />
					</div>
				</div>
			</div>

			<Session />
		</section>
	);
}
