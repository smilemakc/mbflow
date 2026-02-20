package main

// contentGenerationPrompt generates initial content based on topic and requirements
const contentGenerationPrompt = `You are a professional content writer creating high-quality content.

Task: Write comprehensive, engaging content on the following topic.

Topic: {{input.topic}}
Content Type: {{input.content_type}}
Target Length: {{input.target_length}} words
Tone: {{input.tone}}
Target Audience: {{input.target_audience}}

Requirements:
1. Write clear, well-structured content with proper headings and paragraphs
2. Include relevant examples and explanations where appropriate
3. Use professional language suitable for the target audience
4. Ensure accuracy and provide valuable insights
5. Make the content engaging and easy to read
6. Optimize for SEO with natural keyword integration
7. Aim for approximately {{input.target_length}} words

Output the content in plain text format, well-formatted with clear sections.`

// qualityAnalysisPrompt analyzes content quality and returns structured JSON
const qualityAnalysisPrompt = `You are an expert content quality analyst.

Task: Analyze the following content and provide a detailed quality assessment.

Content to analyze:
{{input.content}}

Evaluation criteria:
1. Grammar and spelling accuracy
2. Clarity and coherence
3. Depth and comprehensiveness
4. Engagement and readability
5. Structure and organization
6. Accuracy of information
7. SEO optimization potential

Provide your analysis in the following JSON format:
{
  "score": <integer from 0-100>,
  "issues": [<list of specific issues found>],
  "strengths": [<list of content strengths>],
  "recommendations": [<specific improvement recommendations>]
}

Scoring guidelines:
- 80-100: Excellent quality, ready for publication
- 50-79: Good quality but needs enhancement
- 0-49: Poor quality, requires regeneration

Be objective and thorough in your analysis.`

// enhancementPrompt improves content based on identified issues
const enhancementPrompt = `You are a professional content editor specializing in content improvement.

Task: Enhance the following content by addressing the identified issues while preserving the core message and structure.

Original Content:
{{input.generate.content}}

Quality Analysis:
{{input.analyze.content}}

Instructions:
1. Parse the quality analysis JSON to extract the issues and recommendations
2. Fix all grammar and spelling errors
3. Improve clarity and coherence where needed
4. Enhance depth by adding relevant details or examples
5. Improve structure and flow
6. Maintain the original tone and style
7. Keep approximately the same length
8. Preserve all key points and accurate information

Output the enhanced content in plain text format, maintaining proper formatting.`

// regenerationPrompt creates fresh content avoiding previous issues
const regenerationPrompt = `You are a professional content writer tasked with creating improved content.

Task: Generate fresh, high-quality content on the topic, avoiding the issues from the previous attempt.

Topic: {{input.topic}}
Content Type: {{input.content_type}}
Target Length: {{input.target_length}} words
Tone: {{input.tone}}
Target Audience: {{input.target_audience}}

Previous Content Analysis:
{{input.analyze.content}}

Instructions:
1. Parse the quality analysis JSON to understand what was wrong with the previous attempt
2. Take a completely different approach or angle than the previous attempt
3. Specifically avoid all issues mentioned in the analysis
4. Ensure high quality in grammar, structure, and content depth
5. Make it engaging and well-researched
6. Use clear, professional language
7. Aim for approximately {{input.target_length}} words

Output the content in plain text format with proper formatting.`

// translationPromptES translates content to Spanish
const translationPromptES = `You are a professional translator specializing in English to Spanish translation.

Task: Translate the following content to Spanish (Español) with native quality.

Original Content:
{{input.content}}

Translation Requirements:
1. Maintain the same tone, style, and structure
2. Ensure grammatical accuracy and natural flow
3. Adapt idioms and cultural references appropriately
4. Preserve technical terms accurately
5. Keep formatting (paragraphs, sections)
6. Aim for native Spanish speaker quality

Output only the translated content in plain text format.`

// translationPromptRU translates content to Russian
const translationPromptRU = `You are a professional translator specializing in English to Russian translation.

Task: Translate the following content to Russian (Русский) with native quality.

Original Content:
{{input.content}}

Translation Requirements:
1. Maintain the same tone, style, and structure
2. Ensure grammatical accuracy and natural flow
3. Adapt idioms and cultural references appropriately
4. Preserve technical terms accurately
5. Keep formatting (paragraphs, sections)
6. Aim for native Russian speaker quality

Output only the translated content in plain text format.`

// translationPromptDE translates content to German
const translationPromptDE = `You are a professional translator specializing in English to German translation.

Task: Translate the following content to German (Deutsch) with native quality.

Original Content:
{{input.content}}

Translation Requirements:
1. Maintain the same tone, style, and structure
2. Ensure grammatical accuracy and natural flow
3. Adapt idioms and cultural references appropriately
4. Preserve technical terms accurately
5. Keep formatting (paragraphs, sections)
6. Aim for native German speaker quality

Output only the translated content in plain text format.`

// seoGenerationPrompt generates SEO metadata for content
const seoGenerationPrompt = `You are an SEO expert specializing in search engine optimization.

Task: Generate SEO-optimized metadata for the following content.

Content:
{{input.content}}

Language: {{input.language}}

Generate SEO metadata in the following JSON format:
{
  "title": "<SEO-optimized title, max 60 characters>",
  "meta_description": "<Compelling meta description, max 160 characters>",
  "keywords": ["<keyword1>", "<keyword2>", ...],
  "slug": "<url-friendly-slug>"
}

Requirements:
1. Title: Compelling, includes primary keyword, max 60 characters
2. Meta Description: Persuasive summary, includes keywords, max 160 characters
3. Keywords: 5-10 relevant keywords/phrases for search optimization
4. Slug: URL-friendly (lowercase, hyphens, no special characters)

Focus on maximizing click-through rates and search visibility.`

// jqAggregationFilter combines all outputs into final structure
const jqAggregationFilter = `
# Parse SEO JSON strings into objects
(.seo_original.content | try fromjson catch {}) as $seo_orig |
(.seo_es.content | try fromjson catch {}) as $seo_es |
(.seo_ru.content | try fromjson catch {}) as $seo_ru |
(.seo_de.content | try fromjson catch {}) as $seo_de |

{
  original: {
    content: .merge.content,
    language: "English",
    quality_score: 0,
    seo: $seo_orig
  },
  translations: {
    spanish: {
      content: .trans_es.content,
      language: "Español",
      seo: $seo_es
    },
    russian: {
      content: .trans_ru.content,
      language: "Русский",
      seo: $seo_ru
    },
    german: {
      content: .trans_de.content,
      language: "Deutsch",
      seo: $seo_de
    }
  },
  metadata: {
    workflow_version: "1.0.0",
    generation_path: (if .enhance then "enhanced" elif .regenerate then "regenerated" else "direct" end)
  }
}`
