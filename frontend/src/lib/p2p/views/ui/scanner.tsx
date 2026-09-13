import { useEffect, useRef } from "preact/hooks";
import QrScanner from "qr-scanner";

export function Scanner({ onResult }: { onResult: (text: string) => void }) {
	const video = useRef<HTMLVideoElement>(null);
	const handler = useRef(onResult);
	handler.current = onResult;

	useEffect(() => {
		const el = video.current;
		if (!el) return;

		const scanner = new QrScanner(
			el,
			(result) => handler.current(result.data),
			{
				preferredCamera: "environment",
				highlightScanRegion: true,
				highlightCodeOutline: true,
				returnDetailedScanResult: true,
			},
		);

		scanner
			.start()
			.catch((err: unknown) => console.error("[p2p] camera:", err));
		return () => scanner.destroy();
	}, []);

	return (
		<div class="relative">
			<video
				ref={video}
				class="h-64 w-full rounded-lg bg-black object-cover"
				muted
				playsInline
			/>
		</div>
	);
}
