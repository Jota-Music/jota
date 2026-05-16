import { useQuery } from "@tanstack/preact-query";
import { Link, useParams } from "wouter-preact";
import getUserPlaylists, {
  type PlaylistSummary,
} from "@/lib/music/app/get-user-playlists";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";

export function UserPage() {
  const { user } = useParams<{ user: string }>();

  const { data, isLoading, isError } = useQuery({
    queryKey: ["user-playlists", user],
    queryFn: () => getUserPlaylists(user ?? ""),
    enabled: !!user,
  });

  const playlists = data ?? [];

  if (isError) {
    return (
      <DefaultLayout>
        <div class="p-6 text-sm rounded-md">Error loading playlists</div>
      </DefaultLayout>
    );
  }

  return (
    <DefaultLayout class="gap-6 h-full">
      <div class="flex flex-col gap-6 h-full">
        <header>
          <h2 class="text-xl font-semibold leading-tight">
            Playlists de {user}
          </h2>
        </header>

        {isLoading ? (
          <div class="p-4 rounded-md">Cargando playlists...</div>
        ) : playlists.length === 0 ? (
          <div class="p-4 rounded-md">No se encontraron playlists.</div>
        ) : (
          <div class="grid grid-cols-2 lg:grid-cols-3 gap-8">
            {playlists.map((p: PlaylistSummary) => (
              <Link key={p.id} href={`/playlist/${p.id}`}>
                <div class="rounded-md overflow-hidden cursor-pointer">
                  <div class="flex flex-col gap-3">
                    <h3 class="font-medium truncate">{p.name}</h3>

                    <div class="aspect-square w-full overflow-hidden rounded-md">
                      <img
                        src={"mosaic" in p ? p.mosaic : p.cover}
                        alt={p.name}
                        class="w-full h-full object-cover"
                      />
                    </div>
                  </div>
                </div>
              </Link>
            ))}
          </div>
        )}
      </div>
    </DefaultLayout>
  );
}
