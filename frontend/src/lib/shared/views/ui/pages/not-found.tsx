import { Link } from "wouter-preact";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";

export function NotFoundPage() {
	return (
		<DefaultLayout className="gap-4">
			<div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-3 text-sm">
				<p className="text-zinc-400">404 — Page not found</p>
				<Link
					href="/"
					className="cursor-pointer rounded-lg border border-zinc-700 px-3 py-1.5 text-xs text-zinc-300 transition-colors hover:bg-zinc-800"
				>
					Back home
				</Link>
			</div>
		</DefaultLayout>
	);
}

export default NotFoundPage;
