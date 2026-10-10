# S15 — Notifications

## Objective
Implement centralized notification infrastructure delivering in-app alerts, mobile push notifications, email/SMS dispatches, and granular user preference controls.

## Scope
### Backend
- Central notification dispatcher and template rendering service.
- Integration with delivery providers (evaluating Firebase/FCM for push notifications, SMS/Email providers).
- Granular user notification preferences schema and enforcement.
- Notification delivery status tracking and retry handling.

### Frontend
- In-app notification center (bell icon, unread count, notification list).
- User notification preferences management screen (channel toggles per category).
- Push notification subscription and permission request prompt.
- Real-time notification updates across active sessions.

## Dependencies
- [[S10-society-communication]]

## Deliverables
- Multi-channel notification delivery service.
- User preference management preventing unwanted message dispatches.
- Functional in-app notification center and push notification flow.

## Related Documentation
- [[README|Master Sprint Index]]
- [[../eagle_view|Eagle View Architecture (Section 9: Supporting Services)]]

## Status
Planned
