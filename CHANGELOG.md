<!-- markdownlint-disable -->
<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Changelog](#changelog)
    - [[v0.5.0] - 2020/08/17](#v050-20200817)
        - [Feature](#feature)
        - [Fix](#fix)
        - [Refactor](#refactor)
    - [[v0.4.0] - 2019/07/10](#v040-20190710)
        - [Added](#added)
        - [Fixed](#fixed)
    - [[v0.3.1] - 2019/12/16](#v031-20191216)
        - [Fixed](#fixed-1)
    - [[v0.3.0] - 2019/11/18](#v030-20191118)
        - [Added](#added-1)
    - [[v0.2.0] - 2019/11/15](#v020-20191115)
        - [Added](#added-2)
        - [Changed](#changed)
    - [[v0.1.2] - 2019/10/14](#v012-20191014)
        - [Fixed](#fixed-2)
    - [[v0.1.1] - 2019/10/12](#v011-20191012)
        - [Fixed](#fixed-3)
    - [[v0.1.0] - 2019/10/11](#v010-20191011)
        - [Added](#added-3)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->
<!-- markdownlint-enable -->

<!-- markdownlint-disable MD013 MD024 -->
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.5.0] - 2020/08/17

### Feature

- Add an `ExcludeModules` option to `Config`.

  This is a list of module names
  or whole paths
  to ignore.
- Add a `PreMajor` option to `Config`.

  When `PreMajor` is true, `gotagger` will not rev the major version to 1,
  even if commits are flagged as breaking changes. This has no effect if the
  major version is 1 or higher.
- `TagRepo` and `ModuleVersions` validate
  that a release commit references only modules that are changed by the commit
  and that the commit references all of the changed modules.
- Add a `ModuleVersions` function that takes a variadic list of module names,
  and returns the versions of those modules,
  or all modules if called with no arguments.
- Add a `Version` function
  that returns the version of the project.

  In a multi-module repository,
  `Version` returns the version of the first module found.
- Add support for tagging any go module via release commits.

  A release commit may contain a `Modules` footer
  that is a comma-separated list of module names for gotagger to tag.

### Fix

- `Gotagger` no longer ignores all non-root go modules when given a relative path.
- Correctly set `CreateTag` option to `true` when `-push` flag is used.
- `gotagger` correctly ignores directories named `testdata`
  and directories that begin with `.` and `_`
  when looking for go modules.

### Refactor

- Rewrite git and conventional commit parsing.

  This is preparing for full go module support.
  The existing commit parsing
  and git repository interactions
  won't scale to solve the problem of tagging sumodules.
  These packages will remain until the v1.0.0 release of `gotagger`.

## [v0.4.0] - 2019/07/10

### Added

- The `gotagger` cli now takes
`-remote`
and `-prefix`
options to set the name
of the remote to push to
and the version prefix,
respectively.

### Fixed

- `gotagger` only considers tags that match the version prefix when determining
  the base version.

## [v0.3.1] - 2019/12/16

### Fixed

- `gotagger` no longer reports all git command failures as "not a git repository".

## [v0.3.0] - 2019/11/18

### Added

- The base package now exposes a `Config` struct and a `TagRepo` function that
  preforms the basic operations of `gotagger`.

## [v0.2.0] - 2019/11/15

### Added

- Add `-push` and `-release` flags to control when `gotagger` tags a release commit
  and pushes the commit.
- Source options from `GOTAGGER_`-prefixed environment variables.

### Changed

- When tagging a release commit, increment the patch version if there are no
  feat or fix commits since the last release.

## [v0.1.2] - 2019/10/14

### Fixed

- Use `--merged` argument to `git tag` so that we only generate tags that point to
  parents of HEAD.

## [v0.1.1] - 2019/10/12

### Fixed

- Always create annotated tags, otherwise we can't find our own tags.
- Call `git log` with the `--decorate=full` option, so that tags are properly prefixed
  with `refs/tags/`
- Remove unnecessary quotes from `git tag` format. These were being included in the
  formatted string.
- Address a bug in the cli where we tried to do a release when HEAD is already tagged.

## [v0.1.0] - 2019/10/11

### Added

- git package for interacting with git repository
- marker package for parsing commmit markers
- basic cli capability: printing the new version and tagging a repo

[Unreleased]: https://github.com/sassoftware/gotagger/compare/v0.5.0...master
[v0.5.0]: https://github.com/sassoftware/gotagger/compare/v0.4.0...v0.5.0
[v0.4.0]: https://github.com/sassoftware/gotagger/compare/v0.3.1...v0.4.0
[v0.3.1]: https://github.com/sassoftware/gotagger/compare/v0.3.0...v0.3.1
[v0.3.0]: https://github.com/sassoftware/gotagger/compare/v0.2.0...v0.3.0
[v0.2.0]: https://github.com/sassoftware/gotagger/compare/v0.1.2...v0.2.0
[v0.1.2]: https://github.com/sassoftware/gotagger/compare/v0.1.1...v0.1.2
[v0.1.1]: https://github.com/sassoftware/gotagger/compare/v0.1.0...v0.1.1
[v0.1.0]: https://github.com/sassoftware/gotagger/compare/e3ef062...v0.1.0