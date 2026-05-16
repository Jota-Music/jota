import { effect, signal } from "@preact/signals";
import { currentSong } from "@/lib/music/views/stores/audio";

const DEFAULT_TITLE = "Jota | Free music self-hosted service";
const DEFAULT_DESCRIPTION = "Jota is a free, self-hosted music service. Stream your music anywhere, anytime.";

export const metaTitle = signal<string>(DEFAULT_TITLE);
export const metaDescription = signal<string>(DEFAULT_DESCRIPTION);

const DEFAULT_FAVICON = "/favicon-32x32.png";

function getFaviconLink(): HTMLLinkElement {
  let link = document.querySelector<HTMLLinkElement>(
    "link[rel='icon'][sizes='32x32']",
  );
  if (!link) {
    link = document.createElement("link");
    link.rel = "icon";
    link.sizes = "32x32";
    document.head.appendChild(link);
  }
  return link;
}

function setFavicon(url: string): void {
  getFaviconLink().href = url;
}

function resetFavicon(): void {
  setFavicon(DEFAULT_FAVICON);
}

effect(() => {
  const song = currentSong.value;
  const pageTitle = metaTitle.value;
  const description = metaDescription.value;

  if (song) {
    const artist = song.artists?.[0]?.name ?? "";
    document.title = `${song.name}${artist ? ` · ${artist}` : ""} | Jota`;
  } else {
    document.title = pageTitle;
  }

  let $metaDescription = document.querySelector(
    "meta[name='description']",
  ) as HTMLMetaElement | null;

  if (!$metaDescription) {
    $metaDescription = document.createElement("meta") as HTMLMetaElement;
    $metaDescription.name = "description";
    document.head.appendChild($metaDescription);
  }

  $metaDescription.content = description;
});

effect(() => {
  const song = currentSong.value;
  const cover = song?.album?.covers?.[0];

  if (cover) {
    setFavicon(cover);
  } else {
    resetFavicon();
  }
});

export default function useMeta(title: string, description: string) {
  metaTitle.value = title;
  metaDescription.value = description;
}
