# SruFarm

An independent game at `/apps/farm`, registered in the existing catalogue.
The initial game interface is in Russian; the catalogue description supports
the shell's Russian, Kazakh and English language settings.
Design: warm illustrated garden, cream canvas, forest green ink, lime highlights,
ochre coins. Existing system typeface, an 8px spacing rhythm, 16-24px panels,
small feedback animations and reduced-motion support. Mobile first; on desktop
the garden and barn sit side by side. Existing shell and other apps are preserved.
Visual variance 5, motion 3, density 5, asset dependence 5, shell fidelity 10.

## Rules

- Four initial plots, expandable to eight. Each crop takes exactly three real hours. Mature crops never rot.
- A new account receives 250 coins, seven seeds and two mature welcome plants.
- Eighteen base species and nine craftable hybrids, three qualities, eight mutation states (including ordinary).
- Care is selected at planting: water (free, better quality), fertilizer (12 coins,
  one extra plant), or lamp (20 coins, five extra percentage points of mutation).
- Water, soil and lamp upgrades have five levels. Cost: `100 * (level + 1)^2`.
  Their effects are snapshotted at planting, so later upgrades cannot reroll crops.
- Land adds one plot per level (four levels, base cost 300). Market adds 5% sale
  revenue per level (five levels, base cost 200). Selection adds one percentage
  point of legendary seed probability per level (five levels, base cost 250).
  Genetics adds 2 percentage points to mutation chance per level (five levels, base 350),
  unlocking electric at level 1, ancient at 2 and rainbow at 3. Nursery returns one
  seed per harvested plot with a 10% chance per level (five levels, base 180).
  Gallery adds two trophy slots per level (three levels, base 300), up to twelve.
  Upgrade cost is `base * (level + 1)^2`; server metadata supplies actual levels,
  caps and prices. Personalized weights are returned by the same logic used to roll.
- Plants sell as a batch. Discoveries persist after sale. Collection is 27 species
  times 8 mutation states, 216 entries. Quality affects value, not album completion.
- Packs each contain three seeds. Per-seed probabilities are returned by the API.
  Garden/exotic guarantee at least rare after ten seeds without a qualifying drop.
  The lunar pack costs 900 coins, has no common seeds, a base 30% legendary chance
  and guarantees legendary after six seeds without one. Each pack has a separate
  counter. A qualifying natural drop resets it. Packs use earned game coins.
- The shared top ranks actual harvest revenue for the UTC week starting Monday.
  Ties use user ID. No fictional opponents or automatic prize distribution in v1.

## Server authority

`farm_players` stores each player's state in JSONB. Every mutation locks that
player's row in a PostgreSQL transaction; sale and weekly score commit together.
The client sends only action intent, never currency, prices, rolls or timestamps.
Randomness comes from `crypto/rand`. Harvest checks server time. Authentication
uses the existing Telegram session. Local preview uses the existing dev login.

## Running

Use `docker compose -p srunal-local -f compose.local.yml up -d` for the local
database/API and `npm run dev -- --host 127.0.0.1` inside `web` for the client.
After backend changes, restart the API container to apply migrations and rebuild.
The first two welcome crops can be collected immediately; new plantings take 3h.

## Feedback

Short synthesized sound effects accompany successful planting, harvest, sale,
seed purchases, pack openings and upgrades. Rare drops get a different melody.
Audio starts only after a user gesture; the header toggle persists the preference.
There is no background music or external audio dependency. Individual cues last
under two seconds. Seedlings sway, planting pops into place, harvest highlights the barn,
sales animate the wallet, and pack drops reveal in sequence. Reduced-motion
preferences disable all game animations. `node --test src/apps/farm/sounds.test.ts`
checks audio duration, gain, node cleanup and suspended-context behavior.

Pack purchases enter a dedicated reveal scene. A faceted seed opens after a tap
with 2-5 distinct bell strikes: pitch, loudness, speed and light intensity rise.
Higher rarity gets more beats, color transitions, cracking, a brief silence and
a final impact chord. A summary shows all three
seeds; skipping or closing the scene cannot lose rewards, because the server
credits them in the purchase transaction before the animation begins. Revealing
or replaying an animation never rolls or buys again. Reduced motion disables
visual effects. New species: radish, cucumber, corn, sunflower, watermelon and
legendary moonflower. Pack weights and the guarantee use species rarity, so adding
a new common seed cannot accidentally reset the rare-seed guarantee.

## Breeding, trophies and worlds

The laboratory consumes two copies of a base seed and creates one hybrid seed
after three server-timed hours. Nine deterministic recipes cover all eighteen base
species. Hybrids are craft-only, not sold for zero coins or added to pack odds.
Each has its own hue and markings; progress survives restart and can be claimed
only once. There is one laboratory job at a time.

New harvest batches carry unique IDs and specimen weight. Keeping a trophy moves
exactly one plant and its unit value out of the barn. Six display slots are
available initially, expandable to twelve with the gallery upgrade. Selling harvest never sells trophies; returning a trophy
moves the same specimen back to the barn. Legacy harvest receives stable local
IDs and displays an unknown weight instead of inventing a record. Collection
quality retains the best observed grade.

