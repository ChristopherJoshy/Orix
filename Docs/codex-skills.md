# Codex design and engineering skills

Installed on 2026-11-02 into `/home/chris/.codex/skills` for use across projects. They will be available for discovery on the next turn. The installed skill names below can also be invoked explicitly with `$name`.

## Installed set

| Skill | Purpose | Source |
| --- | --- | --- |
| `frontend-design` | Distinctive visual direction, typography, composition, critique | [Anthropic](https://github.com/anthropics/skills/tree/main/skills/frontend-design) |
| `algorithmic-art` | Seeded generative art, p5.js, parameter controls, exports | [Anthropic](https://github.com/anthropics/skills/tree/main/skills/algorithmic-art) |
| `ui-ux-pro-max` | Searchable design guidance, palettes, fonts, motion, accessibility, Flutter/web stacks | [NextLevelBuilder](https://github.com/nextlevelbuilder/ui-ux-pro-max-skill) |
| `vercel-react-best-practices` | React/Next.js performance and rendering | [Vercel](https://github.com/vercel-labs/agent-skills/tree/main/skills/react-best-practices) |
| `vercel-composition-patterns` | Maintainable React component APIs | [Vercel](https://github.com/vercel-labs/agent-skills/tree/main/skills/composition-patterns) |
| `vercel-react-view-transitions` | Route transitions and shared-element animation | [Vercel](https://github.com/vercel-labs/agent-skills/tree/main/skills/react-view-transitions) |
| `web-design-guidelines` | UI accessibility and interaction review | [Vercel](https://github.com/vercel-labs/agent-skills/tree/main/skills/web-design-guidelines) |
| `pixel-art-web` | Code-native sprites, tiles, palettes, animation, crisp web/Flutter rendering | Locally authored, instruction-only |
| `algorithm-design` | Invariants, complexity, graphs, scheduling, matching, profiling, oracle-based checks | Locally authored, instruction-only |
| `golang-pro` | Idiomatic Go, concurrency, cancellation, interfaces, profiling | [Jeff Allan](https://github.com/Jeffallan/claude-skills/tree/main/skills/golang-pro) |
| `flutter-apply-architecture-best-practices` | UI/data separation and scalable Flutter architecture | [Flutter team](https://github.com/flutter/agent-plugins) |
| `flutter-build-responsive-layout` | Adaptive layouts based on available constraints | Flutter team |
| `flutter-fix-layout-issues` | Constraint and overflow diagnosis | Flutter team |
| `flutter-setup-declarative-routing` | Declarative routing and deep links | Flutter team |
| `flutter-add-widget-test` | Widget behavior verification when authorized | Flutter team |
| `flutter-add-integration-test` | Integration and device verification when authorized | Flutter team |
| `dart-resolve-package-conflicts` | Dependency conflict diagnosis | [Dart team](https://github.com/dart-lang/skills) |
| `dart-fix-runtime-errors` | Runtime error investigation | Dart team |
| `security-best-practices` | Secure implementation/review guidance for Go, Python, JS/TS | [OpenAI](https://github.com/openai/skills/tree/main/skills/.curated/security-best-practices) |

Existing imagegen, browser, Figma, HyperFrames, and Codex Security plugin capabilities remain available. Security-best-practices supplements the existing security workflows; it does not provide Dart-specific security references. Project instructions and the user's testing gate take precedence over a skill's generic workflow.

## Research and selection

Consulted the [skills.sh directory](https://skills.sh/) and primary GitHub sources, downloaded eight candidate repositories at fixed commit IDs, and inspected the selected packages. Observed repository stars included Anthropic 186,666, UI/UX Pro Max 144,170, Vercel agent-skills 33,410, OpenAI skills 28,466, Jeff Allan 12,180, Flutter agent-plugins 3,139, and Dart skills 539. These are dated discovery signals, not security evidence or per-skill install counts. Official ownership, relevant contents, dependency burden, and compatibility carried more weight than popularity.

The [official Flutter tooling documentation](https://github.com/flutter/website/blob/main/sites/docs/src/content/ai/tools.md) identifies the Flutter and Dart repositories. Preferred these over the reviewed third-party `flutter-expert`, whose blanket Riverpod restrictions and major-version upgrade troubleshooting were unsuitable defaults.

Also examined pixel-art options from [Retro Diffusion](https://github.com/Retro-Diffusion/retro-diffusion-mcp), [Pixelbolt](https://github.com/skulitom/Pixelbolt), [Summer](https://github.com/SummerEngine/summer), and [abagames](https://github.com/abagames/agentic-gamedev-skills). The first three depend on services, a separate CLI, or an engine; the abagames asset workflow assumes additional image-processing tools. Installed a local code-native skill tailored to web and Flutter instead. Existing image generation remains available when raster generation is useful. No extra service/account setup was performed.

## Source review and adaptations

- Installed only selected skill directories. The trusted bundled skill-installer helper validated and copied the reviewed local files; it refused existing destinations. No repository install hooks, npm installers, remote shell scripts, MCP servers, or background services were installed.
- Inspected instruction risk patterns, file types, executable imports and side-effect sites. UI/UX Pro Max's five Python modules use standard-library code and local data. Search does not require network access. Optional persistence writes generated design documents and is not invoked during normal search.
- Excluded UI/UX Pro Max's development tests, platform templates/installers, and npm CLI. Assembled its canonical data, references, runtime modules, license, and entrypoint; changed its Claude plugin script path to the installed Codex location.
- Added an algorithmic-art adaptation giving the user's brand, component system, and requested application integration precedence over the upstream Anthropic-branded viewer defaults. The included HTML/JS examples are assets, not installation-time execution. Generated viewers may load p5.js/fonts from external hosts; inspect/pin dependencies for the actual project.
- Vendored [Vercel's interface guidelines](https://github.com/vercel-labs/web-interface-guidelines/blob/434b7f91364665f2f733b310ec54809bf8f37937/command.md) and replaced automatic moving-branch retrieval with local review guidance.
- Removed angle brackets from the React transitions metadata description to pass Codex's validator. Its implementation guidance still requires checking actual React/Next.js/browser support before use; installation does not upgrade any project dependency.
- Preserved available upstream licenses. The two local skills contain instructions only.

No suspicious credential collection, installation hooks, or hidden execution was identified in this review. This is source screening, not a guarantee of malware absence or an antivirus certification. Future upstream updates require a fresh review; no automatic updater was installed.

Exact source commits, adaptations, installed names, and SHA-256 hashes for all 193 files are in [codex-skills-manifest.json](codex-skills-manifest.json). Verification results are in [codex-skills-verification.json](codex-skills-verification.json).

## Actual verification

- All 19 installed entrypoints passed the bundled skill-creator metadata validator.
- All 193 installed files matched the reviewed manifest; no symlinks were present.
- UI/UX Pro Max returned nonempty JSON results for a UX query (`keyboard focus modal`), a Flutter query (`responsive layout`), and a generated design system (`premium creative portfolio`). Executed from `/tmp` with bytecode writing disabled.
- No ORIX code, dependencies, environment values, accounts, or runtime configuration were changed by this skills task. No Go/Python/Flutter application tests, analysis, builds, provider calls, or emulator QA were run. The app migration and its testing gate remain as previously documented.

## Useful prompts

> Use $frontend-design, $ui-ux-pro-max, and $pixel-art-web to build a premium pixel-art website. Establish a coherent art direction, palette, and native pixel grid. Integrate responsive layouts, readable typography, keyboard access, and reduced-motion behavior. Review the rendered result when testing is authorized.

> Use $algorithmic-art to build a deterministic interactive scene that fits my project's brand and component system. Expose useful parameters and keep animation performance bounded.

> Use $golang-pro and $algorithm-design for this matching algorithm. State its invariants, feasibility constraints, complexity, and tradeoffs; preserve atomic capacity and assignment constraints. Respect the project's testing gate.

> Use $flutter-build-responsive-layout and $flutter-fix-layout-issues to refine these screens. Preserve the existing architecture and state-management choices. Follow the current SDK and project testing gate.

> Use $security-best-practices to review the Go API's authentication and authorization boundaries. Use existing Codex Security workflows when a broader audit is requested.
