# Master PRD — Immersive Cross-Platform E-Reader (TUI-First)

## 1) Summary

An immersive e-reader application focused primarily on a **beautiful, full-screen Terminal UI (TUI)** experience. The app is **local-first** in early versions and later expands into a cross-platform ecosystem (mobile/web via React Native) plus community sharing and an SSH “instant access” mode similar to `ssh terminal.shop`.

**Core promise:** distraction-free, book-like reading in the terminal with a premium aesthetic and strong reading ergonomics.

---

## 2) Goals and non-goals

### Goals

- Deliver an **immersive reading experience** in terminal (two-page spread, zen mode).
- Provide a reliable **local library** with import/preprocess, search, and resume.
- Support **EPUB + PDF** early, with a hybrid PDF experience (text mode + layout mode).
- Track reading progress precisely and resume smoothly even with live reflow.
- Establish foundations for later:

  - accounts + syncing + community sharing
  - mobile/web clients
  - public SSH entrypoint that runs the app without local install

### Non-goals (for early versions)

- No backend, no authentication, no community features in v1/v2.
- No release/distribution pipeline required yet (CI only).
- No full fidelity PDF images/graphics guarantee in the local Go-native renderer (best-effort).

---

## 3) Target users and use cases

### Primary users (initial)

- Developers/power users who read on terminals (local or via SSH into their own machines)
- Readers who want a calm, immersive interface with keyboard navigation

### Key use cases

- Import an EPUB/PDF into a local library
- Search library to find a book
- Open and read in a two-page spread
- Toggle **Zen mode** for maximum immersion
- Search within a book
- Resume reading automatically from last position
- (Later) Highlight/bookmark passages and navigate TOC
- (Later) Access via `ssh <service>` without installing anything locally

---

## 4) Product principles

1. **Immersion-first:** reading view is the hero; library is minimal and fast.
2. **Keyboard-native:** efficient, learnable shortcuts; arrow keys primary; vim keys supported.
3. **Calm aesthetics:** inspired by terminal.shop—minimal UI chrome, strong hierarchy, generous spacing.
4. **Local-first reliability:** works offline; progress never lost; fast startup resume.
5. **Honest format support:** EPUB+PDF supported early; PDF is hybrid with clear modes.

---

## 5) Platforms and phased approach

### Platform plan

- **TUI app (Go):** primary platform now (v1, v2, and beyond)
- **Backend (Go):** introduced later for accounts/sync/community
- **Mobile/Web:** React Native + React Native Web later

### Versions

- **v1 (Local TUI MVP):** library + add/import + read; EPUB+PDF hybrid; search; resume.
- **v2 (Local TUI Enhancements):** TOC overlay, settings (layout+theme), highlights/bookmarks.
- **Post-v2 (Ecosystem):** SSH entrypoint, backend, auth, community sharing, cross-platform clients.

---

## 6) Core features (overall product)

### 6.1 Library

- Local library catalog with metadata (title/author), progress, last opened.
- Search from library (`/` to search prompt).
- Add/import books (path input + file browser).
- Hybrid storage model with **managed copy default ON** (copies into app storage by default, while retaining reference to original path as available).
- Removal from library; optional “delete from disk” with a **scary confirmation**.

### 6.2 Reading experience

- Full-screen reading view designed for immersion.
- **Two-page spread** when terminal width allows; fallback to single page otherwise.
- **Zen mode**: hides all controls; only pages + per-page page numbers visible.
- **Live reflow on terminal resize**, anchored to the same text position.
- Progress tracked and persisted; **startup auto-resume** most recent unfinished book.
- “Finished” is **manual** (user marks finished).

### 6.3 Format support

- **EPUB:** native parsing, text extraction, pagination.
- **PDF:** hybrid:

  - **Text mode:** extracted text, paginated like EPUB (supports search well).
  - **Layout mode:** page-based layout rendering (Go-native best-effort, fidelity-oriented), supports two-page spread when wide enough.
  - App remembers separate positions for text vs layout mode.

### 6.4 Search

- Library search: token/ranking based (“good enough”).
- In-book search: on-demand scan of preprocessed text; navigation with next/previous results.

### 6.5 Settings (v2+)

- Global reading layout settings (e.g., margins, width, spacing) and theme options.
- No true font family selection in terminal; settings reflect terminal-realistic controls.

### 6.6 TOC (v2+)

- TOC available as a **reader overlay drawer** (EPUB nav support; PDF if available later).

### 6.7 Highlights/bookmarks (v2+)

- Highlighting in:

  - EPUB (text-flow selection)
  - PDF text mode (text-flow selection)
  - PDF layout mode (region selection on page grid)

