import { Check, CircleAlert, Loader, TriangleAlert } from "lucide-preact";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import type { RelayCheck } from "@/lib/sync/app/transport";

export type State = "idle" | "checking" | "ok" | "error";

// The pill is one binary: can we talk to the relay. Whether it is silent
// because there is no relay yet or because it did not answer is the reason
// line's job, so the label never has to hedge.
export function Status({ state }: { state: State }) {
	const status = {
		idle: {
			label: t("sync.relay.notConnected"),
			text: "text-zinc-500",
			dot: "bg-zinc-700",
		},
		checking: {
			label: t("sync.relay.checking"),
			text: "text-zinc-400",
			icon: <Loader size={12} class="shrink-0 animate-spin" />,
		},
		ok: {
			label: t("sync.relay.connected"),
			text: "text-green-400",
			dot: "bg-green-400",
		},
		error: {
			label: t("sync.relay.notConnected"),
			text: "text-red-400",
			dot: "bg-red-400",
		},
	}[state];

	return (
		<span
			class={cn("ml-auto flex items-center gap-1.5 text-xs", status.text)}
			aria-live="polite"
		>
			{status.icon ?? (
				<span class={cn("size-1.5 shrink-0 rounded-full", status.dot)} />
			)}
			{status.label}
		</span>
	);
}

// The token is only ever judged by the relay itself, so the field reports what
// the last check concluded about it instead of assuming it is fine.
export function TokenStatus({ verdict }: { verdict: RelayCheck }) {
	const status = {
		none: null,
		required: {
			label: t("sync.relay.tokenRequired"),
			text: "text-yellow-400",
			icon: <TriangleAlert size={14} class="shrink-0" />,
		},
		accepted: {
			label: t("sync.relay.tokenAccepted"),
			text: "text-green-400",
			icon: <Check size={14} class="shrink-0" />,
		},
		rejected: {
			label: t("errors.relay.tokenRejected"),
			text: "text-red-400",
			icon: <CircleAlert size={14} class="shrink-0" />,
		},
	}[verdict.token];

	if (!status) return null;
	return (
		<p class={cn("flex items-center gap-1.5 text-xs", status.text)}>
			{status.icon}
			{status.label}
		</p>
	);
}
