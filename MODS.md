# revlis3m mods

Branch `mods` = the SiYuan release in `docker/Dockerfile.mods` (BASE_IMAGE) + our changes.
Every push builds `ghcr.io/revlis3m/siyuan:<release>-mods.<n>` (GitHub Actions, workflow "Mods image").

Updating to a new SiYuan release: rebase `mods` onto the new release tag, bump BASE_IMAGE, push.

Changes on top of the release:
- (none yet: pipeline only)
