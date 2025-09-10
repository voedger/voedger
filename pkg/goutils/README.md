# goutils
Golang utilities

## AI Prompt to Create a Brief README.md

````
You are an AI system specialized in analyzing Go packages and generating concise developer documentation. Your task is to **write a brief `README.md` file** for the specified package. Follow the instructions **strictly** and do not add extra sections, filler text, or assumptions beyond what can be inferred from the package code and tests.

## Requirements

1. **Package Name**
   Write a concise (1–2 sentences) description explaining what the package does and its primary purpose. Focus on the conceptual role, not implementation details.

2. **Problem Statement ("Why")**
   Provide a short (1–2 sentences) explanation of the problem this package solves or why it exists. Make sure this is conceptually distinct from the description (avoid repeating wording).

3. **Features Section**
   Use bullet points to list the fundamental capabilities (not implementation details).

   * Each feature line ≤72 characters
   * Each feature: **2–3 word name** + short description
   * No trailing periods

4. **Platform-Specific Logic**
   If the code has platform-specific behavior (e.g. build tags, OS-specific imports), add a `## Platform Support` section describing it.

   * If none exists, omit this section
   * Never phrase things as "unsupported" or "missing"

5. **Basic Usage Link**
   If a `*_test.go` file has a function like `TestBasicUsage`, `TestUsage`, or `ExampleBasic`, add a `## Basic Usage` section with a markdown link to that test file.

6. **Line Length**
   All lines must wrap at ≤72 characters.

7. **README Structure**
   Follow this exact markdown format:

   ```markdown
   # Name

   Brief description.

   ## Problem

   Brief explanation.

   ## Features

   - **Feature name** - Short description
   - **Feature name** - Short description
   - **Feature name** - Short description

   ## Platform Support

   Notes about platform-specific behavior (if applicable).

   ## Basic Usage

   See [basic usage example](path/to/test_file.go)
   ```

8. **Style Guidelines**

   * Use sentence capitalization for headings and list items
   * No periods at end of list items
   * Add empty lines before lists
   * Keep text concise and conceptual
   * Prioritize developer-facing value over internal details

## AI-Specific Instructions

* **Analyze before writing**: Inspect exported types/functions, tests, and build tags to infer purpose, features, and usage.
* **Avoid repetition**: Ensure description and problem statement are conceptually distinct.
* **Do not invent**: Only include features/usage/platform notes if clearly present.
* **Optimize for clarity**: Developers should quickly understand why the package exists and what it offers.
* **Strict format compliance**: Produce only the README.md content in the requested structure, no extra commentary.

````
