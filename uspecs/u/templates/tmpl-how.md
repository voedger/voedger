# Template: How File

Instructions:

- Focus on implementation approach, not detailed design
- Keep it concise - this is an idea, not a full plan
- When you mention a file always include its folder, `u/actn-uhow.md`, not just `actn-uhow.md`
- Lists are not exhaustive - cover the most important points, not every detail

## First iteration (How File does not exist)

Sections:

- `## Approach` - general strategy, patterns, technologies
  - Use plain file names in the text (e.g. `conf.md`), do not use inline links
  - End with a References list that collects links to all files mentioned in the approach

Example:

```markdown
# How: Add user authentication

## Approach

- Use existing middleware pipeline in `src/app.ts` to add authentication layer
- Add `middleware/auth.middleware.ts` leveraging OAuth 2.0 library for token handling
- Store sessions in Redis using `config/redis.config.ts` for horizontal scalability

References:

- [src/app.ts](../../src/app.ts)
- [middleware/auth.middleware.ts](../../src/middleware/auth.middleware.ts)
- [config/redis.config.ts](../../src/config/redis.config.ts)
```

## Further iterations (How File exists)

AI Agent decides which key elements to add based on existing How File content and Change Request context. Use format from `{templates_folder}/tmpl-td.md`. Do not be exhaustive - cover only what matters most.
