import {
	Check,
	ClipboardPaste,
	Copy,
	Eye,
	EyeOff,
	Link,
	LogOut,
	Plus,
	QrCode,
	Radio,
	ScanLine,
	Unplug,
	Users,
	X,
} from "lucide-preact";
import { useState } from "preact/hooks";
import * as transport from "@/lib/p2p/app/transport";
import * as store from "@/lib/p2p/views/stores";
import { showP2P } from "@/lib/p2p/views/stores";
import { QrModal } from "@/lib/p2p/views/ui/qr-modal";
import { Scanner } from "@/lib/p2p/views/ui/scanner";
import { cn } from "@/lib/shared/utils/tw";
import { Sheet } from "@/lib/shared/views/ui/components/sheet";
import "@/lib/p2p/views/stores/sync";

export function P2PPanel() {
	const r = store.role.value;
	const status = store.status.value;
	const active = r !== "off";
	const connected = status === "open";

	return (
		<>
			<Sheet
				open={showP2P.value}
				close={() => (showP2P.value = false)}
				labelledBy="p2p-panel-title"
				closeLabel="Cerrar P2P"
			>
				<header class="flex shrink-0 flex-col gap-1 border-b border-zinc-800 px-4 py-3">
					<div class="flex items-center justify-between gap-2 text-white">
						<div class="flex items-center gap-2">
							{active ? (
								<Radio
									size={22}
									class={cn(connected ? "text-green-400" : "text-yellow-400")}
								/>
							) : (
								<Users size={22} class="text-zinc-400" />
							)}
							<h2 id="p2p-panel-title" class="sr-only">
								P2P Session
							</h2>
							<span class="text-xs text-zinc-500 tabular-nums">
								{active
									? connected
										? "Connected"
										: "Connecting..."
									: "Offline"}
							</span>
						</div>
						<button
							type="button"
							class="rounded-lg p-2 text-zinc-400 transition hover:bg-zinc-800 hover:text-white cursor-pointer"
							aria-label="Cerrar"
							onClick={() => (showP2P.value = false)}
						>
							<X size={20} />
						</button>
					</div>
				</header>

				<div class="flex-1 overflow-y-auto px-4 py-4">
					<SessionForm />
				</div>
			</Sheet>
			<QrModal />
		</>
	);
}

