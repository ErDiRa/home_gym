import Fuse, { type Expression } from 'fuse.js'

// A field a fuzzy search looks at; a higher weight ranks matches in it higher.
export interface FuzzyKey<T> {
  name: string
  weight: number
  get: (item: T) => string | string[]
}

// Words shorter than this are skipped, so a single typed letter doesn't match almost everything.
const minWordLength = 2

// A fuzzy search over the given fields: every word of the query has to match one of them (a small
// typo is allowed, accents are ignored), the results come best match first. An empty query
// returns the items unchanged. The index is built once per items array, so typing doesn't
// rebuild it, while a reloaded list (a new array) does.
export function fuzzySearch<T>(keys: FuzzyKey<T>[]): (items: T[], query: string) => T[] {
  let indexed: T[] | null = null
  let fuse: Fuse<T> | null = null
  return (items, query) => {
    const words = query.split(/\s+/).filter((w) => w.length >= minWordLength)
    if (!words.length) return items
    if (!fuse || indexed !== items) {
      fuse = new Fuse(items, {
        keys: keys.map((k) => ({ name: k.name, weight: k.weight, getFn: k.get })),
        ignoreLocation: true, // a match anywhere in the text, not only near its start
        ignoreDiacritics: true,
        threshold: 0.4, // a typo or two swapped letters pass ("sqaut"), much more doesn't
      })
      indexed = items
    }
    const expression: Expression = {
      $and: words.map((w) => ({ $or: keys.map((k) => ({ [k.name]: w })) })),
    }
    return fuse.search(expression).map((r) => r.item)
  }
}
