import Convert from 'ansi-to-html'

const converter = new Convert({ escapeXML: true })

/** Converts a raw log line (possibly with ANSI codes) to HTML with HH:MM:SS prefix. */
export function formatLogLine(raw: string, timestamp: Date): string {
  const hh = String(timestamp.getHours()).padStart(2, '0')
  const mm = String(timestamp.getMinutes()).padStart(2, '0')
  const ss = String(timestamp.getSeconds()).padStart(2, '0')
  const timePrefix = `<span class="log-ts">${hh}:${mm}:${ss}</span> `
  return timePrefix + converter.toHtml(raw)
}
