# SEO & LLM Indexing Best Practices Research

## Overview

This document outlines research on optimizing a static HTML site for both traditional search engines (SEO) and Large Language Model (LLM) indexing for AI-powered search and discovery.

---

## Part 1: Traditional SEO Best Practices

### 1.1 On-Page SEO Fundamentals

**Title Tags**
- Keep under 60 characters
- Include primary keyword near the beginning
- Make it descriptive and unique per page

**Meta Description**
- 150-160 characters
- Include primary and secondary keywords naturally
- Write compelling copy that encourages clicks
- Each page should have a unique description

**Heading Structure**
- One H1 per page (main title)
- Use H2-H6 in hierarchical order
- Include keywords in headings naturally
- Helps both users and crawlers understand content structure

**URL Structure**
- Keep URLs short and descriptive
- Use hyphens to separate words
- Include relevant keywords
- Avoid parameters and special characters

### 1.2 Technical SEO

**Core Web Vitals (Page Experience)**
- **LCP (Largest Contentful Paint)**: < 2.5s - load main content fast
- **FID (First Input Delay)**: < 100ms - respond to interactions quickly
- **CLS (Cumulative Layout Shift)**: < 0.1 - avoid layout jumps

**Mobile Optimization**
- Responsive design is mandatory
- Mobile-first indexing is default
- Touch targets should be adequately sized (48x48px minimum)

**Performance**
- Minimize CSS/JS
- Optimize images (use modern formats like WebP)
- Enable compression
- Leverage browser caching

**Crawlability**
- Valid HTML5 markup
- robots.txt file at root
- XML sitemap
- Canonical URLs to avoid duplicate content

### 1.3 Content Best Practices

- Write for humans first, optimize for search second
- Answer questions clearly and directly
- Use natural language and conversational tone
- Include relevant keywords without stuffing
- Provide comprehensive coverage of the topic

### 1.4 GitHub Pages Advantages

- Inherits GitHub's high domain authority
- Free HTTPS/SSL certificates
- Fast CDN delivery
- Reliable uptime

---

## Part 2: Structured Data (Schema.org / JSON-LD)

### 2.1 Why JSON-LD?

- Google's recommended format for structured data
- Sits in `<script>` tag, separate from HTML
- Easy to maintain and validate
- Enables rich snippets in search results
- Can improve CTR by up to 30%

### 2.2 Relevant Schema Types for Software Projects

**SoftwareApplication**
```json
{
  "@context": "https://schema.org",
  "@type": "SoftwareApplication",
  "name": "App Name",
  "description": "Description",
  "applicationCategory": "GameApplication",
  "operatingSystem": "macOS, Linux, Windows",
  "offers": {
    "@type": "Offer",
    "price": "0",
    "priceCurrency": "USD"
  }
}
```

**WebPage**
```json
{
  "@context": "https://schema.org",
  "@type": "WebPage",
  "name": "Page Title",
  "description": "Page description"
}
```

**BreadcrumbList** - for navigation context

### 2.3 Testing Tools

- Google Rich Results Test: https://search.google.com/test/rich-results
- Schema.org Validator: https://validator.schema.org/

---

## Part 3: LLM Indexing & llms.txt

### 3.1 What is llms.txt?

A proposed standard (similar to robots.txt) that helps AI crawlers understand and index website content more efficiently. It acts as a "table of contents" for LLMs.

**Key benefits:**
- Reduces resource strain on LLMs during crawling
- Helps AI skip ads, navigation, and noise
- Provides clean, structured content in Markdown format
- Allows site owners to control what AI systems access

### 3.2 llms.txt Specification

**Location:** `/llms.txt` at domain root

**Required Format (Markdown):**

```markdown
# Project Name (H1 - Required)

> Brief summary of the project in a blockquote (Optional)

Additional paragraphs with detailed information (Optional)

## Section Name (H2 headers for file lists)

- [Page Name](URL): Description of what this page contains
- [Another Page](URL): Another description

## Optional

Links in this section can be omitted when context is limited:
- [Secondary Info](URL): Less critical information
```

