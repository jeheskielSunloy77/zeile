# PRD v1 — Local TUI MVP (EPUB + PDF Hybrid)

## 1) Overview

v1 delivers the first usable version of the product as a **fully local, install-and-run TUI e-reader**. The focus is on an **immersive reading experience** with a minimal but polished library. There is **no backend**, **no accounts**, and **no sharing**.

v1 includes exactly **3 views**:

1. Library List
2. Upsert/Add Book
3. Reading View (with overlays)

The UI is full-screen and inspired by terminal.shop’s calm, premium terminal aesthetic.

---

## 2) Goals

- Enable users to **import EPUB/PDF** into a local library.
- Provide a **fast, pleasant library list** with search (`/`).
- Provide an immersive **two-page reading** experience with **Zen mode**.
- Support **PDF hybrid reading**: Text mode + Layout mode.
- Ensure robust **progress tracking** and **auto-resume** to last unread book at startup.
- Provide **in-book search** (on-demand scan) for EPUB + PDF text mode content.
- Keep the UI responsive and stable (no flicker, no blocking UI during heavy work without feedback).

---

## 3) Non-goals

- No authentication/users.
- No backend services.
- No TOC (moved to v2).
- No highlights/bookmarks (moved to v2).
- No bulk folder import (single-book import only).
- No PDF zoom controls in layout mode.
- No mobile/web clients.

---

## 4) Personas

- **Terminal reader:** wants a calm, distraction-free terminal reading experience.
- **SSH power user:** reads in terminal sessions; expects keyboard-first navigation and resilience on resize (even though public SSH access is post-v2).

---

## 5) User stories

### Library & import

- As a user, I can add an EPUB or PDF by **entering a path**.
- As a user, I can add an EPUB or PDF using a **file browser**.
- As a user, the app **copies** my book into its managed library storage by default.
- As a user, I can **search** my library using `/`.
- As a user, I can remove a book from the library.
- As a user, I can optionally delete the book file from disk after a scary confirmation.

### Reading

- As a user, I can open a book and read in a **two-page spread**.
- As a user, I can toggle **Zen mode** to hide all controls.
- As a user, I can turn pages quickly without lag.
- As a user, if I quit and reopen, the app resumes where I left off.
- As a user, if my terminal resizes, the text reflows while keeping my reading position.

### Search & navigation

- As a user, I can search within a book (`/`) and move between matches (`n/N`).
- As a user, I can jump to a page (`g`) or percent (`G`).
- As a user, I can toggle PDF modes (`m`) and the app remembers both positions.

---

## 6) Functional requirements

### 6.1 App startup behavior

**FR-START-1**: On startup, if there exists at least one unfinished book, open the **most recently read unfinished book** directly in the Reading View at its saved position.
**FR-START-2**: If no unfinished book exists, open the Library List view.
**FR-START-3**: “Finished” state is **manual**—a book is only finished when the user marks it finished (see Reading requirements).

### 6.2 Library List view

**FR-LIB-1**: Full-screen list of books with “rich” rows (minimum: Title, Author, Progress %, Last Opened).
**FR-LIB-2**: Press `/` to open a search prompt; search filters the list based on token/ranking match over title/author/filename.
**FR-LIB-3**: Selecting a book and pressing `Enter` opens it in Reading View.
**FR-LIB-4**: Provide a visible affordance/hint for adding a book (e.g., `a Add`).
**FR-LIB-5**: Provide a remove-from-library action (e.g., `r Remove`) that does not delete files by default.
**FR-LIB-6**: If delete-from-disk is available, it must be guarded by a scary confirmation flow (see Safety).

**Ranking rules (v1 “good enough”)**

- Prefer exact/prefix matches over substrings.
- Boost recently opened books.
- Boost unfinished books.

### 6.3 Upsert/Add Book view

**FR-ADD-1**: Users can add a book via:

- Path input (paste/type a file path)
- File browser picker

**FR-ADD-2**: Supported formats for successful import in v1: **EPUB and PDF**.

- Unsupported formats should show a clear error.

**FR-ADD-3**: Default behavior is **Managed Copy ON**:

- The imported file is copied into an app-managed directory.
- The original path may be stored as metadata for reference.

**FR-ADD-4**: Import performs **preprocessing on import** (not lazy):

- Extract metadata (title/author when possible)
- Compute fingerprint/dedup key
- Prepare caches needed for reading and search:

  - EPUB: extracted plaintext by spine/chapter
  - PDF: extracted text by page for text mode
  - PDF: layout-mode parsing artifacts needed for best-effort page rendering

**FR-ADD-5**: UI must show progress during preprocessing (e.g., parsing, extracting, caching) and remain responsive.
**FR-ADD-6**: Users can cancel import while preprocessing; partial imports must be cleaned up or marked incomplete.

### 6.4 Reading View (core)

**Layout**
**FR-READ-1**: Reading view is full-screen and immersive.
**FR-READ-2**: The view renders a **two-page spread** when terminal width allows; otherwise it falls back to single page.
**FR-READ-3**: Each page shows a **page number** (in zen mode it is the only UI element besides text).

**Navigation**
**FR-READ-4**: Primary navigation uses **arrow keys**; vim keys are also supported.
**FR-READ-5**: Page forward/back are instantaneous (target performance in NFR).
**FR-READ-6**: `g` opens go-to page prompt; `G` opens go-to percent prompt.

**Zen mode**
**FR-READ-7**: `z` toggles Zen mode.
**FR-READ-8**: In Zen mode, all controls and chrome are hidden; only the two pages and page numbers remain visible.
**FR-READ-9**: Zen mode still allows actions via **minimal overlays**:

- `/` search prompt
- `g/G` go-to prompts
- (No TOC/highlights in v1)

