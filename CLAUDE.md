# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Mirra (镜衣) is an NSGo app: one Go codebase for macOS, iOS, Android, Windows and web that combines a face, outfit and pose from three public galleries (plus optional background) and asks an image model to draw the result. The marketing site is a separate TanStack app. Read [AGENTS.md](AGENTS.md) for project principles and [ARCHITECTURE.md](ARCHITECTURE.md) for the full code map and boundaries before changing code.

## Layout

- `apps/mirra/` — the only Go module (`go.mod`, `nsgo.yaml`, theme, locales, backgrounds). Run every Go and `nsgo` command from here.
- `apps/website/` — TanStack Start + React + Tailwind + Bun. Install and run scripts from here. Never use Astro.
- Docs live at the repository root only (`README.md` / `README.zh-CN.md` stay equivalent; `CHANGELOG.md`, `NOTICE.md`, `ARCHITECTURE.md`).

Requires Go 1.27.4+ (the module's go directive asks for it). Website needs Bun (CI uses 1.3.4).

## Commands

### App (`apps/mirra`)

```sh
go tool nsgo dev                          # run this machine (hot reload; -target ios_simulator|android|web, -lang zh-Hans)
go tool nsgo build                        # packages for nsgo.yaml targets into dist/
go tool nsgo doctor                       # which targets this machine can build/run

go tool nsgo test .                       # headless tests (CI default; no network, no display)
go tool nsgo test -run TestName .         # one test or pattern
go tool nsgo test -target macos .         # same tests in a real macOS window
go tool nsgo check -target macos,ios,android,windows,web   # vet every target (CI)

gofmt -l .                                # must be empty (CI fails otherwise)
go mod tidy && git diff --exit-code go.mod go.sum
go run ./tools/appicon                    # redraw assets/Icon.png
go tool nsgo i18n                         # regenerate locales/catalog_gen.go from locales/*.yaml
```

Tests are offline: `fixtures_test.go` fakes the gallery pages and image APIs. `nsgotest` apps call `nsgotest.Main` from `TestMain`. Under `-target`, some gesture tests skip (NSGo limitations; link nsgo issues from workarounds).

### Website (`apps/website`)

```sh
bun install
bun run dev        # http://localhost:5175/
bun run typecheck  # tsgo --noEmit
bun run build      # static prerender into dist/client
bun run og         # retake public/og-<lang>.png (starts its own server on 5176)
```

CI also runs `shellcheck .github/scripts/*.sh` and `.github/scripts/changelog-test.sh`.

### Verification bar

From `apps/mirra`: `go tool nsgo test .`, `go tool nsgo test -target macos .`, `go tool nsgo check -target macos,ios,android,windows,web`. From `apps/website`: `bun run typecheck` and `bun run build`, then look at pages in a browser. Green headless runs do not prove native UI—review UI changes in a real window.

## Architecture (big picture)

**Single Go module, one `studio` state.** `app.go` owns `App` (injected `Client`, `Sites`, `DataDir`, `Pictures` for tests) and `studio`: signals for the chosen look, shot, section, step, service, favorites. Phone (`phone.go`) and desktop (`desktop.go`) are two layouts of that same state; layout switches on window width (`wideWidth = 900`), not device type. Shared library UI is `library.go`; settings/result/Works pages are `pages.go`.

**Look and generate.** Elements are `KindFace | KindCostume | KindPose | KindBackground` (background optional, last). Only pose is required. `generate.go` builds `Request` and the prompt (images ordered pose → outfit → face → background; background last so numbering stays stable). Providers: OpenAI-compatible multipart in `generate.go`, others in `apis.go` (Gemini, Ark, xAI, OpenRouter, Model Studio, Azure…). Presets, per-provider settings and key migration: `providers.go`. Connection test / model list: `service.go`. All-angle sheets: `sheet.go`.

**Galleries are not in the repo.** `gallery.go` parses live indexes from `DefaultSites`; full images download only on Generate. Never commit gallery images. Never put a person’s photo in a request (pose/outfit terms); wardrobe uploads are clothes only. Backgrounds are shipped CC0 scenes in `backgrounds/` (see `NOTICE.md`).

**Persistence boundaries.** API keys only in `cx.Secrets()` (web keeps them in memory and says so). Pictures only through the `pictures` interface (`store.go`)—files natively, memory in the browser—not raw `os`. Works are numbered files plus JSON sidecars (`works.go`). Favorites/outfit wardrobe in storage (`wardrobe.go`).

**i18n.** Every user-visible string is a key in `apps/mirra/locales/*.yaml` (English source, 11 languages). `catalog_gen.go` is generated—do not hand-edit; run `go tool nsgo i18n`. In code: `t := locales.For(cx)` then `t.SomeKey()`. Website copy is separate: `apps/website/src/i18n/copy.ts` (zh `/`, en `/en/`, ja `/ja/`).

**Theme.** Always dark amber from `theme.yaml` (`themeYAML` embedded). Take colors from the theme, not hard-coded hex.

**NSGo.** Use `github.com/nsgo-dev/nsgo/ui` components. Framework bugs go to [nsgo-dev/nsgo](https://github.com/nsgo-dev/nsgo/issues) with the issue linked from any local workaround comment.

## Releases and docs

- Conventional Commits drive releases: `feat` → minor, `fix`/`perf` → patch; `docs`, `chore`, `style`, and scopes `website`/`deps`/`ci` release nothing (`.github/scripts/next-version.sh`). While major is 0, breaking bumps minor.
- User-visible changes go under `## Unreleased` in `CHANGELOG.md` as they land; the Release workflow turns that into the version section (or builds notes from commits).
- Update a doc only when a fact its reader needs changes; repair what you touch rather than appending. Keep `README.md` and `README.zh-CN.md` equivalent; everything else, including comments, English.
