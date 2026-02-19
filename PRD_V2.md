# PRD v2 — Local TUI Enhancements (TOC + Settings + Highlights)

## 1) Overview

v2 expands the fully local TUI app with three major additions while preserving the immersion-first philosophy:

1. **TOC overlay drawer** in the Reader
2. **Global Settings** view for reading layout + theme (terminal-realistic “font customization” reframed)
3. **Highlights/Bookmarks** including support for **EPUB**, **PDF text mode**, and **PDF layout mode** (region selection)

v2 remains **local-only**: no backend, no auth, no community sharing, no public SSH entrypoint.

---

## 2) Goals

- Improve navigation inside books via **TOC overlay** (EPUB).
- Improve readability and comfort via **global layout + theme settings**.
- Enable readers to capture and revisit important passages via **highlights/bookmarks**.
- Keep the experience polished, minimal, and consistent with terminal.shop-inspired aesthetics.
- Maintain v1 guarantees: fast page turning, stable progress, live reflow anchored to text position.

---

## 3) Non-goals

- No per-book settings (global settings only).
- No notes/comments on highlights (can be future).
- No backend sync of highlights/settings.
- No community sharing.
- No bulk folder import.
- No public SSH entrypoint (post-v2).

---

## 4) v2 feature set

### 4.1 TOC overlay drawer (Reader)

**Scope:** EPUB only in v2 (PDF TOC support is optional future work).

#### Functional requirements

**FR-TOC-1**: Reader provides a TOC overlay drawer via `t`.
**FR-TOC-2**: TOC overlay is a drawer-style overlay (not a separate full-screen view).
**FR-TOC-3**: Selecting a TOC item jumps to that location and closes the overlay.
**FR-TOC-4**: Overlay is usable in normal mode and in zen mode via a minimal overlay interaction (zen remains visually minimal).
**FR-TOC-5**: If the EPUB has no usable TOC, show “TOC not available” with graceful fallback.

#### UX requirements

- Drawer shows chapter titles with clear hierarchy if possible (indent levels).
- Provide minimal hints: `↑↓` move, `Enter` jump, `Esc` close.

---

### 4.2 Global Settings view (layout + theme)

v2 introduces a new **Settings** view accessible from Library and/or Reader (recommended: `s`).

#### Settings included (global only)

- **Reading width / column width** (for text modes)
- **Margins/padding** (horizontal and vertical)
- **Line spacing** (simulated)
- **Paragraph spacing** (simulated)
- **Theme**: at minimum, a small set of built-in themes (e.g., default + high contrast + dim)

> Note: Terminal font family/size is controlled by the terminal emulator, not the app. v2 settings cover what the app can realistically control.

#### Functional requirements

**FR-SET-1**: Settings are global only (no per-book overrides).
**FR-SET-2**: Settings are persisted (SQLite and/or TOML; see Data).
**FR-SET-3**: Reader immediately applies settings changes (live).
**FR-SET-4**: Settings changes trigger live reflow while anchoring to same text position.
**FR-SET-5**: Settings view uses keyboard navigation and matches overall UI style.

---

### 4.3 Highlights and bookmarks

v2 adds highlighting and bookmarking capabilities.

#### Supported content modes

- EPUB (text-flow selection)
- PDF text mode (text-flow selection)
- PDF layout mode (**region selection**)

#### Entry points & navigation

- `v` enters **Highlight mode**
- Highlights can be listed in a new view or overlay (see below)
- Users can jump to a highlight from the list

#### Highlight selection UX

**Text-flow selection (EPUB + PDF text mode)**

- Cursor-based selection across rendered text
- Start/end selection
- Confirm highlight action

**PDF layout selection**

- Crosshair cursor over terminal-rendered page grid
- `Enter` sets corner A
- Move cursor
- `Enter` sets corner B (rectangle appears)
- `h` confirms highlight

#### Functional requirements

