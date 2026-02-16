package main

// draftPrompt generates the initial article draft
const draftPrompt = `You are a professional content writer.

Task: Write a compelling article on the following topic.

Topic: {{input.topic}}
Style: {{input.style}}
Target Length: {{input.target_length}} words
Audience: {{input.audience}}

Requirements:
1. Clear structure with introduction, body, and conclusion
2. Engaging opening paragraph
3. Use relevant examples and data
4. Professional tone appropriate for the audience
5. Aim for approximately {{input.target_length}} words

Output the article in plain text with clear section headings.`

// reviewPrompt analyzes content quality and returns a structured JSON score.
// IMPORTANT: The review must echo the article back in its JSON output so that
// downstream nodes (improve, format) can access both the score AND the article.
// This is necessary because the LLM executor wraps output in {"content": "..."}
// which overwrites the original article content field.
const reviewPrompt = `You are a strict content quality reviewer.

Task: Review the following article and provide a quality score with feedback.

Article:
{{input.content}}

Evaluate on these criteria:
1. Clarity and coherence (0-25)
2. Depth and accuracy (0-25)
3. Structure and flow (0-25)
4. Engagement and readability (0-25)

IMPORTANT: You MUST respond with valid JSON in exactly this format:
{
  "score": <total score 0-100>,
  "issues": ["<issue1>", "<issue2>"],
  "strengths": ["<strength1>", "<strength2>"],
  "article": "<copy the FULL article text here exactly as provided above>"
}

Be honest and critical. Only scores above 80 indicate publication-ready quality.
You MUST include the full article text in the "article" field.`

// improvePrompt fixes identified issues in the content.
// Input comes from parse_review which provides: article, score, issues.
const improvePrompt = `You are a professional editor improving content based on reviewer feedback.

Task: Improve the article by addressing the identified issues.

Current Article:
{{input.article}}

Review Score: {{input.score}}/100
Issues Found: {{input.issues}}

Instructions:
1. Fix all issues mentioned in the review
2. Maintain the same general structure and length
3. Improve overall quality to achieve a score above 80
4. Keep the same topic and audience focus

Output the improved article in plain text with clear section headings.`

// seoPrompt generates SEO metadata for the final article
const seoPrompt = `You are an SEO specialist.

Task: Generate SEO metadata for this article.

Article:
{{input.content}}

Respond in JSON format:
{
  "title": "<SEO title, max 60 chars>",
  "description": "<meta description, max 160 chars>",
  "keywords": ["<keyword1>", "<keyword2>", "<keyword3>"],
  "slug": "<url-friendly-slug>"
}`
