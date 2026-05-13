import { useEffect, useState } from 'react'
import { createHighlighterCore, type HighlighterCore } from 'shiki/core'
import { createJavaScriptRegexEngine } from 'shiki/engine/javascript'
import darkPlus from 'shiki/themes/dark-plus.mjs'
import json from 'shiki/langs/json.mjs'
import python from 'shiki/langs/python.mjs'
import shellscript from 'shiki/langs/shellscript.mjs'
import toml from 'shiki/langs/toml.mjs'

let highlighterPromise: Promise<HighlighterCore> | null = null

function getHighlighter() {
  if (!highlighterPromise) {
    highlighterPromise = createHighlighterCore({
      themes: [darkPlus],
      langs: [json, toml, shellscript, python],
      engine: createJavaScriptRegexEngine(),
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
