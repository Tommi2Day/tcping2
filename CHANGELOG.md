# tcping2

## [1.2.0 - 2026-06-06]
### Added
- `tls` command: validate a TLS connection against the system trust store or a custom CA (PEM file, directory, Java JKS trust store, PKCS12/P12)
- `tls show` subcommand: display certificate details (subject, issuer, SANs, validity dates, signature algorithm, serial) and optionally the full peer chain (`--chain`)
- STARTTLS support for `tls` and `tls show`: `--starttls smtp|imap|pop3|ftp`
- Local certificate file validation via `tls --certfile <path>` (PEM or DER)
- Weak signature algorithm detection (MD2, MD5, SHA-1, DSA-SHA1) with a yellow `WARN` line on both commands
### Changed
- update dependencies
- use Go1.26
- update linter config
- fix linter issues by using constants
- update workflow: consolidate Docker build and push into GoReleaser
- add support for GitHub Container Registry (GHCR)
- increase test coverage
### Fixed
- pass version, commit, and date to Docker build for correct version reporting in containers
- fix ldflags package path in GoReleaser config

## [1.1.6 - 2025-12-26]
- update dependencies
- use Go1.25
- update workflow actions

## [1.1.5 - 2025-03-01]
- add darwin_arm64 build
- update dependencies
- fix linter issue in version

## [1.1.4 - 2024-10-03]
- add arm64 target
- use Go1.23
- use Goreleaser V2 and v6 GitHub Action
- update dependencies

## [1.1.3 - 2024-08-18]
- update dependencies
- fix new linter issues
- rename testfunc to testinit

## [1.1.2 - 2024-05-04]
- move to golang 1.22
- update dependencies
- move Dockerfile to folder docker/image
- fix echo server timeout
- use full path for go mod

## [1.1.0 - 2024-04-28]
- added echo Server and Client and related tests
- update dependencies
- reduce complexity for ping tcp

## [1.0.1 - 2024-04-13]
- Initial release
