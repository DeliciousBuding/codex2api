import { useEffect, useState } from 'react'
import { createHighlighter, type Highlighter } from 'shiki'

let highlighterPromise: Promise<Highlighter> | null = null

function getHighlighter() {
  if (!highlighterPromise) {
    highlighterPromise = createHighlighter({
      themes: ['github-light-default', 'dark-plus'],
      langs: ['json', 'toml', 'shellscript', 'python'],
    })
  }
  return highlighterPromise
}

function tuneLightTheme(html: string) {
  return html
    .replace(/#0550AE/gi, '#075985')
    .replace(/#0969DA/gi, '#075985')
    .replace(/#1F2328/gi, '#1f2937')
    .replace(/#953800/gi, '#9a3412')
    .replace(/#0A3069/gi, '#7c2d12')
    .replace(/#CF222E/gi, '#b91c1c')
}

export function useHighlightedHtml(code: string, lang?: string) {
  const [html, setHtml] = useState('')

  useEffect(() => {
    let cancelled = false
    const resolvedLang = lang === 'bash' || lang === 'shell' || lang === 'curl' ? 'shellscript' : (lang || 'text')

    getHighlighter().then((hl) => {
      if (cancelled) return
      try {
        const isDark = document.documentElement.classList.contains('dark')
        const result = hl.codeToHtml(code, {
          lang: resolvedLang,
          theme: isDark ? 'dark-plus' : 'github-light-default',
        })
        setHtml(isDark ? result : tuneLightTheme(result))
      } catch {
        setHtml('')
      }
    })

    return () => { cancelled = true }
  }, [code, lang])

  return html
}
