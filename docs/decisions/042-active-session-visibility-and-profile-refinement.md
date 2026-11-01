# 042 — Active Session Visibility, Metadata Boundaries, and Profile Presentation Refinement

## Context and Problem Statement

The Personal Profile experience (`/app/profile`) previously exhibited several presentation, projection, and metadata boundaries issues:
1. **Activity Timeline Text Occlusion**: The activity timeline component was rendering blank event rows (icons and timestamps rendered, but titles and summaries were invisible) due to field key discrepancies between frontend expectations (`display_title`, `display_summary`) and backend payload keys (`title`, `summary`).
2. **Login Log Pollution**: Routine successful login and single logout events appeared on the personal profile timeline, cluttering personal audit trails and conflicting with the separation between user-facing personal activity and internal platform security investigations.
3. **Session Visibility vs Historical Log Confusion**: The user profile did not provide visibility into active devices. In accordance with the platform's strict invariant (maximum 2 active devices under deterministic FIFO replacement), users needed to view their currently active sessions (distinguishing current device vs other recognized device) with the ability to revoke other active sessions.
4. **Metadata Boundaries & Privacy**: The system tracks `user_agent` and `client_ip` in `user_sessions`. Raw IP addresses must never be leaked to normal user screens. Furthermore, no external geolocation provider is configured; decorative stock photo metadata (e.g. `Oslo, Norway · Nordic Light Axis`) was mistakenly appearing on the profile cover hero, misleadingly implying personal geographic identity.
5. **Cover Date Legibility & Heading Proportions**: The date badge displayed on the cover backdrop was disproportionately small (`11px`), and when a user lacked an explicit profile name, their raw phone number was rendered in a giant `5.5rem` display headline that dominated the page.

## Decisions

### 1. User-Facing Active Sessions vs Internal Security Logs
- The profile page now displays **only currently active sessions** (`revoked_at IS NULL AND expires_at > NOW()`, capped at the system invariant limit of 2).
- Users see clear device identification:
  - **This Device**: Current active browser session with single-device Sign Out action.
  - **Other Active Device**: Secondary active session with dedicated single-session Revoke Access action.
  - Slot status indicator showing `{count} of 2 Devices Active`.
  - Global emergency termination action ("Log Out All Devices").
- Historical security and authentication event history remains internal to `platform_audit_logs` for authorized audit and administrative review.

### 2. Device and Geolocation Metadata Handling
- **Device & Browser Labels**: Derived from the HTTP `User-Agent` header using deterministic parsing into OS and browser families (e.g., `Chrome on macOS`, `Safari on iOS`). Unparseable agents fall back to `Unknown Device` without fabricating names.
- **Privacy Enforcement**: Raw client IP addresses are omitted from all client-facing session responses.
- **Geolocation**: Because no external GeoIP service is provisioned in the application environment, geolocation is omitted or marked "Unknown" rather than fabricated.

### 3. Activity Timeline Projection & Filtering
- Fixed the field mapping by projecting both `title`/`summary` and `display_title`/`display_summary` in `UserActivityEvent` and falling back cleanly in SvelteKit.
- Excluded routine `USER_LOGIN` and `USER_LOGOUT` events from the user-facing timeline feed.
- Persisted and displayed high-priority security notifications:
  - `SESSION_ROTATED`: Recorded in `platform_audit_logs` when FIFO session replacement retires the oldest active device upon sign-in.
  - `USER_LOGOUT_ALL`: Recorded when global revocation is triggered.
  - `SESSION_REVOKED`: Recorded when a specific active session is terminated.

### 4. Profile Cover Presentation & Date Typography
- **Cover Image Metadata Cleanup**: Removed decorative location (`backdrop.location`) and style (`backdrop.title`) pills from the cover hero. Retained discrete Unsplash photographer attribution (`Photo by [photographer]`).
- **Date Typography**: Increased the date badge font size from `11px` to `text-xs sm:text-sm md:text-base` with increased padding and contrast, ensuring readability across desktop and mobile without wrapping.
- **Identity Heading Hierarchy**:
  - When profile name (`first_name` / `last_name`) exists, it renders as the primary editorial serif headline (`h1`).
  - When no name exists, a restrained, elegant `Personal Profile` title is displayed alongside an invitation to complete profile details, keeping verified phone numbers in their proper role as secondary cryptographic identity details.

## Consequences

- Active session count is strictly bounded to at most 2 items.
- Session revocation is concurrency-safe and ownership-verified (`user_id = auth.user_id`).
- No raw IP addresses or unverified locations are exposed to users.
- Timeline events render with clear titles and summaries.
- Mobile viewports remain responsive without horizontal overflow.

## Related Documentation
- [[../api/02-authentication-api|Authentication API Specification]]
- [[../architecture/05-authentication|Authentication Architecture]]
- [[../decisions/041-gatekeeper-experience-visitor-identity-and-activity-timeline|ADR 041]]
