# revlis3m mods

Branch `mods` = the SiYuan release in `docker/Dockerfile.mods` (BASE_IMAGE) + our changes.
Every push builds `ghcr.io/revlis3m/siyuan:<release>-mods.<n>` (GitHub Actions, workflow "Mods image").

Updating to a new SiYuan release: rebase `mods` onto the new release tag, bump BASE_IMAGE, push.

Changes on top of the release:
- Status property type (Notion-like): options in To-do / In progress / Complete groups, "is complete"
  filters, sort and grouping by group, "Percent / Count complete" calculations and rollups,
  select <-> status conversion, `.<Field>_complete` in templates. Kernel: av/status.go and the
  places handling select; frontend: protyle/render/av/status.ts and the select/col/filter/calc UI.
- Timeline database layout (Notion-like): start/end date fields (or the start date's own end date), day/week/month/quarter
  zoom, colour by a select/status field or conditional colours, unscheduled list, drag to move/resize/create, today line.
  Kernel: av/layout_timeline.go, model/attribute_view_timeline.go, apicontract/av_timeline.go; frontend:
  protyle/render/av/timeline/ and assets/scss/business/_av-timeline.scss. Grouping (lanes) is not applied.