### 3.3 Current Adoption Status

As of early 2026:
- No official confirmation that OpenAI, Anthropic, or Google crawlers use llms.txt
- However, adoption is growing and it's considered good future-proofing
- Low effort to implement, potential high reward

### 3.4 General LLM Optimization

**Content Structure**
- Clear, hierarchical heading structure
- Short, focused paragraphs
- Bullet points and lists for key information
- Direct answers to common questions

**Technical Considerations**
- Clean HTML without excessive JavaScript
- Content accessible without JS execution
- Fast-loading pages (LLMs have timeout limits)
- robots.txt should allow AI crawlers (GPTBot, ClaudeBot, etc.)

**AI Crawler User Agents to Allow:**
- `GPTBot` (OpenAI)
- `ClaudeBot` or `Claude-Web` (Anthropic)
- `Google-Extended` (Google AI)
- `PerplexityBot` (Perplexity)
- `Amazonbot` (Amazon)

---

## Part 4: Social Media Optimization

### 4.1 Open Graph Tags

Used by Facebook, LinkedIn, Discord, Slack, and many others.

```html
<meta property="og:title" content="Page Title">
<meta property="og:description" content="Page description">
<meta property="og:image" content="https://example.com/image.png">
<meta property="og:url" content="https://example.com/page">
<meta property="og:type" content="website">
<meta property="og:site_name" content="Site Name">
```

**Image Requirements:**
- Minimum 1200x630px for high-res displays
- 1.91:1 aspect ratio
- Under 8MB file size

### 4.2 Twitter/X Cards

```html
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:title" content="Page Title">
<meta name="twitter:description" content="Page description">
<meta name="twitter:image" content="https://example.com/image.png">
```

**Card Types:**
- `summary` - Small thumbnail
- `summary_large_image` - Large image card (recommended)

Note: Twitter falls back to OG tags if Twitter-specific tags are missing.

---

## Part 5: Accessibility (a11y)

Accessibility improves SEO and ensures all users can access content.

### 5.1 Key Requirements

- Semantic HTML elements (`<nav>`, `<main>`, `<article>`, `<section>`)
- Alt text for all images
- Sufficient color contrast (4.5:1 for normal text)
- Keyboard navigation support
- ARIA labels where needed
- Skip links for keyboard users
- Focus indicators

### 5.2 Tools

- WAVE: https://wave.webaim.org/
- Lighthouse (built into Chrome DevTools)
- axe DevTools extension

---

## Sources

### SEO
- [Mastering SEO for GitHub Pages](https://www.jekyllpad.com/blog/mastering-github-pages-seo-7)
- [The Ultimate Guide to GitHub SEO for 2025](https://www.infrasity.com/blog/github-seo)
- [10 Best SEO Practices in 2026](https://content-whale.com/blog/best-seo-practices-2026/)

### Structured Data
- [Schema.org](https://schema.org/)
- [Google Structured Data Documentation](https://developers.google.com/search/docs/appearance/structured-data/intro-structured-data)
- [JSON-LD Beginners Guide](https://salt.agency/blog/json-ld-structured-data-beginners-guide-for-seos/)

### LLM Indexing
- [llms.txt Specification](https://llmstxt.org/)
- [Meet llms.txt - Search Engine Land](https://searchengineland.com/llms-txt-proposed-standard-453676)
- [What Is llms.txt? 2026 Guide](https://www.bluehost.com/blog/what-is-llms-txt/)
- [AI Crawlers Complete Guide](https://www.qwairy.co/guides/complete-guide-to-robots-txt-and-llms-txt-for-ai-crawlers)

### Social Media
- [Ultimate Guide to Social Meta Tags](https://www.everywheremarketer.com/blog/ultimate-guide-to-social-meta-tags-open-graph-and-twitter-cards)
- [Twitter Cards Getting Started](https://developer.x.com/en/docs/x-for-websites/cards/guides/getting-started)
