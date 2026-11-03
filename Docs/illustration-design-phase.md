# Illustration design phase — 2026-11-03

The owner approved the original Kerala street illustration and requested the same
visual family across both clients, with different illustrations for app and web.
Used the built-in image_gen tool with the approved street PNG as a style reference.
The original logo and strong code-native animations are preserved. Functional route
art remains because it carries real route labels rather than decoration alone.

## Selected generated assets

Website only:
- `website/public/art/orix-web-welcome.png`: two travellers meeting at a chaya shop,
  shopkeeper, tiled roof, palm and parked auto. Login illustration, 1448×1086.
- `website/public/art/orix-web-companions.png`: three travellers coordinating beside
  a low wall, palm and bench. Onboarding introduction and About, 1448×1086.

Android app only:
- `frontend/assets/images/orix_app_welcome.png`: two travellers greeting at a ferry
  landing with palm, bollard and distant ferry. Login illustration, 1254×1254.
- `frontend/assets/images/orix_app_companions.png`: two women sharing an umbrella
  at a Kerala pickup point with banana plant, wall and puddle. Introduction and
  About illustration, 1254×1254.

Rejected the bus-shelter draft and its attempted cleanup because the surrounding
haze did not match the approved clean silhouette. These drafts are not shipped.
Selected artwork was copied into project assets; source generation files stay in
the tool's generated_images directory. No Python raster editing used.

## Prompts / rendering contract

Each prompt requested a new scene in the approved image's handcrafted nostalgic
point-and-click pixel style: deliberate stepped contours, strong character
silhouettes, economical pale background, complete figures and a finite ground
island on transparency. Strict charcoal/middle-gray/pale-gray/white palette; no
text, logos, numbers, interface, watermark, gradients or painterly blur.

Web welcome prompt: a compact horizontal4:3 chaya shop with tiled awning; two
backpacking young adult travellers meeting, one holding tea; shopkeeper behind
counter; palm/banana leaves and half an auto at the edge. Understated candid
expressions, strong foreground and pale background, premium minimal login art.

Web companions prompt: three young adult travellers meeting at a roadside pickup;
one with phone, one gesturing down the road; low wall, small palm, bench and pale
distant tiled roof. No vehicle/shop/tea. Horizontal4:3, readable at280px.

App welcome prompt: a distinct compact1:1 Kerala ferry landing; two backpacking
travellers greeting; stone quay, mooring bollard, distant pale ferry and slender
palm. No auto or shop. Readable at260px, full figures, clean transparent island.

App introduction prompt: two young women travellers sharing a large umbrella;
one holds phone by her side, banana plant, low pale wall, puddle and paving. No
building/shelter/ferry/shop/auto. Compact square, complete umbrella and feet,
welcoming Kerala monsoon moment, abrupt pixel silhouette without haze or glow.

Web images use grayscale and pixelated rendering. Flutter uses a grayscale
ColorFilter and FilterQuality.none. Both retain source transparency. These are
pixel-inspired raster illustrations, not claimed to have an exact native sprite
grid. Illustrations are decorative; names, routes, actions and errors remain real
semantic UI. No mock user records or fake journey outcomes are introduced.

## Composition and verified scope

Web login: generous desktop split, art left and Google-only sign-in right; compact
windows stack a smaller illustration above the sign-in block. Original logo stays.
App login: distinct ferry art between heading and controls on compact windows;
large windows place the art beside the form. Both layouts scroll when enlarged
text needs more height. Onboarding art appears only in the introductory step, not
in every form step or over the user's own photo.

Design-only verification explicitly requested despite the wider testing hold:
- Website TypeScript/Vite build passed.
- Browser login reviewed at1440×1000 and390×844 with screenshots. No horizontal
  overflow at320px (sign-in button bottom534px in568px viewport) or900px with200%
  root text. Images loaded and Google-only action remained available.
- Two Flutter illustration/layout tests passed:320×568,390×844,900×700 at1x/2x
  text, scroll-to-sign-in reachability, and narrow introduction asset loading.
- Illustration APK debug build/install succeeded; visible emulator login inspected
  with the actual app-only ferry illustration. No login interaction performed.
- Broad backend/integration/regression verification remains paused at owner's
  request. This phase does not change the outstanding implementation status.

Screenshots: `/tmp/orix-illustrated-web-login-desktop.png`,
`/tmp/orix-illustrated-web-login-phone.png`, `/tmp/orix-illustrated-app.png`.
