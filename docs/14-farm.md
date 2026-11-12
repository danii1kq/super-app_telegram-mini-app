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

- Four plots, each crop takes exactly three real hours. Mature crops never rot.
- A new account receives 250 coins, seven seeds and two mature welcome plants.
- Twelve species, three qualities, four mutation states (including ordinary).
- Care is selected at planting: water (free, better quality), fertilizer (12 coins,
  one extra plant), or lamp (20 coins, five extra percentage points of mutation).
- Water, soil and lamp upgrades have five levels. Cost: `100 * (level + 1)^2`.
  Their effects are snapshotted at planting, so later upgrades cannot reroll crops.
- Plants sell as a batch. Discoveries persist after sale. Collection is 12 species
  times 4 mutation states, 48 entries. Quality affects value, not album completion.
- Packs each contain three seeds. Per-seed probabilities are returned by the API.
  Every tenth seed in a pack-specific streak is at least rare. A natural rare or
  legendary seed resets that streak. No real money, payment or cash-out.
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
There is no background music or external audio dependency. Effects last under
one second. Seedlings sway, planting pops into place, harvest highlights the barn,
sales animate the wallet, and pack drops reveal in sequence. Reduced-motion
preferences disable all game animations. `node --test src/apps/farm/sounds.test.ts`
checks audio duration, gain, node cleanup and suspended-context behavior.

Pack purchases now enter a dedicated reveal scene. Each seed is revealed on tap,
with a brief charge, a burst, rotating light rays and rarity-specific colors.
Legendary seeds have gold stars and a longer sound cue. A summary shows all three
seeds; skipping or closing the scene cannot lose rewards, because the server
credits them in the purchase transaction before the animation begins. Revealing
or replaying an animation never rolls or buys again. Reduced motion disables
visual effects. New species: radish, cucumber, corn, sunflower, watermelon and
legendary moonflower. Pack weights and the guarantee use species rarity, so adding
a new common seed cannot accidentally reset the rare-seed guarantee.

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
