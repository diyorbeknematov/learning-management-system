import { ExternalLink, Play } from 'lucide-react'
import { useState } from 'react'

type Source = { kind: 'embed'; embed: string; poster?: string } | { kind: 'file'; url: string } | { kind: 'link'; url: string }

function parse(url: string): Source {
  try {
    const link = new URL(url)
    const host = link.hostname.replace(/^www\./, '')

    let youtube: string | null = null

    if (host === 'youtu.be') youtube = link.pathname.slice(1)
    else if (host.endsWith('youtube.com')) youtube = link.searchParams.get('v') ?? link.pathname.match(/\/(?:embed|shorts)\/([\w-]+)/)?.[1] ?? null

    if (youtube) {
      return {
        kind: 'embed',
        embed: `https://www.youtube-nocookie.com/embed/${youtube}?autoplay=1&rel=0`,
        poster: `https://img.youtube.com/vi/${youtube}/hqdefault.jpg`,
      }
    }

    const vimeo = host.endsWith('vimeo.com') ? link.pathname.match(/(\d+)/)?.[1] : null

    if (vimeo) return { kind: 'embed', embed: `https://player.vimeo.com/video/${vimeo}?autoplay=1` }

    if (/\.(mp4|webm|ogg|mov)$/i.test(link.pathname)) return { kind: 'file', url }
  } catch {
    // not a valid address: shown as a plain link
  }

  return { kind: 'link', url }
}

/** A video as on YouTube: a picture with a play button; a click starts it right here. */
export function VideoPlayer({ url, title }: { url: string; title?: string }) {
  const [playing, setPlaying] = useState(false)
  const source = parse(url)

  if (source.kind === 'file') {
    return <video src={source.url} controls preload="metadata" className="aspect-video w-full max-w-3xl rounded-xl bg-black" aria-label={title ?? 'Video'} />
  }

  if (source.kind === 'link') {
    return (
      <a
        href={source.url}
        target="_blank"
        rel="noreferrer"
        className="inline-flex items-center gap-2 font-medium text-primary underline-offset-4 hover:underline"
      >
        <ExternalLink className="size-4" /> Open the video in a new tab
      </a>
    )
  }

  if (playing) {
    return (
      <iframe
        src={source.embed}
        title={title ?? 'Video'}
        className="aspect-video w-full max-w-3xl rounded-xl bg-black"
        allow="accelerometer; autoplay; encrypted-media; picture-in-picture; fullscreen"
        allowFullScreen
      />
    )
  }

  return (
    <button
      type="button"
      onClick={() => setPlaying(true)}
      aria-label={`Play ${title ?? 'the video'}`}
      className="group relative flex aspect-video w-full max-w-3xl items-center justify-center overflow-hidden rounded-xl bg-slate-900 outline-none focus-visible:ring-3 focus-visible:ring-ring/50"
    >
      {source.poster && <img src={source.poster} alt="" className="absolute inset-0 size-full object-cover transition group-hover:scale-[1.02]" />}
      <span className="absolute inset-0 bg-black/20 transition group-hover:bg-black/30" />
      <span className="relative flex size-16 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-xl transition group-hover:scale-110">
        <Play className="size-7 translate-x-0.5 fill-current" />
      </span>
    </button>
  )
}
