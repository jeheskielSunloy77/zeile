# PRD v5 - Final Multi-Platform Reading Platform

## 1) Overview

PRD v5 defines the final product version of kern as a complete and polished multi-platform reading platform.

This version is:

- available on Android, iOS, and web
- built from one shared React Native product foundation using Expo and React Native Web
- local-first across all clients
- fully cloud-backed for cross-device library continuity
- EPUB-only by design
- community-enabled with lightweight social features
- completely free
- aligned with TUI as an active first-party client

v5 is the point where kern stops being an early client rollout and becomes the full product.

---

## 2) Goals

- Deliver a polished public product on Android, iOS, and web.
- Preserve local-first reading as a core principle across all clients.
- Support full cloud library continuity so books, reading state, and annotations follow the user across devices.
- Maintain strict major-feature parity across Android, iOS, and web.
- Keep TUI as an active first-party client aligned with the same platform model.
- Provide lightweight community features that strengthen reading without turning the product into a generic social network.
- Keep the product intentionally focused on EPUB quality instead of widening format scope.

---

## 3) Non-goals

- No PDF support in v5.
- No marketplace or e-commerce layer.
- No paid tiers, subscriptions, or monetization.
- No heavy academic/collaborative annotation platform.
- No broad social-network mechanics beyond reading-centered community features.
- No separate web codebase outside the shared React Native product foundation.

---

## 4) Product Positioning

kern v5 is a local-first, cloud-backed reading platform for people who want ownership, continuity, and a high-quality reading experience across devices.

It is not just:

- a file reader
- a sync utility
- a social reading app

It is the combination of:

- personal library ownership
- deep EPUB reading
- reliable cross-device continuity
- lightweight reading-centered community

The product should feel calm, deliberate, and premium even though it is free.

---

## 5) Platforms

### 5.1 Public product platforms

- Android
- iOS
- web

### 5.2 Client strategy

- Android, iOS, and web must have strict major-feature parity.
- Web must use React Native Web from the same shared app foundation.
- TUI remains an active first-party client, but parity expectations for TUI are based on product semantics and capability alignment rather than identical UI depth.

### 5.3 Platform rule

Platform-specific behavior should exist only where the platform forces it, such as:

- file import and acquisition mechanics
- local file persistence details
- background processing behavior
- platform-specific integration APIs

Feature logic, product behavior, and user-visible capability should otherwise be shared.

---

## 6) Product Principles

1. Local-first always: users must be able to read and work locally without making the cloud the sole source of truth.
2. Cloud-backed continuity: the full library can follow the user across devices and platforms.
3. Reading first: social and community features support reading rather than replacing it.
4. Platform parity: Android, iOS, and web should feel like the same product, not separate interpretations.
5. Explicit ownership boundaries: private library, cloud backup, local cache, and public sharing must remain clearly differentiated.
6. Focus over breadth: EPUB quality is more important than adding more formats.
7. Free product integrity: product quality and trust must not depend on monetization hooks.

---

## 7) Core Product Model

v5 should operate as a `cloud-backed local-first` platform.

This means:

- every client keeps reliable local reading state
- books can exist both as cloud assets and as local device copies/caches
- offline reading remains first-class after a book is locally available
- cloud services provide continuity, restore, and sync rather than replacing local operation

### 7.1 Content states

The product should explicitly support these states:

- `local only`: book exists on one device and has not been uploaded
- `cloud synced`: book exists in the user’s cloud library
- `local cached`: a cloud-backed book is also stored locally on a specific device for offline use

This distinction must be reflected in both product logic and UX.

---

## 8) Feature Set

### 8.1 Personal reading

- import EPUBs from supported platform file flows
- acquire EPUBs from supported built-in catalog/source integrations
- browse and organize personal library
- read EPUBs with stable resume
- table of contents navigation
- in-book search
- reader preferences
- bookmarks
- highlights
- notes

### 8.2 Cloud library

- upload imported EPUBs to the user’s cloud library
- restore library on new devices
- download/cache books to specific devices
- manage local storage and cached availability
- keep library metadata consistent across clients

### 8.3 Sync

- sync reading progress
- sync bookmarks
- sync highlights
- sync notes
- sync profile/account state
- sync library metadata and availability state

### 8.4 Community

- public profiles
- follow/following
- shareable books
- shareable lists
- shareable highlights
- comments and discussion around shared reading artifacts
- lightweight discovery/feed surfaces centered on reading activity

### 8.5 TUI alignment

TUI should continue to support:

- local reading
- account connection
- cloud library continuity where practical
- sync
- a useful subset of community participation

---

## 9) UX Requirements

### 9.1 Core UX posture

v5 must feel complete, calm, and trustworthy.

That means:

- reading remains the center of the product
- cloud features are powerful but understandable
- social features are available but not intrusive
- local, cached, cloud, and public states are always clear
- Android, iOS, and web feel consistent in structure and intent

### 9.2 Onboarding

The final product should support two valid first-run paths:

1. start reading now
2. connect/create account for cloud continuity

Neither path should make the other feel incorrect.

