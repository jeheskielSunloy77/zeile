# PRD v4 - Local-First Android EPUB App

## 1) Overview

PRD v4 defines the first mobile platform release for kern.

This version is:

- Android-only for public release
- built as one Expo app in `apps/app`
- local-first by default
- EPUB-only
- account-optional
- free
- architected so web can later use React Native Web and iOS can follow from the same codebase

v4 is not a separate web project and not a cloud-first app. It is a local reading product first, with account and sync layered on top to support continuity later.

---

## 2) Goals

- Ship a credible Android alpha for local EPUB reading.
- Let users import and manage their own EPUB files from device storage.
- Deliver a strong deep-reading experience with highlights, bookmarks, notes, and in-book search.
- Preserve full app usability without requiring login.
- Add optional account connection for profile continuity and state sync.
- Structure the frontend so future web support uses React Native Web, not a separate client codebase.
- Keep the product approachable enough to expand beyond power users later.

---

## 3) Non-goals

- No PDF support in v4.
- No separate `apps/web` codebase in v4.
- No public web release in v4.
- No iOS release in v4.
- No raw EPUB upload or cloud backup in v4.
- No mandatory sign-in wall.
- No payments, subscriptions, or monetization.
- No OPDS, marketplace, or remote catalog ingestion.
- No community/sharing features inside the mobile app in v4.

---

## 4) Target Audience

v4 targets a bridge audience:

- readers who already own EPUB files and care about offline access, control, and data ownership
- users who are not necessarily technical, so the app must still feel clean and approachable

This means the product should satisfy power-reader expectations without presenting itself like a niche utility.

---

## 5) Platforms and Release Posture

### 5.1 Release target

- Public platform in v4: Android
- Release posture: internal alpha

### 5.2 Platform planning from day 1

- Web is planned from day 1 but not shipped in v4.
- Web must use React Native Web from the same app foundation.
- iOS is not shipped in v4, but architectural choices must not block it later.

### 5.3 Codebase rule

- Build one Expo app in `apps/app`.
- Do not create a separate web app codebase for this product.

---

## 6) Product Principles

1. Local-first ownership: reading and library workflows must work without an account.
2. Fast time to reading: users should be able to import and start reading immediately.
3. Optional cloud: account features add continuity but never gate core use.
4. Shared product foundation: build one React Native product foundation that can extend to web and iOS later.
5. Honest platform promises: do not imply file backup or full-device restore before those capabilities exist.
6. Deep reading over content breadth: focus on EPUB and annotation quality instead of supporting many formats too early.

---

## 7) Feature Set

### 7.1 Local library

- import EPUB files from Android device storage
- support file picker and Android open/share entry points
- store local library metadata and derived covers/assets
- support basic organization and browsing of imported books

### 7.2 EPUB reading

- open and read reflowable EPUB books
- remember reading position
- expose table of contents
- support reading preferences such as theme, font sizing, spacing, and similar reader controls
- derive progress from stable reading locations

### 7.3 Deep reading

- create and manage bookmarks
- create and manage highlights
- create inline notes and review notes later
- support in-book search

### 7.4 Optional account connection

- sign in only if the user chooses to
- manage profile basics
- sync supported reading state across sessions/devices in future-compatible form

### 7.5 Sync scope in v4

- profile basics
- library metadata
- reading progress
- bookmarks
- highlights
- notes data where backend support exists

### 7.6 Explicitly out of scope in v4

- raw EPUB cloud backup
- restoring a library from cloud files
- community publishing/sharing inside mobile

---

## 8) UX Requirements

### 8.1 First-run experience

The default path should be:

1. open app
2. import book
3. start reading

Account prompts must not appear as a gate before the user experiences core value.

### 8.2 Product tone

- language should be reader-friendly and plain English
- screens should not feel like developer tooling
- account and sync messaging should be clear but secondary

### 8.3 Data-loss posture

