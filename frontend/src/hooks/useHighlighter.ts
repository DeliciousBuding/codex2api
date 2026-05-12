import { useEffect, useState } from 'react'
import { createHighlighter, type Highlighter } from 'shiki'

let highlighterPromise: Promise<Highlighter> | null = null

function getHighlighter() {
  if (!highlighterPromise) {
    highlighterPromise = createHighlighter({
      themes: ['dark-plus'],
      langs: ['json', 'toml', 'shellscript', 'python'],
    })
  }
  return highlighterPromise
}

export function useHighlightedHtml(code: string, lang?: string) {
  const [html, setHtml] = useState('')

  useEffect(() => {
    let cancelled = false
    const resolvedLang = lang === 'bash' || lang === 'shell' || lang === 'curl' ? 'shellscript' : (lang || 'text')

    getHighlighter().then((hl) => {
      if (cancelled) return
      try {
        const result = hl.codeToHtml(code, {
          lang: resolvedLang,
          theme: 'dark-plus',
        })
        setHtml(result)
      } catch {
        setHtml('')
      }
    })

    return () => { cancelled = true }
  }, [code, lang])

  return html
}