**Live reflow**
**FR-READ-10**: On terminal resize, content reflows live.
**FR-READ-11**: After reflow, the reader anchors to the **same text position** as before resize (not the same page number).

**Progress & finished**
**FR-READ-12**: Progress is persisted frequently enough to survive crashes (e.g., on page turn and on exit).
**FR-READ-13**: The user can manually mark a book as finished (keybinding TBD, but must exist in v1).
**FR-READ-14**: Finished state affects startup auto-resume logic.

### 6.5 EPUB support

**FR-EPUB-1**: Parse EPUB container, spine order, and extract readable text content with paragraph breaks.
**FR-EPUB-2**: Pagination is based on terminal size and current reading layout defaults (v1 defaults only; no settings view yet).
**FR-EPUB-3**: Store progress using stable locators (chapter/spine + offsets) so it survives reflow.

### 6.6 PDF support (hybrid)

**Modes**
**FR-PDF-1**: Provide PDF Text mode + PDF Layout mode.
**FR-PDF-2**: `m` toggles between modes.
**FR-PDF-3**: The app remembers **separate positions** for text and layout modes and restores the correct one when toggling.

**Text mode**
**FR-PDF-T-1**: Extract text per page during preprocessing and paginate into virtual pages.
**FR-PDF-T-2**: In-book search works in PDF text mode via on-demand scan.

**Layout mode**
**FR-PDF-L-1**: Layout mode renders by PDF page (not continuous scroll).
**FR-PDF-L-2**: When width allows, show two PDF pages side-by-side (spread).
**FR-PDF-L-3**: No zoom controls in v1.
**FR-PDF-L-4**: Rendering is Go-native best-effort and prioritizes fidelity within those constraints (text placement preferred; images/graphics may be omitted depending on library capabilities).

### 6.7 In-book search

**FR-SEARCH-1**: `/` opens search prompt; Enter confirms.
**FR-SEARCH-2**: Search uses on-demand scan of preprocessed text content for current mode/content.
**FR-SEARCH-3**: `n` goes to next match, `N` to previous match.
**FR-SEARCH-4**: Search UI is minimal and works in zen via overlay.

### 6.8 Safety requirements (delete-from-disk)

**FR-SAFE-1**: Default removal action removes from library only.
**FR-SAFE-2**: Delete-from-disk is optional and must require scary confirmation:

- Show full file path and warning
- Require typing `DELETE` (case-sensitive)
- Only then perform deletion

---

## 7) Data requirements

### SQLite (minimum v1)

- `books`:

  - id, fingerprint, title, author, format, added_at, last_opened_at
  - source_path (optional), managed_path (required if managed copy)
  - metadata_json (optional), size_bytes

- `reading_state`:

  - book_id, mode (epub/pdf_text/pdf_layout)
  - locator_json (stable offsets; for PDF layout: page index + cursor/region anchor)
  - progress_percent, updated_at
  - is_finished (boolean)

- `config_meta` (optional): schema version, etc.

### File storage

- Managed copies stored under app data dir.
- Preprocessed caches stored under app data dir.

### Config

- TOML config file for user-level configuration (paths and behavior toggles). v1 ships with sensible defaults.

---

## 8) Non-functional requirements (NFR)

- **Responsiveness:** Page turns should feel instant; UI should not freeze during import (show progress).
- **Stability:** Crash-safe progress persistence.
- **Cross-platform:** Runs on major OSes where Go can run in terminal.
- **Terminal UX:** Full-screen, minimal flicker, clean redraws.
- **Performance targets (guidelines):**

  - Page turn (cached): ~<100ms perceived
  - Startup resume (warm): ~<500ms perceived
  - Import preprocessing: visible progress; acceptable if longer for large PDFs

---

## 9) UX specifics and keybindings (v1)

**Global**

- `q`: back/quit
- `?`: help overlay/modal

**Library**

- `/`: search
- `a`: add book
- `Enter`: open
- remove/delete actions: defined in UI (must include scary confirm flow)

**Reader**

- Arrow keys: navigate
- vim keys: also supported
- `z`: zen
- `/`: search
- `n/N`: next/prev match
- `g`: go-to page
- `G`: go-to percent
- `m`: toggle PDF mode
- `f` (proposed): mark finished (must be implemented; exact key can be adjusted but should be documented)

---

## 10) Technical requirements (v1)

- Language: **Go**
- Architecture: **Clean Architecture**
- TUI: **Charm toolkit** (Bubble Tea, Lip Gloss, Bubbles, Huh)
- DB: **SQLite pure Go**
- PDF: **Go-native**, based on an existing Go PDF parser library (no native deps)
- Repo: **Monorepo** with **Turborepo**
- CI: **GitHub Actions**

  - `go test ./...`
  - `golangci-lint`
  - formatting/vet checks as needed

---

## 11) Out of scope explicitly (v1)

- TOC UI and navigation
- Highlights/bookmarks
- Settings view (layout/theme)
- Bulk import folder
- Backend, auth, users
- Community sharing
- Public SSH entrypoint

---

## 12) Acceptance criteria (v1)

A v1 build is acceptable when:

1. User can import an EPUB and read it with two-page spread + zen mode.
2. User can import a PDF and:

   - read in text mode
   - toggle to layout mode and see page-based rendering
   - toggle back and positions are remembered separately

3. Library shows rich rows and `/` search filters results correctly.
4. App resumes most recent unfinished book on startup.
5. Resizing terminal reflows while preserving the same reading position.
6. In-book search works for EPUB and PDF text mode and is usable in zen.
7. Progress persists reliably across restarts.
8. Remove-from-library works; delete-from-disk requires scary confirmation.
