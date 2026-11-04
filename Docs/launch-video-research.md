# ORIX launch video: research and tooling

2026-11-04. Setup only; production remains paused.

## Owner direction

High-energy, colorful comic film using
`Videos/Orix-Launch/Orix-theme-SOng-Amal-Kl-10.mp3`. Synchronize edits and motion
to the rhythm and energy, not lyric meaning. Color explicitly supersedes the
monochrome direction for the film; app/website styling has not changed. Existing
app and website artwork may be used, and focused capture tests are authorized.

The owner required skill installation and script approval before production,
then requested installation only and said no scripts were needed. No script was
written or approved. Do not infer permission to start captures, animation or
rendering from the setup-only request. Broad application QA remains on hold.

## Installed

Official npm latest stable was checked: HyperFrames CLI **0.9.10**, successfully
invoked with `npx --yes hyperframes@0.9.10 --version`. The bare `hyperframes`
executable is the desktop app; attempts to pass CLI commands to it hit an Electron
sandbox failure. Use npx for CLI work. No product dependencies were upgraded.

Official HyperFrames skill check at v0.9.10: **9 current, 0 outdated, 0 missing**.
Existing: hyperframes, hyperframes-cli, hyperframes-registry, media-use, figma,
slideshow and remotion-to-hyperframes. Newly installed from that release:

- `reverse-any-video`: optional reference-film reconstruction workflow.
- `style-sampling`: optional style/template workflow.

Also installed official `remotion-dev/skills` packages:

- `remotion-best-practices`, declared skill version 4.0.540, including bundled
  production references.
- `remotion-multimedia`, declared skill version 4.0.540, for media inspection.

All four were downloaded with the skill-installer helper into
`/home/chris/.agents/skills`; actual SKILL.md files and references were inspected.
New skills are discoverable on the next turn. HyperFrames remains the intended
renderer; no alternate Remotion project was created. Optional reverse/style
services require HyperFrames Pro; installation does not purchase or enable them.
They are not necessary to author an original motion comic.

Imagegen and agent-browser skills already exist; agent-browser CLI is 0.41.1.
No browser capture or account interaction was performed. No additional unknown
third-party skill bundles were installed.

## Research findings

The maintained HyperFrames core covers launch films, music-led edits, character
motion, sound and review. Some leaderboard names for older separate animation,
creative and audio skills no longer resolve in the current official tree. Prefer
official current sources over discovery rankings and unsupported quality claims.

Recommended craft, pending creative approval: original Kerala comic inks,
halftone/hatch shading, bold yellow/turquoise/red/green, layered parallax,
expressive held poses, panel transitions and selective pixel accents. Sony's
production account describes stepped timing and graphic shading; Toon Boom's
artist interview describes effects designed per shot. These inform general craft,
not copying franchise artwork or characters. IBM's motion guidance supports clear
staging and anticipation rather than competing effects.

After production approval, measure the supplied audio's actual strong transients,
energy and section changes. Vary cut density with the track. Exact BPM, duration,
excerpt and cut timestamps remain unmeasured; no sync claim is made. Show real
product screens and the original logo. Decorative generated actors are never
fabricated app users or matches. Follow current integration limits; no guaranteed
matches, measured savings, safety guarantees or Play availability claims.

## Actual checks and limits

The first local-only audio query failed because ffprobe was missing. It returned
an item-level error despite exit code zero; it is not a successful analysis.
No audio upload occurred. Initial HyperFrames doctor reported available Node,
Chrome cache and resources, but missing FFmpeg/ffprobe. Optional transcription,
TTS, music generation and Docker were also absent; none is needed for the current
supplied-music motion-comic plan. Do not install Docker for this task.

After the owner installed FFmpeg, both version commands and HyperFrames doctor
confirmed FFmpeg/ffprobe 8.1.2-2ubuntu2. Node and cached Chrome also passed their
dependency checks. Doctor's overall `ok` remains false for absent optional
transcription/TTS/music-generation and Docker checks. No video runtime/render has
been verified. Doctor notes that its loudness numbers are comparable only on
FFmpeg8.0.1; do not compare this machine's numbers to that reference baseline.
Two sandboxed npx dependency probes failed DNS resolution; the network-authorized
probe succeeded. This was tooling diagnosis, not an application test.

One built-in image_gen colorful concept request began before the owner's
production hold and finished afterwards. It is an unapproved visual study only,
left in the tool's generated_images directory, not a selected repository asset.
Prompt: colorful Kerala auto beside a chaya shop, tiled houses, palms, monsoon
puddle reflections, two backpacking adults; wide low camera, layered diagonals,
bold editorial comic inks, halftones, pixel accents; yellow/turquoise/vermilion/
green/indigo; no lettering, logo, UI, franchise, watermark or unsafe passengers.

No application tests, screenshot captures, script, composition, animation,
preview, render, deployment or publication. User-supplied audio is untouched and
excluded from the documentation commit. No credential/config files were staged.

## Sources reviewed

- [Official HyperFrames repository](https://github.com/heygen-com/hyperframes)
- [Current skill installation and catalog](https://github.com/heygen-com/hyperframes/blob/main/docs/guides/skills.mdx)
- [CLI and audio analysis documentation](https://github.com/heygen-com/hyperframes/blob/main/docs/packages/cli.mdx)
- [Registry skill](https://github.com/heygen-com/hyperframes/blob/main/skills/hyperframes-registry/SKILL.md)
- [Reverse-video workflow](https://github.com/heygen-com/hyperframes/blob/main/skills/reverse-any-video/SKILL.md)
- [Style workflow and Pro limits](https://github.com/heygen-com/hyperframes/blob/main/skills/style-sampling/SKILL.md)
- [Official Remotion skills](https://github.com/remotion-dev/skills)
- [Agent-browser documentation](https://agent-browser.dev/)
- [Sony Imageworks: comic animation production](https://stg.imageworks.com/node/1371)
- [Toon Boom: interview with effects artists](https://www.toonboom.com/behind-the-amazing-2dfx-in-spider-man-across-the-spider-verse)
- [IBM: animation principles](https://www.ibm.com/design/language/animation/classic-principles/)
- [GSAP timelines](https://www.gsap.com/docs/v3/GSAP/gsap.timeline%28%29/)
- [Skills directory, discovery only](https://skills.sh/)

Next: await explicit owner direction before creative production. Preserve the
prior approval gate unless the owner clearly lifts it.
