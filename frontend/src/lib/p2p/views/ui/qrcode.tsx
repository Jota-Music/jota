import { useMemo } from "preact/hooks";
import { encode } from "uqr";

export function QRCode({
	value,
	class: className,
}: {
	value: string;
	class?: string;
}) {
	const qr = useMemo(() => {
		try {
			return encode(value, { ecc: "L", border: 4 });
		} catch {
			return null;
		}
	}, [value]);

	if (!qr)
		return <p class="text-xs text-red-400">Code too large for QR, use copy.</p>;

	let path = "";
	for (let y = 0; y < qr.size; y++) {
		let x = 0;
		while (x < qr.size) {
			if (!qr.data[y][x]) {
				x++;
				continue;
			}
			let len = 1;
			while (x + len < qr.size && qr.data[y][x + len]) len++;
			path += `M${x},${y}h${len}v1h-${len}z`;
			x += len;
		}
	}

	return (
		<svg
			viewBox={`0 0 ${qr.size} ${qr.size}`}
			class={className}
			role="img"
			aria-label="QR code"
			shape-rendering="crispEdges"
		>
			<rect width={qr.size} height={qr.size} fill="#ffffff" />
			<path d={path} fill="#000000" />
		</svg>
	);
}
