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
- Twelve species, three qualities, four mutation states (including ordinary).
- Care is selected at planting: water (free, better quality), fertilizer (12 coins,
  one extra plant), or lamp (20 coins, five extra percentage points of mutation).
- Water, soil and lamp upgrades have five levels. Cost: `100 * (level + 1)^2`.
  Their effects are snapshotted at planting, so later upgrades cannot reroll crops.
- Land adds one plot per level (four levels, base cost 300). Market adds 5% sale
  revenue per level (five levels, base cost 200). Selection adds one percentage
  point of legendary seed probability per level (five levels, base cost 250).
  Upgrade cost is `base * (level + 1)^2`; server metadata supplies actual levels,
  caps and prices. Personalized weights are returned by the same logic used to roll.
- Plants sell as a batch. Discoveries persist after sale. Collection is 12 species
  times 4 mutation states, 48 entries. Quality affects value, not album completion.
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

## Growing plants

Every species has its own growing silhouette, rather than a shared sprout. Roots
have distinct foliage, vines spread close to the ground, corn grows upright,
strawberries form a rosette, and dragonfruit develops cactus stems. Three visual
stages are derived from the saved planting and maturity times: shoots before 35%,
fuller foliage at 35%, buds or unripe fruit at 70%. Harvest-ready art stays separate.
Plot index adds a small stable lean, so neighboring plants are not exact clones.
No random rewards or mutations are disclosed by growth illustrations.
Production build and manual comparison covered all 36 species/stage combinations.

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
