# SruAura: охотник Системы

The game remains at `/apps/game/aura`. This upgrade preserves existing heroes,
currency, class choices and owned relics. Migration 000022 extends the existing
player table rather than replacing its records. The unrelated `demo/` folder is
not part of this implementation.

## Approved direction

A dark manhwa RPG in which reading mathematical runes powers attacks. Short
dungeon runs connect combat, route choices, a boss chest and permanent equipment.
Three regions are the flooded crypt, ash forge and void rift. Each region has
its own boss illustration, atmosphere and crafting material. Hero artwork is
visible on the home screen and combat scene.

Visual mode: overhaul inside SruAura, preserving the surrounding app and routes.
Variance 7, motion 6, density 6, asset dependence 8, shell fidelity 10. Graphite
and navy surfaces, violet System magic, gold rewards and red danger cues. The
existing typeface stays local. Large answer targets serve phone use; desktop
uses the extra width for the hero, objectives and equipment. Reduced motion and
sound controls apply to feedback. Artwork details and generation prompt are in
[aura-art.md](aura-art.md).

## Gameplay contracts

- Math difficulty is chosen independently of hero level.
- New gate runs have two encounters followed by a boss. Between encounters the
  player chooses an ordinary fight, elite, shrine or cursed chest, or retreats.
- Correct answers attack; errors and late answers expose the hero to damage.
  Server deadlines and question IDs determine what is accepted.
- Three successful answers charge a class ability. Each of the five existing
  classes has an actual effect rather than an informational description.
- Defeat does not award the final boss relic. Room materials already earned
  remain available; retreat has an explicit result.
- Strength, agility, intelligence, vitality and perception affect the run.
  Growing equipment and stats are snapshotted for the active run.
- Existing relics retain identity, level and equipment slot. Duplicate and paid
  upgrades increase actual effects. Signature relics add combo echoes, healing,
  additional time and finishing damage. Recipes consume named dungeon materials
  and aura. Prices and requirements come from the server.
- Signature recipes stay accessible for early build experiments. Crafting the
  original relics scales quadratically with rarity. Awakening stages four and
  five require quoted dungeon materials as well as aura.
- System objectives reward progress and rank advancements. Claiming a completed
  objective pays once. Ranking uses dungeon achievement rather than spendable
  currency alone. The UTC Monday weekly board stores each hunter's best
  dungeon/difficulty/errors result and clear count in the same transaction as
  victory. The original arena, tower and grind modes remain accessible.

## Persistence and verification

Currency, loot, equipment changes, completed runs and request records must
commit together under the player row lock. A retried or stale answer must not
advance a newer question. Active answer keys remain server-owned.

`http_integration_test.go` creates a temporary PostgreSQL schema, applies the
actual Aura migrations and calls authenticated HTTP handlers. It checks a
concurrent answer race, cross-account isolation, a complete dungeon victory,
defeat without boss loot, and preserving and upgrading old equipment. Test
resources exist only inside that isolated schema.

The suite also verifies charged abilities, server deadlines, a quest paid once
and simultaneous crafting requests consuming exactly one recipe. A before/after
local migration check matched hero counts, total aura, XP and owned relic
levels. Those checks do not model long-term pacing.

Run from the repository root:

```powershell
docker compose -p srunal-local -f compose.local.yml exec -T -e SRUNAL_TEST_DATABASE_URL=postgres://srunal:local-preview-only@db:5432/srunal?sslmode=disable api go test ./...
```

Frontend checks run inside `web`:

```powershell
npm run build
node --test src/apps/aura/display.test.ts
```

Restart the local API after backend changes to apply migrations and compile the
new server. Guilds, cooperative raids and branching class evolutions are future
content, outside this first approved release. Rank exams are achievement
contracts in this release, and abilities charge in combat without an upgrade
tree. Numerical tuning still requires
playtesting beyond automated correctness checks.
