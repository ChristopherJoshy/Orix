# Landing art direction — 2026-11-03

Rebuilt the public landing page around an asymmetric editorial composition: pixel
headlines and a Kerala street illustration on quiet neutral paper. The original
ORIX SVG wordmark remains. Replaced the small hero vignette, orbital avatar
composition, repeated feature cards and filler copy with route-first content,
a real ordered sequence, a contrasting comfort section and native FAQ disclosures.
No invented usage statistics or testimonials. Existing Google login entry points
and public legal routes remain linked.

## Generated artwork

Tool: built-in `image_gen` (transparent background generation).
Runtime asset: `website/public/art/orix-street-corner.png`, 1122 × 1402 PNG.
CSS applies grayscale to remove minor generated colour fringes and pixelated
sampling when the image scales. The generated image is pixel-inspired raster art;
it is not claimed to use a mathematically consistent native sprite grid.

Prompt brief used: Create one original pixel-art editorial asset for ORIX travel
companions and shared autos in Kochi. Transparent background, no text, numbers,
logos or watermark. Strict achromatic palette: charcoal, medium gray, pale gray
and white. Handcrafted 1990s point-and-click aesthetic with crisp pixel clusters.
Street-level three-quarter auto-rickshaw, a young woman and man with backpacks
meeting alongside, and a driver. Tall coconut palm above, modest Kerala wall,
low house and utility pole behind. Curved asphalt corner and two lane marks form
an irregular pixel island. Portrait composition, strong dark focal auto below,
pale background elements and generous transparency. No isometric city map, phone
UI, gradients, colour cast or decorative noise.

## Responsive implementation and review status

Scoped stylesheet: `website/src/pages/Landing.css`. Desktop uses a wide two-column
hero and 1440px maximum content width; below 760px it stacks in reading order.
Headline sizing combines rem and viewport units, body measures are bounded,
CTA rows wrap, and layout columns permit shrinkage. FAQ uses native details/summary;
focus outlines, descriptive illustration text and 44px navigation targets provided.
The illustration is static and links respect reduced-motion preference.

Source and generated artwork were inspected. No build, automated test, browser
render or emulator QA was run for this landing change, respecting the user-supplied
current testing gate. Responsive/zoom visual verification remains pending at 320,
390, 760, 1024 and 1440+ pixels. This change is local and has not been deployed.
