# mods-tests

Checks of the revlis3m mods on a real, throwaway SiYuan: the kernel API (`api.mjs`) and the editor in a
browser (`ui.mjs`). They complement the unit tests in `kernel/` and `app/`: these start the built server and
editor and use them like a user or a script would.

    ./run.sh          # download the current "mods-preview" build, start it, run everything

| File | What it checks |
|---|---|
| `api.mjs` | Status property (default options and groups, values), Timeline / Chart / Form layouts render, notebook PIN lock (set, lock, refuse without PIN, wrong PIN, unlock, remove) |
| `ui.mjs` | Logs in, opens a page with a database, sees rows and Status chips, switches to Chart and sees it drawn, fails on any page error. Screenshots in `shots/` |
| `lib.mjs` | API helpers shared by both (create a notebook, a database page, add rows) |

Rule for new mods: every mod adds its checks here before its image is given out.

The test SiYuan uses its own empty workspace under `$TMPDIR/mods-tests`; it never touches real notes.
