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
- Six species, three qualities, four mutation states (including ordinary).
- Care is selected at planting: water (free, better quality), fertilizer (12 coins,
  one extra plant), or lamp (20 coins, five extra percentage points of mutation).
- Water, soil and lamp upgrades have five levels. Cost: `100 * (level + 1)^2`.
  Their effects are snapshotted at planting, so later upgrades cannot reroll crops.
- Plants sell as a batch. Discoveries persist after sale. Collection is 6 species
  times 4 mutation states, 24 entries. Quality affects value, not album completion.
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