### 9.3 Cloud clarity

The product must clearly communicate:

- whether a book is local only, cloud synced, or cached locally
- what happens when a local cache is deleted
- what happens when a cloud copy is deleted
- whether a shared item is private, follower-visible, or public

### 9.4 Quality bar

Polished in v5 means:

- strong accessibility
- intentional empty/loading/error states
- stable migrations and upgrades
- low crash rates
- low sync confusion
- low risk of accidental data loss

---

## 10) Community and Sharing Rules

Community in v5 should remain reading-centered.

That means:

- profiles exist to represent readers, not generic creators
- follows create a lightweight social graph
- books, lists, and highlights can be shared explicitly
- comments/discussion belong around shared reading artifacts
- discovery prioritizes reading relevance, not engagement loops

Important product rules:

- a private cloud library item is not public by default
- uploading a book to the cloud library does not publish it
- public sharing requires explicit user action
- personal ownership and public sharing are separate trust boundaries

---

## 11) Technical Architecture

### 11.1 Frontend foundation

Base stack:

- Expo
- Expo Router and related Expo libraries
- React Native Web
- TanStack React Query
- Tamagui
- Zustand
- TypeScript

Dependencies should use the latest stable compatible versions at the time implementation is finalized, then be locked in the repo.

### 11.2 App structure

Use one shared app foundation in `apps/app` with modular internal boundaries:

- `src/features`
- `src/navigation`
- `src/components`
- `src/theme`
- `src/state`
- `src/data`
- `src/storage`
- `src/reader`
- `src/platform`

### 11.3 Reader subsystem

The EPUB reader should remain a dedicated subsystem with:

- shared domain models for locations, progress, bookmarks, highlights, and notes
- shared controller logic
- renderer adapters where platform internals differ

The renderer is the main area where Android, iOS, and web may need deeper implementation differences. Product behavior must remain consistent despite those differences.

---

## 12) Persistence, Storage, and Library Continuity

Use a split persistence model for app clients:

- `expo-sqlite` for structured persistent data
- `expo-file-system` or platform-equivalent local file storage for EPUB assets and derived files
- `expo-secure-store` or platform-equivalent secure credential storage

Local structured data should cover:

- library records
- metadata
- reading progress
- bookmarks
- highlights
- notes
- sync metadata
- cache/download state
- community interaction state where locally needed

The platform must support:

- importing locally
- uploading to cloud
- discovering cloud books on other clients
- downloading/caching locally for offline reading
- restoring library presence on a new device

---

## 13) Auth and Sync Requirements

### 13.1 Auth

- native/web-appropriate session models
- short-lived access credentials with refresh support
- secure credential handling on all clients
- logout that clears credentials without implicitly deleting local books unless explicitly requested

### 13.2 Sync

- local writes happen first
- sync is queue-based and failure-tolerant
- reconnect reconciles pending changes
- offline usage must remain reliable once content is locally available
- sync must cover reading state, cloud library state, and community state where applicable

### 13.3 Conflict handling

- reading progress resolves deterministically
- bookmarks are additive where possible
- highlights and notes must prefer preserving user data over destructive overwrite
- cloud-library operations must have explicit semantics across devices

---

## 14) Content Acquisition

v5 should support:

- bring-your-own EPUB import
- cloud library continuity for owned books
- curated source/catalog integrations such as OPDS or curated feeds

v5 should not become a marketplace. Acquisition should remain reader-centered and compatible with the product’s ownership model.

---

## 15) Non-Functional Requirements

- Android, iOS, and web must all be production-quality.
- Accessibility is a release requirement.
- Sync regressions are release-blocking.
- Data-loss scenarios are release-blocking.
- Upgrade and migration safety are required.
- Web must feel like a first-class product surface, not a compromised companion.
- TUI must remain compatible with the platform’s core account, library, and sync semantics.

---

## 16) Testing Strategy

Testing should prioritize behavior where the product can fail in costly ways:

- EPUB import and acquisition flows
- cloud upload/download/restore flows
- local cache management
- reader location and resume behavior
- annotation persistence and sync
- sync queue behavior and reconciliation
- account/session handling
- follows, sharing, comments, and visibility rules
- cross-platform parity for major user-facing features

Testing should include:

- unit tests for domain and sync logic
- integration tests for persistence and API behavior
- feature/UI tests for critical user flows
- cross-platform regression coverage for Android, iOS, and web

---

## 17) Risks

- full cloud library introduces more complex storage, upload, and restore behavior
- strict parity across Android, iOS, and web raises implementation discipline requirements
- EPUB rendering and annotation consistency across platforms remains a major technical risk
- community features can clutter the product if they are not tightly scoped around reading
- keeping TUI active alongside the graphical clients increases platform-maintenance scope

---

## 18) Release Standard

v5 is the final release target only when:

- Android, iOS, and web all meet public production quality
- strict major-feature parity exists across those three platforms
- local-first reading remains intact across all major clients
- full cloud library continuity works end to end
- community features are complete and polished
- TUI remains an active aligned client
- the product feels coherent, trustworthy, and complete
