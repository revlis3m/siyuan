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
- Mask amounts: notebook menu → "Mask amounts in this notebook". Editors of a marked notebook paint numbers over
  (CSS custom highlight; documents are not modified), dates and times excepted, and charts show "•••" for values.
  Every marked notebook starts masked when the app loads; the eye button in the editor toolbar reveals it until the
  next load. The block being edited stays visible. Marked notebooks are kept in local storage `local-mask-notebooks`.
  Code: app/src/protyle/util/maskAmounts.ts, maskAmountsText.ts.
- Fix: creating a document with markdown no longer adds an empty `tags` attribute, which encrypted notebooks refused
  (`/api/filetree/createDocWithMd` failed with error 390 in encrypted notebooks).
- Notebook PIN lock: notebook menu → "Lock with PIN…". A locked notebook is a closed notebook that the kernel only
  opens after `/api/notebook/unlockNotebookPIN` (model.Mount refuses it otherwise), so it is out of the tree, search,
  backlinks and embeds. It locks again after N idle minutes (default 15, `setNotebookPINMinutes`; activity is the
  existing `touchEncryptedNotebooks` ping), with "Lock now", and on restart. PINs: PBKDF2-SHA256 in
  `conf/mods-notebook-lock.json` (not synced); wrong PINs are slowed down. Privacy lock only, files are not encrypted.
  Code: kernel/model/mods_notebook_lock.go, kernel/api/mods_notebook_lock.go, app/src/util/notebookPIN.ts.
- Workflow "Mods API generate": push a commit to branch `mods-apigen` to get the generated schema/declarations as a
  gzip+base64 patch in the job log (for environments that cannot run the Go generator).
