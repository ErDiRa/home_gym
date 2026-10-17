export type VideoProvider = 'youtube' | 'instagram'

export interface VideoEmbed {
  provider: VideoProvider
  /** Safe iframe URL, built from the parsed video id and never from the raw input. */
  src: string
}

const YOUTUBE_HOSTS = new Set(['youtube.com', 'www.youtube.com', 'm.youtube.com', 'music.youtube.com'])
const INSTAGRAM_HOSTS = new Set(['instagram.com', 'www.instagram.com'])
// Strict id formats, so nothing but an id can end up in the iframe URL.
const YOUTUBE_ID = /^[\w-]{11}$/
const INSTAGRAM_CODE = /^[\w-]+$/

// Extracts the video id from youtu.be/ID, /watch?v=ID, /shorts/ID, /embed/ID and /live/ID links.
function youtubeId(url: URL): string | null {
  let id: string | null = null
  if (url.hostname === 'youtu.be') {
    id = url.pathname.split('/')[1] ?? null
  } else if (YOUTUBE_HOSTS.has(url.hostname)) {
    const [, kind, pathId] = url.pathname.split('/')
    if (kind === 'watch') id = url.searchParams.get('v')
    else if (kind === 'shorts' || kind === 'embed' || kind === 'live') id = pathId ?? null
  }
  return id && YOUTUBE_ID.test(id) ? id : null
}

// Extracts the post code and whether it embeds as a post (p) or a reel (reel, reels, tv).
function instagramCode(url: URL): { kind: 'p' | 'reel'; code: string } | null {
  if (!INSTAGRAM_HOSTS.has(url.hostname)) return null
  // /p/CODE, /reel/CODE, /reels/CODE, /tv/CODE, also /username/p/CODE
  const parts = url.pathname.split('/').filter(Boolean)
  const i = parts.findIndex((part) => ['p', 'reel', 'reels', 'tv'].includes(part))
  const code = i >= 0 ? parts[i + 1] : undefined
  if (!code || !INSTAGRAM_CODE.test(code)) return null
  return { kind: parts[i] === 'p' ? 'p' : 'reel', code }
}

/** Returns an embeddable version of a YouTube or Instagram video link, or null for anything else. */
export function videoEmbed(value: string): VideoEmbed | null {
  let url: URL
  try {
    url = new URL(value.trim())
  } catch {
    return null
  }
  if (url.protocol !== 'https:' && url.protocol !== 'http:') return null

  const id = youtubeId(url)
  if (id) return { provider: 'youtube', src: `https://www.youtube-nocookie.com/embed/${id}` } // no tracking cookies until played

  const post = instagramCode(url)
  if (post) return { provider: 'instagram', src: `https://www.instagram.com/${post.kind}/${post.code}/embed` }

  return null
}
