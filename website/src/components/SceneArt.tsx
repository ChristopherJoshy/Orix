const scenes = {
  welcome: "/art/orix-web-welcome.webp",
  companions: "/art/orix-web-companions.webp",
};
/** Decorative editorial scenes; real route and identity information stays in text. */
export function SceneArt({
  scene,
  className = "",
  priority = false,
}: {
  scene: keyof typeof scenes;
  className?: string;
  priority?: boolean;
}) {
  return (
    <img
      className={`orix-scene-art ${className}`}
      src={scenes[scene]}
      srcSet={
        scene === "companions"
          ? "/art/orix-web-companions-384-alpha.webp 384w, /art/orix-web-companions-768-alpha.webp 768w, /art/orix-web-companions-1448-alpha.webp 1448w"
          : undefined
      }
      sizes={
        scene === "companions" ? "(max-width: 420px) 90vw, 380px" : undefined
      }
      width="1448"
      height="1086"
      alt=""
      loading={priority ? "eager" : "lazy"}
      fetchPriority={priority ? "high" : "auto"}
      decoding="async"
    />
  );
}