- Highlights list and navigation (v2).

---

## 7) User experience specification (global)

### Visual style

- Full-screen, centered content blocks and consistent spacing.
- Minimal UI chrome; small footers/hints outside zen.
- Overlays are lightweight and non-disruptive.

### Input model

- **Arrow keys** are primary navigation in reader; vim keys also supported.
- Zen mode still allows core actions via **minimal overlays** (search/go-to/highlight/TOC as applicable).
- Key concepts:

  - `/` search
  - `g` go-to page, `G` go-to percent
  - `z` toggle zen
  - `m` toggle PDF mode (text/layout)
  - `?` help
  - v2: `t` TOC overlay, `v` highlight mode, `s` settings

### Startup behavior

- If there is an unfinished book: open it to last position automatically.
- Otherwise open library list.

---

## 8) Data and persistence (overall)

- **SQLite (pure Go)** as the primary persistent store:

  - books, metadata, progress states, (later) highlights/bookmarks/settings

- **TOML config** for user configuration (paths, behavior toggles, etc.)
- Managed library storage directory for copied book files + caches (preprocessed text, layout render caches).

---

## 9) Technical architecture (product-level constraints)

- **Clean Architecture** principles:

  - domain entities and use-cases decoupled from UI, DB, and external libs

- TUI stack: **Charm toolkit** (Bubble Tea, Lip Gloss, Bubbles, Huh etc.)
- PDF must be **Go-native** (no native dependencies).
- Monorepo: **Turborepo**
- CI: **GitHub Actions** running tests + lint (**golangci-lint**); no release pipeline required yet.

---

## 10) Metrics of success (master-level)

Early indicators (local TUI):

- Import success rate for common EPUB/PDF files
- Time-to-first-page after import (perceived performance)
- Page turn responsiveness (no stutter)
- Resume reliability (returns to correct position after restart/resize)
- Low crash rate; clear errors for unsupported/DRM content

Later indicators (ecosystem):

- Successful SSH sessions and retention
- Account conversion and sync reliability
- Community sharing engagement (uploads, downloads, discussions)

---

## 11) Risks and mitigations

### PDF layout fidelity without native deps

- Risk: layout mode can’t fully match PDF visuals (images/graphics).
- Mitigation: define layout mode as “best-effort text layout fidelity”; keep text mode strong and searchable; communicate clearly.

### Live reflow + stable progress

- Risk: page numbers shift on resize/settings changes.
- Mitigation: store progress as stable locators (offset-based) and re-anchor on reflow.

### Scope creep (formats, features)

- Mitigation: v1 locked to EPUB+PDF; TOC/highlights in v2; SSH/backend/community post-v2.

---

## 12) Roadmap (high-level)

### v1 — Local TUI MVP

- Library + Add/Import + Reader
- EPUB + PDF hybrid (text/layout), search, resume
- Zen mode, two-page spread, live reflow anchor
- SQLite persistence, TOML config

### v2 — Local TUI Enhancements

- TOC overlay (EPUB)
- Settings (layout + theme)
- Highlights/bookmarks (EPUB + PDF text + PDF layout)

### Post-v2 — Ecosystem Expansion

- Public SSH entrypoint: `ssh <service>` runs the app instantly (no local install)
- Backend (Go): accounts, sync, community sharing
- React Native mobile/web clients
- Multi-device reading sync, shared libraries/communities, moderation, etc.

---

## 13) Open items to capture explicitly in PRDs (so they don’t surprise engineering)

These aren’t blockers to proceed, but should be explicitly stated in version PRDs:

- Exact list of “planned future formats” beyond EPUB+PDF (TXT/MD/HTML/CBZ…)
- DRM handling behavior (import refusal vs “unreadable” entry)
- Exact managed-copy storage structure (by book_id vs preserved folders)
- Terminal capability baseline (Unicode/truecolor expectations, minimum size)
- PDF layout limitations (images/graphics) and how it’s messaged in UI

## AI Agent Implementation Workflow

1. When instructed to implement a specific version (`V1`, `V2`, and so on), agents must execute that version as a sequence of milestone-complete changes rather than one start-to-finish monolithic change.
2. The workflow applies equally across all versions and must preserve clean architecture and safety constraints defined in this document.
3. "Small and concise" applies to commit scope and diff size; commit message detail remains mandatory.

example journey: the user instructs to implement `V2`, the agent starts building the version incrementally, first implement feature 1 then do a git commit with a detailed message, then move to feature 2, and so on until all features in `V2` are implemented. Each commit should be small and focused on a single feature or change to maintain clarity and ease of review.
