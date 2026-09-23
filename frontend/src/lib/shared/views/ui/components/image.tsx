import type { JSX } from "preact";
import { useState } from "preact/hooks";
import { cn } from "@/lib/shared/utils/tw";

type Props = Omit<
	JSX.ImgHTMLAttributes<HTMLImageElement>,
	"class" | "style" | "onLoad" | "onError"
> & {
	class?: string;
	style?: string | JSX.CSSProperties;
	onLoad?: JSX.GenericEventHandler<HTMLImageElement>;
	onError?: JSX.GenericEventHandler<HTMLImageElement>;
};

// WebKit paints a bordered empty box for src-less <img>. A transparent 1x1 PNG
// keeps the shimmer background clean until a real source shows up.
const EMPTY =
	"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR4nGNgYGBgAAAABQABpfZFQAAAAABJRU5ErkJggg==";

export function Image({
	class: cls,
	onLoad,
	onError,
	style,
	alt,
	src,
	...props
}: Props) {
	const [ready, setReady] = useState(false);
	const hasSrc = !!src;

	const bound = (e: JSX.TargetedEvent<HTMLImageElement, Event>) => {
		setReady(true);
		e.type === "load" ? onLoad?.(e) : onError?.(e);
	};

	return (
		<img
			{...props}
			src={hasSrc ? src : EMPTY}
			alt={alt ?? ""}
			onLoad={hasSrc ? bound : undefined}
			onError={hasSrc ? bound : undefined}
			class={cn(cls, !ready && "shimmer")}
			style={typeof style === "object" ? style : undefined}
		/>
	);
}
