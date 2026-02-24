# Template: Pull request

## PR title and body

Derived from the specs diff. `draft_title` is a concise summary of changes; `draft_body` is a concise description in max three phrases.

With issue reference:

```text
pr_title: [{issue_id}] {draft_title}
pr_body:  [{issue_id}]({issue_url}) {draft_title}

         {draft_body}
```

Without issue reference:

```text
pr_title: {draft_title}
pr_body:  {draft_title}

         {draft_body}
```
