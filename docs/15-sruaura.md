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
- The expanded catalog contains 64 relics: 18 exclusive to events, 43 craftable
  E-A items and three ordinary S items obtainable as drops. All original IDs
  remain valid. S, SS, SSS and event items have no recipe, enforced by the API.
- Signature recipes stay accessible for early build experiments. Crafting the
  original relics scales quadratically with rarity. Awakening stages four and
  five require quoted dungeon materials as well as aura.
- System objectives reward progress and rank advancements. Claiming a completed
  objective pays once. Ranking uses dungeon achievement rather than spendable
  currency alone. The UTC Monday weekly board stores each hunter's best
  dungeon/difficulty/errors result and clear count in the same transaction as
  victory. The original arena, tower and grind modes remain accessible.

## Rare events and loot

Each gate entry snapshots a single hidden event roll: false S-rank gate 0.8%,
red gate 1.2%, hidden shrine 1.5%, otherwise ordinary. The event reveals after
room one or two and asks for an explicit enter/leave choice. The public state
omits the pending result; refreshing or retrying cannot reroll it. Event battles
retain the chosen math difficulty. The Silent Warden has three phases, red
gates have two locked encounters with health drain, and the shrine heals its
guardian on mistakes. Declining preserves room materials. Successful events
grant seals and a relic from their separate pools.

Base item chances per ordinary victory: crypt 35%, forge 45%, void 55%. Their
maximum ranks are C, A and S. The void's S chance is 1% conditional on an item
dropping. The tenth empty ordinary victory guarantees an item without raising
its rank. SS requires a void event after 20 clears; SSS requires 80 clears.
Eligible false-gate victories build separate SS and SSS guarantees at five
and twenty victories, reset by the corresponding exclusive drop. Rank S threat
does not promise rank S loot. The server returns odds and current guarantees.

Gate recovery is 20 minutes after victory, two minutes after failure or leaving.
The new boss portrait is `web/public/aura/silent-warden.png`, generated with the
built-in image tool using an original cathedral executioner with an obsidian
mask, fractured gold halo and violet rim lighting. It matches the existing
atlas palette. Dungeon cards preserve square portrait geometry and align the
image to the top, keeping heads visible on mobile. Rank colors match across
icons, item cards and loot dialogs.

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
