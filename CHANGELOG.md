## [0.0.9] - 2026-07-13

### 🚀 Features

- Describe each command in the usage banner
- Replace per-resource apply logs with a live progress indicator
- Rename bops to bops-env, implement self-install, fix install order

### 🐛 Bug Fixes

- Repair broken build, skip crossplane from auto-install, restore dependencies commands
- Restore the original explicit install/uninstall pipeline
- Restore the original explicit install/uninstall pipeline (#45)

### 💼 Other

- Skip calico dependency during manifest discovery
- Fix manifest apply order — shipwright before cluster_strategies
- Install Crossplane core before applying its Provider manifest
- Restore dependencies install/uninstall, stub top-level install/uninstall
- Restore dependencies install/uninstall, stub top-level install/uninstall (#35)
- V0.0.9 (#20)

### 📚 Documentation

- Refresh README, add provenance/signing section, rename to bops

### ⚙️ Miscellaneous Tasks

- Sync develop with main after release/v0.0.8
- *(release)* Update changelog for v0.0.8
- Fill in truncated Apache license header in cmd/all.go
- Add CodeQL and govulncheck security scanning
- Fix govulncheck fail-on-findings and upload SARIF to Security tab
- Dedupe SARIF rule tags before uploading
- Comment out dependencies install/uninstall subcommand
- Run vet/test in CI, sign the dynamic build, symmetric job summaries
- Split CodeQL into its own Advanced workflow, stop blocking on govulncheck
- Add step summary to CodeQL Advanced workflow
- Add Dependabot config and a real security policy
- Add Dependabot config and a real security policy (#36)
- Lowercase org references in SECURITY.md
- *(ci)* Bump actions/checkout from 4 to 7
- *(ci)* Bump actions/checkout from 4 to 7 (#44)
- *(ci)* Bump actions/upload-artifact from 4 to 7
- *(ci)* Bump actions/upload-artifact from 4 to 7 (#42)
- *(ci)* Bump actions/setup-go from 5 to 6
- *(ci)* Bump softprops/action-gh-release from 2 to 3
- *(ci)* Bump softprops/action-gh-release from 2 to 3 (#40)
- *(ci)* Bump actions/setup-go from 5 to 6 (#39)
## [0.0.8] - 2026-07-05

### ⚙️ Miscellaneous Tasks

- Sync develop with main after release/V0.0.7
## [V0.0.7] - 2026-07-05

### 💼 Other

- *(sanitize)* Update
- *(docs)* Add license
- *(docs)* Sanitize usage, shorten
- *(ci)* Update ci, resolve conflicts, sign and attest
- *(ci)* Update ci, resolve conflicts, sign and attest

### ⚙️ Miscellaneous Tasks

- Sync develop with main after release/v0.0.6
- *(release)* Update changelog for v0.0.6
- Require Cosign signing + attestation on CI builds, document workflows
## [0.0.6] - 2026-04-27

### 💼 Other

- Merge release/v0.0.6 into main

### ⚙️ Miscellaneous Tasks

- *(buildTool)* Fix and update magefile
## [0.0.5] - 2026-04-26

### 💼 Other

- *(deps)* Crossplane eso setup force
- *(ci)* Add workflows for release

### ⚙️ Miscellaneous Tasks

- *(release)* Update changelog for v0.0.4
## [0.0.4] - 2026-03-17

### ⚙️ Miscellaneous Tasks

- *(release)* Update changelog for v0.0.3
## [0.0.2] - 2026-03-17

### 🚀 Features

- *(cli)* Maintenance, switch off ci
## [0.0.1] - 2026-03-15
