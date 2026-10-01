import type { ComponentChildren, JSX } from "preact";
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
	// Replaces the image once it has failed to load. Without it a broken src
	// leaves an empty box, which reads as "still loading" next to real covers.
	fallback?: ComponentChildren;
};

export function Image({
	class: cls,
	fallback,
	onLoad,
	onError,
	style,
	alt,
	src,
	...props
}: Props) {
	// ready and failed are tracked together so a new src can restart both.
	const [load, setLoad] = useState({ src, ready: false, failed: false });
	if (load.src !== src) setLoad({ src, ready: false, failed: false });

	const bound = (e: JSX.TargetedEvent<HTMLImageElement, Event>) => {
		if (e.type === "load") {
			setLoad((l) => ({ ...l, ready: true, failed: false }));
			onLoad?.(e);
		} else {
			setLoad((l) => ({ ...l, failed: true }));
			onError?.(e);
		}
	};

	// No source and a failed source look the same to a reader: there is no
	// artwork to show, so stop pretending something is on its way.
	if (!src || load.failed) return <>{fallback ?? null}</>;

	return (
		<img
			{...props}
			src={src}
			alt={alt ?? ""}
			onLoad={bound}
			onError={bound}
			class={cn(cls, !load.ready && "shimmer")}
			style={typeof style === "object" ? style : undefined}
		/>
	);
}
