# revlis3m mods: changelog

One entry per published image `ghcr.io/revlis3m/siyuan:<release>-mods.<n>`. Newest first.
Numbers that are missing (2, 3, 8, 10, 17) were builds that failed or were cancelled: no image exists for them.
What each mod does in detail is in [MODS.md](MODS.md).

## 3.8.9

### mods.20 — 2026-11-12
- Build: the editor's database and mods tests now run before an image is built.
- Build: the image is started and checked (API and browser checks in `mods-tests/`) before it is published.
- Form: the list of field types a form can ask is written the way the editor's own rules require. No change in behaviour.

### mods.19 — 2026-11-07
- Form: typing in a form field no longer loses focus after each character.

### mods.18 — 2026-11-07
- Form: a new form view shows every field that can be answered.
- Form: the "Added" message stays visible when the form redraws.

### mods.16 — 2026-11-07
- New: Form view for databases. The visible fields are the questions; Submit adds an item. Per field: required, default value, "show only when".

### mods.15 — 2026-11-07
- New: notebook PIN lock (notebook menu → "Lock with PIN…"), with auto-lock after idle minutes, "Lock now", and lock on restart.

### mods.14 — 2026-11-07
- Fix: creating a page through the API works in encrypted notebooks.

### mods.13 — 2026-11-07
- Mask amounts: database toolbars stay visible while amounts are masked.

### mods.12 — 2026-11-07
- New: mask amounts in the notebooks you mark (notebook menu → "Mask amounts in this notebook"), with the eye button to reveal.

### mods.11 — 2026-11-07
- Chart: running total option (for example a balance over time).

### mods.9 — 2026-11-07
- Chart: stack by the page name or by a relation.

### mods.7 — 2026-11-02
- Chart: group by the page name (one bar per page).

### mods.6 — 2026-11-02
- New: Chart view for databases (bar, horizontal bar, line, donut, number).

### mods.5 — 2026-11-02
- New: Timeline view for databases.

### mods.4 — 2026-11-01
- New: Status property type (To-do / In progress / Complete groups, "is complete" filters, percent complete).

### mods.1 — 2026-11-01
- Build pipeline only: same as the official SiYuan 3.8.9.
