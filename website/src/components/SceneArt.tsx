const scenes = {
  welcome: "/art/orix-web-welcome.png",
  companions: "/art/orix-web-companions.png",
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
      width="1448"
      height="1086"
      alt=""
      loading={priority ? "eager" : "lazy"}
      fetchPriority={priority ? "high" : "auto"}
      decoding="async"
    />
  );
}
