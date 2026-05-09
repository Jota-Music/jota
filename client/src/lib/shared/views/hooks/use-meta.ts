import { effect, signal } from "@preact/signals";

export const metaTitle = signal<string>(
  "Jota | Free music self-hosted service",
);

export const metaDescription = signal<string>(
  "Jota is a free, self-hosted music service. Stream your music anywhere, anytime.",
);

effect(() => {
  document.title = metaTitle.value;

  let $metaDescription = document.querySelector(
    "meta[name='description']",
  ) as HTMLMetaElement | null;

  if (!$metaDescription) {
    $metaDescription = document.createElement("meta") as HTMLMetaElement;
    $metaDescription.name = "description";
    document.head.appendChild($metaDescription);
  }

  $metaDescription.content = metaDescription.value;
});

export default function useMeta(title: string, description: string) {
  metaTitle.value = title;
  metaDescription.value = description;
}
