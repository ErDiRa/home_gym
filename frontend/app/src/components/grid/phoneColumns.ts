import type { ColDef } from 'ag-grid-community'

// A list's columns for the screen: on a phone only the ones in keep (by colId or field), so a row
// fits the width; the others stay defined and come back on a wider screen.
export function phoneColumns<T>(columns: ColDef<T>[], phone: boolean, keep: string[]): ColDef<T>[] {
  return columns.map((c) => ({ ...c, hide: phone && !keep.includes(c.colId ?? String(c.field)) }))
}

// Rows on a phone are tall enough to tap (44px and a bit); undefined keeps AG Grid's default.
export const phoneRowHeight = (phone: boolean) => (phone ? 48 : undefined)
