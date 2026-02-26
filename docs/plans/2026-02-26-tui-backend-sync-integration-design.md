# TUI Backend Sync Integration Design

Date: 2026-02-26
Status: Approved for planning

## 1. Summary

This design integrates the local-first TUI reader with the backend platform while preserving offline usability. Users can continue reading and managing books locally without authentication, then connect an account using device authorization flow to enable automatic background sync for library, reading progress, highlights, sharing, and community features.

## 2. Scope

In scope:
- Device authorization login flow for TUI.
- Local-first behavior with optional account connection.
- One-time first-login reconciliation from local library to backend.
- Automatic sync (event-driven and periodic) plus manual sync trigger.
- Sync coverage for library, reading state, highlights, sharing, and community data.
- API adjustments needed for non-browser device auth and TUI session lifecycle.

Out of scope:
- Forcing login before using TUI.
- Replacing current local reader stack with cloud-only behavior.
- Full event-sourced sync rewrite.

## 3. Final Decisions

- Runtime model: local-first with sync.
- Auth UX: browser-based device authorization flow.
- Device flow variant: device authorization code + polling.
- Sync surface: library + progress + highlights + sharing + community.
- Unauthenticated mode: full local experience remains available.
- Sync execution: automatic background sync with manual "Sync now".
- First-login behavior: one-time reconcile/migrate existing local library.

## 4. Architecture

### 4.1 TUI modules

- `auth`: device authorization session orchestration and token handling.
- `remote`: typed backend client for auth/library/sharing/community/moderation endpoints.
- `sync`: reconcile engine, outbound queue processing, periodic pull loop, and retries.
- `mapping`: local-to-remote entity link and version metadata management.

### 4.2 Existing boundaries to preserve

- Keep local domain/application services as source of truth for reader UX.
- Keep local SQLite and managed files as primary offline persistence.
- Keep current API business logic inside backend services; TUI remains a client.

### 4.3 API additions

- Add device authorization endpoints:
  - start device auth (returns user code + verification URI + polling interval).
  - poll auth completion (returns tokens/session once approved).
  - approve/complete auth action on verified browser side.
- Ensure refresh/revoke behavior supports non-cookie TUI client sessions.

## 5. Data Model Changes (TUI SQLite)

Add sync metadata tables:
- `sync_accounts`: active account identity and token/session metadata.
- `sync_book_links`: map local `book_id` to remote `user_library_book_id`, `catalog_book_id`, and preferred asset.
- `sync_state_versions`: track remote versions/etags and last-synced timestamps per entity.
- `sync_queue`: outbound operations with state (`pending`, `in_progress`, `failed`), retry count, and next-attempt timestamp.

Design intent:
- Keep existing `books` and `reading_state` tables unchanged for local continuity.
- Keep sync metadata isolated to avoid coupling local reading code to remote IDs.

## 6. Sync Flows

### 6.1 First-login reconcile (one-time)

1. User starts device auth in TUI and approves in browser.
2. TUI receives session/tokens and enters connected mode.
3. For each local book:
   - create/find catalog book remotely.
   - upload/link asset when allowed.
   - upsert remote user library book.
   - persist local-to-remote mapping.
4. Push local reading states and highlights.
5. Pull remote sharing/community/moderation relevant records.
6. Save baseline version metadata for incremental sync.

### 6.2 Ongoing automatic sync

- Event-driven push when local writes occur.
- Periodic pull/push loop for remote updates and missed events.
- Manual "Sync now" command for immediate cycle.

### 6.3 Conflict strategy

- Reading state: use backend version contract; on conflict, fetch latest and apply newer/higher-progress winner rule.
- Highlights/sharing/community: prefer server state on hard conflicts; retain local op as flagged retry when merge is ambiguous.
- Sync failures must never block core reading/library interactions.

## 7. Error Handling and UX States

Connection states:
- `Not connected`: local-only mode.
- `Connecting`: device flow in progress.
- `Connected`: sync active.
- `Degraded`: auth/network/retry failures.

UX rules:
- Show compact status in footer or small sync panel.
- Surface actionable messages (`reauth required`, `retrying`, `conflict needs review`).
- Attempt refresh before failing protected calls; fall back to degraded mode on failure.
- Use idempotency keys for retryable write operations.

## 8. Testing Strategy

TUI unit tests:
- Device auth state machine and token transitions.
- Sync scheduler decisions and retry/backoff behavior.
- First-login reconciliation and mapping persistence.
- Conflict resolution and queue lifecycle.

TUI integration tests:
- Local DB with mocked backend flows.
- Offline/online transitions and eventual consistency checks.
- Full cycle: local change -> queued op -> remote ack -> version update.

API tests:
- Device auth handler/service behavior.
- Token refresh/revoke for TUI sessions.
- Regression coverage for existing library/sharing/community/moderation contracts.

## 9. Delivery Exit Criteria

- Device auth works end-to-end for TUI login.
- Existing local library can be reconciled on first login.
- Automatic sync runs for library/progress/highlights/sharing/community.
- Manual "Sync now" is available.
- Unauthenticated and offline local reading remains fully usable.

## 10. Risks and Mitigations

- Risk: first-login reconcile takes too long on large libraries.
  - Mitigation: progress feedback, resumable queue, and partial completion checkpoints.
- Risk: conflict rules confuse users.
  - Mitigation: deterministic policy, clear status text, and minimal manual resolution points.
- Risk: auth/session edge cases in headless terminal environments.
  - Mitigation: strict device flow states, explicit timeout handling, and robust reauth prompts.

