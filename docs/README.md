# docs

Engineering docs for Voyagami. Each one starts as a stub and is filled in by the ticket that builds the thing it describes. Update the matching doc in the same PR whenever behaviour or data shape changes.

| Doc                                  | What it covers                                                 | Filled in by                                  |
| ------------------------------------ | -------------------------------------------------------------- | --------------------------------------------- |
| [data-model.md](data-model.md)       | Firestore collections, fields, money and locale rules          | VOY-10 and each feature ticket that adds data |
| [sync.md](sync.md)                   | Local-first storage, offline writes, conflicts and sync status | VOY-62, VOY-63, VOY-64                        |
| [security.md](security.md)           | Auth, secrets, access rules, privacy in logs                   | auth tickets in D0, hardened in P1            |
| [accessibility.md](accessibility.md) | Hit areas, labels, text scaling, motion, screen reader checks  | VOY-12 (UI kit), audited in P1                |
| [github-bot.md](github-bot.md)       | Bot identity that opens Claude's PRs, branch protection        | VOY-212                                       |

Product rules that must never be broken live in [CLAUDE.md](../CLAUDE.md). The design specs (Data Map, Tech Stack, Locale Spec, Plans Spec, Recs Spec, Offline Sync) are in [design/BOARD_INDEX.md](../design/BOARD_INDEX.md). If a doc here and a spec board disagree, the spec board wins until the doc is updated.
