# SruAura art assets

## Raster character atlas

Project asset: `web/public/aura/hunters-atlas.webp`.

Technique: built-in `image_gen`, one original square 2-by-2 atlas. The image is
copied into the project; the application has no dependency on an external image
host or the original personal output folder. Four tiles contain the hunter,
crypt warden, forge knight and void entity. `art.css` selects the tile through
CSS background positioning, without changing the image.

Prompt:

> Use case: stylized-concept. Asset type: a SINGLE square 2-by-2 character
> portrait atlas texture for an original dark fantasy manhwa RPG called SruAura.
> Four equally sized square panels, zero gutters, clean exact quarter boundaries.
> NOT a UI mockup, no text, no labels, no lettering, no logos, no panel borders.
> Top left: original young adult male hunter with short windswept black hair,
> visible serious face, long layered black combat coat with fine silver armor
> details, holding a luminous violet sword diagonally downward, framed from
> knees up. Top right: formidable skeletal crypt warden, ornate weathered black
> iron armor, skull face with brilliant ice-blue eyes, crown-like branching
> helmet, pale blue runic greatsword, framed knees up. Bottom left: giant armored
> forge knight with layered jagged basalt plates, glowing molten orange cracks
> and a furnace heart, immense heavy blade, framed knees up. Bottom right:
> ominous void entity, regal hood and broken floating black armor, faceless
> star-filled head with bright violet eyes, black tendrils and cosmic violet
> energy, framed knees up. Each portrait has one clear central subject with head
> fully inside its own tile and enough breathing room, subdued dark atmospheric
> background matching character: violet ruined gate, flooded crypt, ember forge,
> cosmic rift. Style: premium original Korean fantasy webtoon concept art,
> elegant sharp ink contours, richly painted cel shading, dramatic rim lighting,
> readable expressive silhouettes, refined detailed costumes, cinematic volume
> and magical particulate, polished commercial videogame key art. Unified
> graphite/navy black palette with restrained violet, icy blue and molten gold
> accents. Each subject should stand out from background and fill its own
> square. No franchise characters. Produce high resolution square image.

## Code-native illustrations

`web/src/apps/aura/art.tsx` provides `HunterArt`, `EnemyArt`, `DungeonArt` and
`RelicArt`. Gate illustrations use distinct stone, fire and cosmic environments.
Relic icons use matching inline SVG metal shading, rank colors and recognizable
weapon, armor, hourglass, heart, crown and pendant shapes. SVG gradient IDs are
unique per component instance. Each illustration has an accessible image label.

The only animated art is the hunter's small light particles. Reduced-motion
preferences disable this animation. Character portraits are static; combat
feedback belongs to the page layer.

## Silent Warden

`web/public/aura/silent-warden.webp` is an original standalone boss portrait,
generated with built-in image_gen for the false S-rank gate. Prompt: a monumental
stone executioner in angular obsidian cathedral armor, ice-violet eyes, broken
gold halo and ceremonial iron blade, a tiny hunter for scale, flooded dark
cathedral, violet cracks, cold rim lighting, fine webtoon ink contours and painted
cel shading, no text or franchise characters. The output was copied into the
workspace and visually checked against the original atlas.

## Delivery compression

Original PNG assets totalled 6,201,819 bytes. Pillow WebP quality 84, method 6
produced 671,114 bytes at unchanged dimensions, saving 5,530,705 bytes (89.2%).
Runtime CSS references only WebP. PNG source versions remain in Git history
and generated source storage, outside the public build.
