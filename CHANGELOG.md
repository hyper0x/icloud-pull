# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- Replace real vault name with generic example (`MyVault`) in README to avoid
  leaking personal information in the public repository.

### Changed

- Remove `-v` short flag for `--version`; only `--version` and `version`
  subcommand are supported. Short flags are intentionally omitted to avoid
  conflicts with wrapped commands.
- Add custom `Usage()` for `status` and `download` subcommands. Previously,
  `status --help` and `download --help` printed bare Go flag output without
  usage context. Now they show a description, flag list, and example.
- Remove non-existent Homebrew install instructions from README. The `brew
  install hyper0x/tap/icloud-pull` line referenced a tap that does not exist
  yet.