Plants have mutation- and species-specific character effects: large specimens
take more room, golden and crystal plants sparkle, moon plants glow, and ember
plants shed tiny sparks. Harvest sound follows the specimen's mutation.

Six world palettes cover navigation, panels, shop, dialogs, laboratory, showcase,
buttons, scenery and CSS atmosphere. New worlds are enchanted forest, autumn
and frost, unlocked locally by `FOREST`, `AUTUMN`, `FROST`. Existing Stars and
promo-code transactions support them through migration 000013. The mobile menu
uses two visible rows rather than hiding the new sections offscreen.

## Genetics and recovery

There is at most one mutation per plant. Lamp and genetics levels plus care give
`3 + 2*lamp + 2*genetics + (lamp care ? 5 : 0)` percent chance, at most 28%.
An independent weighted roll then selects from unlocked mutation types:
giant x2, dew x2, gold x5, electric x6, crystal x8, ancient x10, rainbow x12.
Classic mutations remain obtainable at every genetics level. Mutation metadata
supplies value multipliers and unlock levels to the client. Dew droplets,
lightning arcs, ancient runes and a rainbow halo distinguish the new forms.
Reduced-motion preferences disable their animations.

Genetics and nursery levels are saved with the plot at planting. A nursery roll
returns at most one seed of the harvested species, regardless of yield, quality
or mutation. It can also recover a hybrid seed. The harvest transaction consumes
the plot once and reports any recovered seeds in the success message.

## Levels and rewards

Harvest gives ten XP per plot. Orders, daily rewards and expeditions also grant XP. The curve is
`100 + (level-1)*40` per level, capped at level 10; XP keeps accumulating past
the cap so the bar never freezes mid-track. The server owns the curve and
returns `level`, `title`, `into`, `need`, `max`, `pending` and the full reward
table, so the client never recomputes progression.

Rewards are claimed once each, oldest pending first, and every claim is
rejected if the level was not reached or was already paid. Coins and seeds are
obvious; the two plot rewards raise the land upgrade level rather than appending
beds directly, so the existing cap and price logic stays authoritative. If the
garden is already at the land cap, the plot reward pays its base cost in coins
instead of vanishing.

Titles are cumulative milestones: Новичок at 1, Садовод at 3, Хозяин фермы at 5,
Мастер урожая at 7, Легенда сада at 9.

## Village orders

The barn is not a single "sell everything" button. Each player gets three
orders a day, one from each difficulty tier: an easy common crop, a mid order
that may require choice quality, and a hard one that may demand a specific
mutation. They pay above market, so holding a batch is a real decision.

The board is generated deterministically from the player ID and the UTC date,
so it is stable all day without being written until first requested, and two
players see different boards. Fulfilment consumes exactly the requested units,
keeping each batch's unit value consistent so a partial take does not inflate
the rest, and an emptied batch leaves the barn. Each order pays once; the
fulfilled list is stored in the save.

Matching is server-side: species must match, a quality order accepts that grade
or better, and a mutation order requires that exact mutation. The client mirrors
the same rules only to show "0 из 2" in the barn, never to decide a payout.

## Growing plants

Every species has its own growing silhouette, rather than a shared sprout. Roots
have distinct foliage, vines spread close to the ground, corn grows upright,
strawberries form a rosette, and dragonfruit develops cactus stems. Three visual
stages are derived from the saved planting and maturity times: shoots before 35%,
fuller foliage at 35%, buds or unripe fruit at 70%. Harvest-ready art stays separate.
Plot index adds a small stable lean, so neighboring plants are not exact clones.
No random rewards or mutations are disclosed by growth illustrations.
Production build and manual comparison covered all 54 base species/stage combinations.

## Farm themes and payments

Garden is included. Tropical and lunar themes change the scenery and UI palette,
without changing crop probabilities or harvest timing. Owned themes and the active
choice live in the server save. Promos are normalized, stored by SHA-256 hash,
and redeemed transactionally with per-account, quota and expiry checks.
`TROPIC` and `LUNA` are seeded only with `SRUNAL_DEV_INSECURE_KEYS=1`.
Production codes are provisioned in `farm_theme_codes` with a theme, a hashed code,
maximum uses and optional expiry. No demo codes are installed in production.

Cosmetic purchases use Telegram Stars: 150 XTR tropical, 200 XTR lunar. Enable
`SRUNAL_FARM_STARS_ENABLED=1`, supply `SRUNAL_BOT_TOKEN` and
`SRUNAL_WEBHOOK_SECRET`, then configure the bot's HTTPS `/bot/webhook` with the
same secret token and `message` / `pre_checkout_query` updates. These secrets are
not configured by this project. The local browser offers working promo codes;
payment buttons stay disabled until Stars and the Telegram invoice UI are available.

