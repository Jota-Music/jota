import { useLocation } from "wouter-preact";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";
import useMeta from "@/lib/shared/views/hooks/use-meta";

export function MainPage() {
    const [, setLocation] = useLocation();
    useMeta(
        "Jota | Free music self-hosted service",
        "Jota is a free, self-hosted music service. Stream your music anywhere, anytime.",
    );

    let input: HTMLInputElement | null | undefined;

    const handleSubmit = (e: Event) => {
        e.preventDefault();
        const value = input?.value?.trim();
        if (value) {
            setLocation(`/${value}`);
        }
    };

    return (
        <DefaultLayout class="gap-6 h-full">
            <div class="flex flex-col gap-6 h-full">
                <header>
                    <h2 class="text-xl font-semibold leading-tight">Jota</h2>
                    <p class="text-sm opacity-70 mt-1">
                        Free music self-hosted service
                    </p>
                </header>

                <form onSubmit={handleSubmit} class="flex gap-2">
                    <input
                        ref={(el) => { input = el }}
                        type="text"
                        placeholder="Enter a username..."
                        class="h-10 w-full rounded-md border border-zinc-800 bg-zinc-950 pl-3 text-sm text-white outline-none focus:border-zinc-600"
                    />
                    <button
                        type="submit"
                        class="rounded-lg px-4 py-2 font-bold text-sm text-(--binary-color) bg-(--dominant-color) hover:opacity-75 transition-opacity cursor-pointer"
                    >
                        Go
                    </button>
                </form>
            </div>
        </DefaultLayout>
    );
}

export default MainPage;