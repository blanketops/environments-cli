## What

<!-- One or two sentences. What does this PR change? -->

## Why

<!-- The intent. Link the issue if one exists: Closes #123 -->

## Area

<!-- Mark all that apply -->

* [ ] `bops-env install` / `uninstall` (operator manifests)
* [ ] `bops-env dependencies` (platform stack)
* [ ] `bops-env cluster` (Kind lifecycle)
* [ ] `bops-env self` (self-update)
* [ ] Embedded manifests (`dependencies/`)
* [ ] CI / release tooling / docs

## Compatibility

* [ ] No breaking change to CLI flags/commands
* [ ] Breaking change — old command/flag removed or renamed (document the migration below)

<!-- If breaking: what breaks, and what must users do? -->

## Checklist

* [ ] `mage vet` and `mage build` pass locally
* [ ] `mage test` passes
* [ ] Tested against a real (or Kind) cluster, not just unit tests, if touching install/dependencies logic
* [ ] `mage static` still builds if touching embedded assets or ldflags
* [ ] README's command list / usage examples updated if flags or commands changed
* [ ] Commit messages follow Conventional Commits

## Notes for reviewer

<!-- Anything non-obvious: design trade-offs, deferred follow-ups, areas needing close attention -->