**FR-HL-1**: User can create a highlight from Reader.
**FR-HL-2**: User can view a list of highlights per book (new view or overlay; see “Views”).
**FR-HL-3**: Selecting a highlight jumps to its location.
**FR-HL-4**: Highlights are persisted in SQLite.
**FR-HL-5**: Highlight mode and minimal overlays work in zen mode without fully exiting zen visually.
**FR-HL-6**: Highlights render visually when viewing the page:

- Text modes: inverted color/underline style over selected range (as terminal allows)
- PDF layout: rectangle region inversion/overlay on the page grid

#### Bookmarks

v2 may implement bookmarks as either:

- a separate “bookmark” entity, or
- a highlight with a special kind (no range, just a point)

**FR-BM-1**: User can add a bookmark and jump back to it.
(If you want bookmarks explicitly separate from highlights, we’ll specify it; otherwise treat as highlight-kind.)

---

## 5) Views and navigation (v2)

v1 has 3 views; v2 adds:

### Added views/overlays

1. **Settings View** (full-screen view)
2. **Highlights List View** (full-screen) _or_ overlay panel
3. **TOC overlay drawer** (overlay inside Reader)

#### Requirements

**FR-VIEW-1**: TOC is an overlay in Reader (not a top-level view).
**FR-VIEW-2**: Settings is a dedicated view.
**FR-VIEW-3**: Highlights list is accessible from Reader (keybinding TBD; recommended `H` or `l`), and from there user can jump to a highlight.

---

## 6) Data requirements (v2)

### SQLite additions

Add tables (or extend schema) from v1:

- `settings_global`

  - key/value or structured columns for:

    - reading_width, margin_h, margin_v
    - line_spacing, paragraph_spacing
    - theme_id
    - updated_at

- `highlights`

  - id, book_id, created_at, updated_at
  - kind: `epub_text` | `pdf_text` | `pdf_layout`
  - locator_json:

    - EPUB/PDF text: chapter/page + offsets + optional snippet hash
    - PDF layout: page_number + rect {x1,y1,x2,y2} + render_grid_id

  - quote_text (best-effort extracted snippet)
  - note (nullable; future)

- Optional `bookmarks` (if separated), else encode as highlight-kind.

### TOML config

- Continue to use TOML for app configuration. Settings that affect reading experience should be stored in SQLite for transactional consistency, but mirroring defaults in TOML is acceptable.

---

## 7) Non-functional requirements (v2)

- **No regressions** vs v1 performance and stability.
- Highlight mode should feel responsive:

  - entering/exiting highlight mode must not lag
  - rendering highlighted regions must be fast

- Settings changes should apply immediately with stable anchor.

---

## 8) Keybindings (v2 additions)

- `t`: TOC overlay
- `s`: Settings view
- `v`: Highlight mode
- `h`: Confirm highlight (in highlight mode; especially for PDF layout region confirm)
- `Esc`: cancel/close overlays or exit highlight mode
- Highlights list: recommended `H` to open list (or `l`); must be documented

Zen mode:

- All of the above remain accessible through minimal overlays (no full chrome reappearing unless necessary for usability).

---

## 9) Out of scope explicitly (v2)

- Backend, auth, users
- Community sharing
- Public SSH entrypoint (`ssh <service>`)
- Mobile/web clients
- Per-book settings
- Highlight notes/comments (future)
- PDF TOC parsing/navigation (optional future)

---

## 10) Acceptance criteria (v2)

v2 is acceptable when:

1. Reader can open TOC overlay (`t`) for EPUBs with TOC and jump to chapters.
2. Settings view exists; user can change global layout/theme settings and they apply immediately.
3. User can create highlights in:

   - EPUB text-flow
   - PDF text mode
   - PDF layout mode via region selection

4. Highlights persist across restart and render visibly in the reader.
5. Highlights list can be opened and used to jump to highlights.
6. All v1 requirements still pass (import, hybrid PDF, search, resume, live reflow anchor, zen).
