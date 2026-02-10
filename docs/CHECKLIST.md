# SEO & LLM Optimization Checklist

Use this checklist to verify your static site is optimized for both search engines and AI/LLM indexing.

---

## HTML Head Tags

- [ ] `<title>` - Unique, descriptive, under 60 characters
- [ ] `<meta name="description">` - Compelling, 150-160 characters
- [ ] `<meta name="viewport">` - Mobile responsive
- [ ] `<meta charset="UTF-8">` - Character encoding
- [ ] `<link rel="canonical">` - Canonical URL to prevent duplicates
- [ ] `<html lang="en">` - Language attribute

## Open Graph (Social Sharing)

- [ ] `og:title` - Page title for social
- [ ] `og:description` - Description for social
- [ ] `og:image` - Image URL (1200x630px recommended)
- [ ] `og:url` - Canonical URL
- [ ] `og:type` - Content type (website, article, etc.)
- [ ] `og:site_name` - Site name

## Twitter/X Cards

- [ ] `twitter:card` - Card type (summary_large_image)
- [ ] `twitter:title` - Title (can fallback to og:title)
- [ ] `twitter:description` - Description
- [ ] `twitter:image` - Image URL

## Structured Data (JSON-LD)

- [ ] Schema.org `@context` included
- [ ] Appropriate `@type` for content (SoftwareApplication, WebPage, etc.)
- [ ] All required properties for chosen type
- [ ] Validated with Google Rich Results Test
- [ ] No syntax errors in JSON

## Content Structure

- [ ] Single H1 tag per page
- [ ] Hierarchical heading structure (H1 > H2 > H3)
- [ ] Descriptive headings with keywords
- [ ] Short, focused paragraphs
- [ ] Bullet/numbered lists for scannable content
- [ ] Internal links where relevant

## Accessibility (a11y)

- [ ] Semantic HTML elements (`<nav>`, `<main>`, `<article>`, `<section>`, `<footer>`)
- [ ] Alt text on all images
- [ ] Color contrast ratio 4.5:1 minimum
- [ ] Keyboard navigable (tab through all interactive elements)
- [ ] Focus indicators visible
- [ ] Skip link to main content
- [ ] ARIA labels on non-semantic elements

## Performance

- [ ] Minimal CSS (inline critical CSS if needed)
- [ ] No unnecessary JavaScript
- [ ] Optimized images (WebP, compressed)
- [ ] No render-blocking resources
- [ ] Fast load time (< 3 seconds)

## LLM Optimization

- [ ] `/llms.txt` file at root
- [ ] Clean, readable content structure
- [ ] Direct answers to likely questions
- [ ] Content accessible without JavaScript
- [ ] robots.txt allows AI crawlers

## Files to Create

- [ ] `index.html` - Main page
- [ ] `llms.txt` - LLM crawler guidance
- [ ] `robots.txt` - Crawler permissions
- [ ] `sitemap.xml` - Page listing for crawlers
- [ ] `styles.css` - Stylesheet (optional, can be inline)
- [ ] Social share image (og:image)

## GitHub Pages Specific

- [ ] Repository is public (for github.io hosting)
- [ ] GitHub Pages enabled in repo settings
- [ ] Custom domain configured (optional)
- [ ] HTTPS enforced

## Testing Tools

- [ ] [Google Rich Results Test](https://search.google.com/test/rich-results)
- [ ] [Schema.org Validator](https://validator.schema.org/)
- [ ] [Google PageSpeed Insights](https://pagespeed.web.dev/)
- [ ] [WAVE Accessibility](https://wave.webaim.org/)
- [ ] [Twitter Card Validator](https://cards-dev.twitter.com/validator)
- [ ] [Facebook Sharing Debugger](https://developers.facebook.com/tools/debug/)

## Pre-Launch

- [ ] All links work (no 404s)
- [ ] Spelling and grammar checked
- [ ] Mobile view tested
- [ ] Multiple browsers tested
- [ ] Print stylesheet (optional)
- [ ] Favicon included

---

## Quick Reference: File Locations

```
/
├── index.html          # Main page
├── llms.txt            # LLM guidance (root)
├── robots.txt          # Crawler rules (root)
├── sitemap.xml         # Sitemap (root)
├── styles.css          # Stylesheet
└── assets/
    └── og-image.png    # Social share image
```
