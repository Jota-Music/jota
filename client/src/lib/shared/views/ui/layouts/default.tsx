import type { PropsWithChildren } from "preact/compat";
import { Player } from "@/lib/music/views/ui/player";
import Queue from "@/lib/music/views/ui/queue";
import { cn } from "@/lib/shared/utils/tw";
import { Header } from "@/lib/shared/views/ui/components/header";
import { ErrorBar } from "@/lib/shared/views/ui/error-bar";

function DefaultLayout({
  children,
  className,
  class: _class,
}: PropsWithChildren & { className?: string; class?: string }) {
  return (
      <div
        class={cn(
          "h-dvh flex flex-col pb-6",
          className,
          _class,
        )}
      >
        <div class="flex flex-col gap-6 sticky top-0 z-50 bg-stone-950 pb-6 ">
          <Header />
        
          <div class="shrink-0 w-full overflow-hidden max-w-2xl px-4 md:px-0 mx-auto">
            <Player />
          </div>

          {/* <div class="bg-red-400 z-50 absolute top-full h-16">

          </div> */}
        </div>


        <Queue />

        <main class="flex min-h-0 flex-1 flex-col min-w-0 overflow-hidden w-full max-w-2xl px-2 md:px-0 mx-auto">{children}</main>

        <ErrorBar />
      </div>
  );
}

export default DefaultLayout;