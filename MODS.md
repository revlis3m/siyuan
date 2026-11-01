# revlis3m mods

Branch `mods` = the SiYuan release in `docker/Dockerfile.mods` (BASE_IMAGE) + our changes.
Every push builds `ghcr.io/revlis3m/siyuan:<release>-mods.<n>` (GitHub Actions, workflow "Mods image").

Updating to a new SiYuan release: rebase `mods` onto the new release tag, bump BASE_IMAGE, push.

Changes on top of the release:
- Status property type (Notion-like): options in To-do / In progress / Complete groups, "is complete"
  filters, sort and grouping by group, "Percent / Count complete" calculations and rollups,
  select <-> status conversion, `.<Field>_complete` in templates. Kernel: av/status.go and the
  places handling select; frontend: protyle/render/av/status.ts and the select/col/filter/calc UI.
