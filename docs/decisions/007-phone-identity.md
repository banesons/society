# ADR 007: Phone Number as Primary Identity and Canonical Normalization

## Status
Accepted

## Context
Cooperative Housing Society (CHS) governance and Gate operations in India are centered around mobile communication. Email addresses are often absent or rarely monitored by residents, domestic staff, and security personnel. We must establish a universal, unambiguous primary identity.

## Decision
1. **Primary Identity**: The user's mobile number is the primary identity in CHS. A user account is identified by a unique `phone` column in the `users` table.
2. **Canonical E.164 Format**:
   - All phone numbers must be normalized to canonical E.164 format for India (`+91XXXXXXXXXX`) before database lookup, insertion, OTP generation, or rate limiting.
   - Normalization removes whitespace, hyphens, parentheses, leading zeros, and prefixes, converting valid 10-digit Indian numbers (starting with 6-9) to `+91XXXXXXXXXX`.
   - Any phone number failing the regex `^\+91[6-9]\d{9}$` is rejected with `INVALID_PHONE`.
3. **PII Masking**:
   - Raw canonical phone numbers are stored internally in the database.
   - Public and resident-facing API responses must return masked phone numbers by default (e.g. `+91 9967xxxx01`), preventing unnecessary PII leakage.

## Consequences
- Every person has exactly one account per physical mobile number.
- No duplicate records due to differing input formats (e.g. `9967089101`, `09967089101`, `+91 99670 89101` all map to `+919967089101`).

## Related Documentation
- [[../architecture/05-authentication|Authentication Architecture]]
- [[../database/01-authentication-schema|Authentication Schema]]
- [[../security/01-authentication-security|Authentication Security]]
