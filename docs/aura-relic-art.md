# Relic inspection atlas

Project asset: `web/public/aura/relics-atlas.webp`.

The built-in `image_gen` tool produced one original 4-by-3 item atlas. The
selected source stays intact in the personal image output folder. The project
uses its own WebP asset and does not depend on that folder at runtime.

The WebP is 1448 by 1086 pixels, with twelve equal 362 by 362 pixel tiles.
File size is 185,018 bytes, compared with 2,031,621 bytes for the source PNG.
Pillow saved RGB WebP at quality 88, method 6, as explicitly requested. There
was no resizing, cropping, compositing or change to the source image.

## Tile order

| Row | Column 1 | Column 2 | Column 3 | Column 4 |
| --- | --- | --- | --- | --- |
| 1 | `rusty_blade` | `moonlit_saber` | `dragon_fang` | `world_ender` |
| 2 | `troll_heart` | `world_shell` | `origin_crown` | `hourglass` |
| 3 | `echo_blade` | `executioner` | `void_reaver` | `storm_spear` |

The atlas has no gutters. For a square tile frame, use CSS background size
`400% 300%`. Column positions are `0%`, `33.333333%`, `66.666667%`, `100%`;
row positions are `0%`, `50%`, `100%`.

## Prompt

Use case: stylized-concept. Asset type: ONE production item atlas texture for
an original dark fantasy manhwa RPG. Create a landscape 4:3 image, ideally
2048 by 1536, with EXACTLY FOUR EQUAL COLUMNS and THREE EQUAL ROWS: 12 equal
square tiles, row-major order below. This is a single sprite atlas, not a UI
mockup. No text, lettering, numbers, captions, panel borders or gutters. All
twelve tile backgrounds must be the same nearly black navy (#090b16) and blend
seamlessly at their boundaries. Every tile contains ONE highly detailed,
clearly readable fantasy relic centered completely inside that tile, with
ample margin, no item crossing its tile boundary. Each item has a distinct
recognizable silhouette. Keep small painted magical glows local to the tile.

ROW 1, left to right: (1) battered iron short sword with chipped straight
blade, brown leather wrapped hilt, restrained cold steel reflections; (2)
elegant curved silver crescent saber with pale moonlight, ornate dark guard
and long graceful blade; (3) imposing ivory serrated dragon-bone greatsword,
fang-shaped blade and old bronze hilt; (4) broken black greatsword with
floating separated blade sections surrounding a brilliant purple core,
sinister royal detail.

ROW 2, left to right: (5) pulsing emerald anatomical troll heart suspended
inside a black iron protective cage, strong green aura, no gore; (6)
monumental black and gold chest armor with layered angular pauldrons and a
heavy gilded breastplate, no body or mannequin; (7) an ancient regal gold
crown with tall spires and one large dark violet void gemstone; (8) delicate
silver hourglass with clear glass bulbs and violet glowing sand, intricate
silver frame.

ROW 3, left to right: (9) three distinct slender silver blade fragments
orbiting a violet energy core, elegant echo magic, clearly different from
tile 4; (10) red obsidian executioner's seal pendant with a geometric carved
motif and crimson inner light, hanging from a short dark metal chain; (11)
huge black and violet reaper scythe with a long dark handle and sharp curved
crescent edge; (12) a long blue lightning spear with a silver split spearhead
and small electric arcs.

Style: premium original Korean fantasy webtoon item illustration, crisp
elegant ink contours, richly painted cel shading, believable polished and
weathered metal, detailed craftsmanship, dramatic magical rim light,
cinematic game inventory art. Same unified dark manhwa visual family across
all 12 items. Restrained violet, silver, gold, emerald and crimson accents
on deep navy black. Objects only, no characters, hands, scenery, cards,
frames, symbols outside the objects, logos, franchise references or words.
Enforce equal 4 columns x 3 rows composition and completely separate centered
objects.

## Verification

The source atlas was visually inspected: all twelve requested objects appear
in the intended positions, with distinct silhouettes and no labels or frames.
The saved WebP was reopened and checked for matching dimensions, format and
equal square tile geometry. Runtime item mapping is owned by `art.tsx` and
should retain the order above.
