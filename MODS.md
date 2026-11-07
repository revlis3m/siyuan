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
- Chart database layout (Notion-like): vertical/horizontal bar, line, donut and number charts of the view's filtered
  entries, grouped by an option, checkbox, date (day/ISO week/month/quarter/year), relation, text or number field,
  measured by count or sum/average/min/max of a number, rollup or template field, optional stacking by a
  select/status/checkbox field, the page name (one part per page) or a relation (one part per related page; an entry
  linked to several pages counts in each part, and once in the group total). The kernel aggregates in one pass and returns only the summary (plus up to 10 sample
  entries per data point for the click menu); ECharts draws it. Kernel: av/layout_chart.go, model/attribute_view_chart.go,
  apicontract/av_chart.go; frontend: protyle/render/av/chart/ and assets/scss/business/_av-chart.scss.
  Sorting, grouping, conditional colours and paging do not apply to charts.
  Running total (`cumulative`): bar, horizontal bar and line charts of a count or sum show each group's value as the
  total up to that group (per series when stacked; the empty group keeps its own value), e.g. a balance-over-time line.