If a user never connects an account, local data is local only. Losing that data on uninstall/reinstall is acceptable in v4. The product should not hide that trade-off in settings or account screens.

---

## 9) Technical Architecture

### 9.1 Frontend stack

Base stack:

- Expo and Expo libraries, including Expo Router
- TanStack React Query
- Tamagui
- Zustand
- TypeScript

Dependency policy:
- implementation should use the latest stable compatible versions available at the time the app is initialized
- exact versions should be locked in the repo once chosen

### 9.2 App structure

Use one app in `apps/app` with internal module boundaries:

- `src/features`
- `src/navigation`
- `src/components`
- `src/theme`
- `src/state`
- `src/data`
- `src/storage`
- `src/reader`
- `src/platform`

No `packages/ui` should be created for v4.

### 9.3 Reader boundary

The reader should be its own subsystem with:

- shared domain models for positions, progress, highlights, bookmarks, and notes
- shared controller logic
- renderer adapters that can vary by platform

The renderer is the main place where Android, web, and iOS may diverge later. The rest of the product should be shared as much as practical.

---

## 10) Local Persistence and File Handling

Use a split persistence model:

- `expo-sqlite` for structured persistent data
- `expo-file-system` for EPUB files and derived assets
- `expo-secure-store` for auth/session credentials

SQLite data should include:

- library records
- parsed metadata
- reading progress
- bookmarks
- highlights
- notes
- sync queue and remote mapping metadata

The file system should hold imported EPUBs and derived reader assets.

Zustand should not be treated as the primary persistent database.

---

## 11) Auth and Sync Requirements

### 11.1 Auth model

- native-app-friendly token/session model
- short-lived access credentials with refresh support
- secure credential storage via `expo-secure-store`
- logout clears credentials without deleting local books unless explicitly requested

### 11.2 Sync model

- local writes happen first
- sync runs only when account connection exists
- supported changes are queued and retried
- sync failures must not block local reading

### 11.3 Conflict expectations

- reading progress resolves deterministically
- bookmarks are additive where possible
- highlights and notes should preserve user data conservatively

---

## 12) Non-Functional Requirements

- Android alpha should feel stable enough for real reading sessions.
- Local persistence must be trustworthy.
- Import and reader startup should feel responsive on normal modern Android devices.
- Crashes and corrupted local state are higher priority problems than visual polish.
- Architecture must remain compatible with future React Native Web adoption.

---

## 13) Testing Strategy

Focus testing on decision-heavy behavior:

- reader domain logic
- annotation anchoring
- reading progress calculation
- local persistence and migrations
- import pipeline behavior
- sync queue and retry logic
- auth/session refresh and logout handling
- UI flows for import, reading, annotation, settings, and account connection

Do not spend time snapshot-testing large UI surfaces with weak behavioral value.

---

## 14) Rollout Plan

1. Foundation
- Expo app shell
- Tamagui setup
- navigation
- local persistence foundation
- EPUB import pipeline
- basic reader rendering

2. Reader alpha
- library
- progress
- bookmarks
- highlights
- notes
- reading preferences

3. Connected alpha
- optional account connection
- profile basics
- sync for supported state
- degraded/offline behavior

4. Hardening
- migrations
- crash reduction
- UX cleanup
- instrumentation
- future React Native Web readiness checks

---

## 15) Risks

- EPUB rendering and stable annotation anchors are the main technical risk.
- Premature optimization for web can slow Android delivery.
- Under-planning for web can recreate the separate-codebase problem later.
- Sync expectations can drift into false file-backup promises if the UX is careless.

---

## 16) Exit Criteria

- `apps/app` exists as the single Expo app for the product
- Android is the only shipped platform in v4
- EPUB import and reading work locally
- highlights, bookmarks, notes, and search are implemented at alpha quality
- users can read fully without signing in
- optional account connection works
- supported reading state sync works without breaking offline-first behavior
- the architecture remains suitable for future React Native Web and iOS support
