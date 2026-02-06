# Template: How File

Instructions:

- Group uncertainties by topic when appropriate (when they share common concepts, entities, or problem domains)
- Use valid markdown footnotes for references
- Omit footnotes if no web search was performed

Example:

```markdown
# How: Add user authentication

## Authentication method selection

Use OAuth 2.0 with JWT tokens (confidence: high).

Rationale: OAuth 2.0 is the industry standard for secure authentication and authorization.[^1]

Alternatives:

- Session-based authentication (confidence: medium)
  - Simpler to implement but less scalable[^2]
- Basic authentication (confidence: low)
  - Not recommended for production use

[^1]: Hardt, D. (2012). *The OAuth 2.0 Authorization Framework*. [IETF RFC 6749](https://tools.ietf.org/html/rfc6749)
[^2]: OWASP (2021). *Session Management Cheat Sheet*. [OWASP](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
```