function SessionForm() {
	const r = store.role.value;
	const [offer, setOffer] = useState("");
	const [answer, setAnswer] = useState("");
	const [scan, setScan] = useState<null | "offer" | "answer">(null);
	const [showCode, setShowCode] = useState(false);
	const canScan = !!navigator.mediaDevices?.getUserMedia;

	if (scan && canScan) {
		return (
			<div class="flex flex-col gap-3">
				<p class="text-sm text-zinc-400">
					{scan === "offer"
						? "Point the camera at the host's QR code."
						: "Point the camera at the guest's QR code."}
				</p>
				<Scanner
					onResult={(text) => {
						const code = text.trim();
						setScan(null);
						if (!code) return;
						if (scan === "offer") {
							setOffer(code);
							void transport.joinGuest(code);
						} else {
							setAnswer(code);
							void transport.acceptAnswer(code);
						}
					}}
				/>
				<button
					type="button"
					class="flex items-center justify-center gap-2 rounded-lg bg-zinc-800 px-4 py-2.5 text-sm text-zinc-100 hover:bg-zinc-700 transition-colors cursor-pointer"
					onClick={() => setScan(null)}
				>
					<X size={16} />
					Cancel
				</button>
			</div>
		);
	}

	if (r === "off") {
		return (
			<div class="flex flex-col gap-3">
				<p class="text-sm text-zinc-400">Listen together with a friend.</p>
				<button
					type="button"
					class="flex items-center justify-center gap-2 rounded-lg bg-zinc-800 px-4 py-2.5 text-sm text-zinc-100 hover:bg-zinc-700 transition-colors cursor-pointer"
					onClick={() => void transport.startHost()}
				>
					<Plus size={16} />
					Host session
				</button>
				<div class="flex items-center gap-2 text-xs text-zinc-600">
					<div class="h-px flex-1 bg-zinc-800" />
					<span>or join</span>
					<div class="h-px flex-1 bg-zinc-800" />
				</div>
				<textarea
					class="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-3 text-xs text-zinc-100 outline-none focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600 transition-all resize-none"
					rows={3}
					placeholder="Paste invite code..."
					value={offer}
					onInput={(e) => setOffer(e.currentTarget.value)}
				/>
				<div class="flex gap-2">
					<button
						type="button"
						class="flex flex-1 items-center justify-center gap-2 rounded-lg border border-zinc-800 px-4 py-2.5 text-sm text-zinc-100 hover:bg-zinc-800 transition-colors cursor-pointer"
						onClick={async () => {
							const text = await transport.paste();
							if (text) setOffer(text);
						}}
					>
						<ClipboardPaste size={16} />
						Paste
					</button>
					{canScan && (
						<button
							type="button"
							class="flex flex-1 items-center justify-center gap-2 rounded-lg border border-zinc-800 px-4 py-2.5 text-sm text-zinc-100 hover:bg-zinc-800 transition-colors cursor-pointer"
							onClick={() => setScan("offer")}
						>
							<ScanLine size={16} />
							Scan host QR
						</button>
					)}
				</div>
				<button
					type="button"
					class="flex items-center justify-center gap-2 rounded-lg bg-zinc-800 px-4 py-2.5 text-sm text-zinc-100 hover:bg-zinc-700 transition-colors cursor-pointer disabled:opacity-40"
					disabled={!offer.trim()}
					onClick={() => {
						const code = offer.trim();
						if (code) void transport.joinGuest(code);
					}}
				>
					<Link size={16} />
					Join session
				</button>
				{store.error.value && (
					<p class="text-xs text-red-400">{store.error.value}</p>
				)}
			</div>
		);
	}

	if (r === "host") {
		return (
			<div class="flex flex-col gap-3">
				<p class="text-sm text-zinc-400">
					Share this invite code with a guest.
				</p>
				{showCode && (
					<textarea
						class="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-3 text-xs text-zinc-100 outline-none resize-none font-mono"
						rows={3}
						readOnly
						value={store.inviteCode.value}
					/>
				)}
				<div class="grid grid-cols-2 gap-2">
					<button
						type="button"
						class="flex items-center justify-center gap-2 rounded-lg bg-zinc-800 px-4 py-2.5 text-sm text-zinc-100 hover:bg-zinc-700 transition-colors cursor-pointer"
						onClick={() => void transport.copy(store.inviteCode.value)}
					>
						<Copy size={16} />
						Copy invite
					</button>
					<button
						type="button"
						class="flex items-center justify-center gap-2 rounded-lg border border-zinc-800 px-4 py-2.5 text-sm text-zinc-100 hover:bg-zinc-800 transition-colors cursor-pointer"
						onClick={() => setShowCode((v) => !v)}
					>
						{showCode ? <EyeOff size={16} /> : <Eye size={16} />}
						{showCode ? "Hide code" : "Show code"}
					</button>
				</div>
				{store.inviteCode.value && (
					<button
						type="button"
						class="flex items-center justify-center gap-2 rounded-lg border border-zinc-800 px-4 py-2.5 text-sm text-zinc-100 hover:bg-zinc-800 transition-colors cursor-pointer"
						onClick={() => (store.qrValue.value = store.inviteCode.value)}
					>
						<QrCode size={16} />
						Show QR
					</button>
				)}
				<div class="flex items-center gap-2 text-xs text-zinc-600">
					<div class="h-px flex-1 bg-zinc-800" />
					<span>then paste answer</span>
					<div class="h-px flex-1 bg-zinc-800" />
				</div>
				<textarea
					class="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-3 text-xs text-zinc-100 outline-none focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600 transition-all resize-none font-mono"
					rows={3}
					placeholder="Paste answer code..."
					value={answer}
					onInput={(e) => setAnswer(e.currentTarget.value)}
				/>
				<div class="flex gap-2">
					<button
						type="button"
						class="flex flex-1 items-center justify-center gap-2 rounded-lg border border-zinc-800 px-4 py-2.5 text-sm text-zinc-100 hover:bg-zinc-800 transition-colors cursor-pointer"
						onClick={async () => {
							const text = await transport.paste();
							if (text) setAnswer(text);
						}}
					>
						<ClipboardPaste size={16} />
						Paste
					</button>
					{canScan && (
						<button
							type="button"
							class="flex flex-1 items-center justify-center gap-2 rounded-lg border border-zinc-800 px-4 py-2.5 text-sm text-zinc-100 hover:bg-zinc-800 transition-colors cursor-pointer"
							onClick={() => setScan("answer")}
						>
							<ScanLine size={16} />
							Scan guest QR
						</button>
					)}
				</div>
				<button
					type="button"
					class="flex items-center justify-center gap-2 rounded-lg bg-zinc-800 px-4 py-2.5 text-sm text-zinc-100 hover:bg-zinc-700 transition-colors cursor-pointer disabled:opacity-40"
					disabled={!answer.trim()}
					onClick={() => {
						const code = answer.trim();
						if (code) void transport.acceptAnswer(code);
					}}
				>
					<Check size={16} />
					Accept answer
				</button>
				<button
					type="button"
					class="flex items-center justify-center gap-2 rounded-lg bg-red-900/40 px-4 py-2.5 text-sm text-red-200 hover:bg-red-900/60 transition-colors cursor-pointer mt-2"
					onClick={() => transport.stop()}
				>
					<Unplug size={16} />
					Stop session
				</button>
				{store.error.value && (
					<p class="text-xs text-red-400">{store.error.value}</p>
				)}
			</div>
		);
	}

	return (
		<div class="flex flex-col gap-3">
			<p class="text-sm text-zinc-400">You are connected as guest.</p>
			{store.answerCode.value && (
				<>
					{showCode && (
						<textarea
							class="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-3 text-xs text-zinc-100 outline-none resize-none font-mono"
							rows={3}
							readOnly
							value={store.answerCode.value}
						/>
					)}
					<div class="grid grid-cols-2 gap-2">
						<button
							type="button"
							class="flex items-center justify-center gap-2 rounded-lg bg-zinc-800 px-4 py-2.5 text-sm text-zinc-100 hover:bg-zinc-700 transition-colors cursor-pointer"
							onClick={() => void transport.copy(store.answerCode.value)}
						>
							<Copy size={16} />
							Copy answer
						</button>
						<button
							type="button"
							class="flex items-center justify-center gap-2 rounded-lg border border-zinc-800 px-4 py-2.5 text-sm text-zinc-100 hover:bg-zinc-800 transition-colors cursor-pointer"
							onClick={() => setShowCode((v) => !v)}
						>
							{showCode ? <EyeOff size={16} /> : <Eye size={16} />}
							{showCode ? "Hide code" : "Show code"}
						</button>
					</div>
					<button
						type="button"
						class="flex items-center justify-center gap-2 rounded-lg border border-zinc-800 px-4 py-2.5 text-sm text-zinc-100 hover:bg-zinc-800 transition-colors cursor-pointer"
						onClick={() => (store.qrValue.value = store.answerCode.value)}
					>
						<QrCode size={16} />
						Show QR
					</button>
				</>
			)}
			<button
				type="button"
				class="flex items-center justify-center gap-2 rounded-lg bg-red-900/40 px-4 py-2.5 text-sm text-red-200 hover:bg-red-900/60 transition-colors cursor-pointer mt-2"
				onClick={() => transport.stop()}
			>
				<LogOut size={16} />
				Leave session
			</button>
			{store.error.value && (
				<p class="text-xs text-red-400">{store.error.value}</p>
			)}
		</div>
	);
}