The server creates a per-account invoice, checks payer identity, currency, price
and order age at precheckout, then grants the theme only after a secret-validated
`successful_payment`. Charge IDs are recorded and repeated webhooks are idempotent.
Client invoice callbacks only refresh data; they cannot grant themes.
See [Telegram's digital goods payment flow](https://core.telegram.org/bots/payments-stars).
Live Stars payments have not been exercised; local tests use a fake Bot API.

## Validation

- `npm run build`: TypeScript strict checks and production bundle.
- `go test ./...`: backend unit checks (PostgreSQL integration tests skip without a test DSN).
- `SRUNAL_TEST_DATABASE_URL=... go test -v ./internal/apps/farm`: creates a temporary
  isolated schema, applies the actual farm migration, and checks authenticated
  HTTP actions, persistence, account isolation, three-hour planting, early harvest
  rejection, client-currency rejection, pack quantities and leaderboard revenue.
  Eight simultaneous sales must credit the player exactly once. The temporary
  schema is removed by test cleanup; live players are untouched.

Remaining limits: no Telegram crop-ready notifications, automated season prizes,
or live multi-user load test. The economy uses an initial balance suitable for
playtesting and will need tuning from actual player behavior.

## First run and visual polish

A brand new player lands in the garden with no idea what to do, so the
walkthrough opens automatically on first visit and never again: the fact is a
local flag, not server state, because it is a UI preference rather than
progress. Five steps, one idea each, skippable from any step. The same component
backs the "Правила игры" button, replacing the old wall of text.

Free beds previously read as broken tiles: their label was a fixed olive that
vanished on the dark worlds (1.34:1 contrast). The label now uses the theme ink,
the furrow lines are faded behind it, and the caption says "Свободная грядка"
instead of a poetic line. Disabled primary buttons dim their fill and keep white
ink; relying on opacity alone left the green fill looking actionable, and the
theme palettes overrode the disabled colour until it was pinned per world.

Forward-pointing actions (sell, open pack, activate code, tutorial next) use a
right chevron. The existing arrow glyph is a left chevron and means back, so
reusing it made primary actions read as navigation.

## Content expansion and illustrated village

Design Read: redesign-preserve of a Russian mobile farm game for short visits.
Keep the six world palettes, server-owned saves, navigation journeys, three-hour
rhythm and accessibility semantics. Evolve the illustrated garden with a village
scene, richer plant volume and legible growth states. Dials: visual variance 6,
motion 4, density 5, asset dependence 6, brand fidelity 10. All art is native SVG
and CSS; no runtime image service, new dependency or external asset request.

Six base crops extend the original catalogue without reordering its first twelve
entries: sweet pepper, pear, lavender, lotus, mango and starfruit. Three new
recipes create flame pepper, honey pear and astral lotus. There are eighteen
base crops, nine hybrids and 216 possible collection entries. The store offers
rarity filters. Each new species has a mature drawing and three growth stages.
Every plant instance has unique SVG paint IDs so gradients cannot collide.

The orchard pack (Сад чудес) costs 1100 coins, contains three seeds and has a
25% base legendary chance. It covers all six new crops and selected older rare
plants, with a rare-or-better guarantee after ten seeds. Existing pack weights
retain their old probabilities with zero weights appended for new species.
Selection only adds legendary weight to species already offered by that pack.

The village illustration adds a cottage, greenhouse, pond, path, fence, shrubs
and a wooden sign. Each world has its own scene palette, with moonlight in lunar
and enchanted worlds. Plots have deeper soil, labels and maturity highlights.
Ambient ripples, clouds and feedback respect reduced-motion preferences.

## Expeditions

One gardener can travel while every planted crop continues growing. All routes
last exactly three hours and display a guaranteed reward before purchase:

| Route | Coins | Seeds | XP |
| --- | ---: | --- | ---: |
| Цветущий луг | 90 | Pepper x2, lavender x1 | 10 |
| Забытый сад | 360 | Pear x2, mango x1 | 20 |
| Звёздные острова | 950 | Lotus x1, starfruit x2 | 35 |

Route, rewards, XP and server start/return times are snapshotted in the player
JSON save when starting. Changing a route's catalogue cannot change an existing
promise. An overdue trip does not expire. Claiming atomically grants its seeds
and XP, clears the job and cannot pay twice. No new SQL migration is required.
Tests cover insufficient funds, unknown routes, early and repeated claims,
independent snapshots and eight simultaneous HTTP claims against an isolated
PostgreSQL schema. Exactly one claim succeeds; live player clocks are untouched.

## Order and tutorial corrections

Orders consume the cheapest matching specimens first. Their quote is the larger
of the listed base reward and 125% of the selected specimens' actual market
value, including the market upgrade. Quotes appear in the response without
changing the saved base reward. Fulfilment recalculates under the player lock,
credits coins, lifetime sales and weekly revenue in the same transaction.
The leaderboard refreshes after fulfilment. Tests cover preserving gold plants,
paying the quoted premium and recording HTTP order revenue in the shared top.
Closing the tutorial by its close button, Escape or backdrop records the same
local preference as its skip and finish buttons.
