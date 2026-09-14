import type { PropsWithChildren } from "preact/compat";
import { cn } from "@/lib/shared/utils/tw";
import { AppShell } from "@/lib/shared/views/ui/layouts/app-shell";

function ClearLayout({
	children,
	className,
	class: _class,
}: PropsWithChildren & { className?: string; class?: string }) {
	return (
		<AppShell rootClass={cn("pb-6", className, _class)}>{children}</AppShell>
	);
}

export default ClearLayout;
