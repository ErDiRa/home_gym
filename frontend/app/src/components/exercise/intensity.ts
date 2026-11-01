import type { Intensity, IntensityMetric } from '../../types'

// The intensity of an endurance exercise (an erg, a run): average watts, calories per hour, or the
// pace per 500 m (rower, ski erg) or 1000 m (BikeErg), a pace in seconds. Values are kept as the
// machine shows them, never converted between metrics.

export const intensityMetrics: { id: IntensityMetric; name: string }[] = [
  { id: 'watts', name: 'W' },
  { id: 'cal_per_hour', name: 'cal/h' },
  { id: 'pace_500m', name: '/500 m' },
  { id: 'pace_1000m', name: '/1000 m' },
]

export const intensityName = (metric: IntensityMetric | null) => intensityMetrics.find((m) => m.id === metric)?.name ?? ''

export const isPace = (metric: IntensityMetric | null) => metric === 'pace_500m' || metric === 'pace_1000m'

// A value as shown: a pace as m:ss, with tenths as monitors show them (1:55.3); others as numbers.
export function formatIntensityValue(value: number, metric: IntensityMetric | null): string {
  if (!isPace(metric)) return `${value}`
  const tenths = Math.round(value * 10)
  const m = Math.floor(tenths / 600)
  const s = (tenths % 600) / 10
  const ss = s < 10 ? `0${s}` : `${s}`
  return `${m}:${ss}`
}

// A value or range with its metric, e.g. "1:55–2:00 /500 m" or "220 W".
export function formatIntensity(metric: IntensityMetric | null, value: number, max: number | null = null): string {
  const v = formatIntensityValue(value, metric)
  return `${max === null ? v : `${v}–${formatIntensityValue(max, metric)}`} ${intensityName(metric)}`
}

// Reads what was typed: a pace as m:ss, also with "." or "," for the colon (phones' number pads
// lack it), e.g. "1:55", "1.55", "1:55.3"; other metrics as a number. null when empty or unreadable.
export function parseIntensity(text: string, metric: IntensityMetric | null): number | null {
  const t = text.trim()
  if (!t) return null
  if (isPace(metric)) {
    const m = /^(\d+)[:.,](\d{1,2})(?:[.,](\d))?$/.exec(t)
    if (!m) return /^\d+$/.test(t) ? Number(t) * 60 : null // whole minutes, e.g. "2" = 2:00
    const seconds = Number(m[2])
    if (seconds > 59) return null
    return Number(m[1]) * 60 + seconds + (m[3] ? Number(m[3]) / 10 : 0)
  }
  const n = Number(t.replace(',', '.'))
  return Number.isFinite(n) ? n : null
}

export const emptyIntensity = (): Intensity => ({
  intensity_metric: null,
  intensity_value: null,
  intensity_max: null,
  recovery_value: null,
  recovery_max: null,
})

// Copies just the intensity fields.
export const pickIntensity = (i: Intensity): Intensity => ({
  intensity_metric: i.intensity_metric,
  intensity_value: i.intensity_value,
  intensity_max: i.intensity_max,
  recovery_value: i.recovery_value,
  recovery_max: i.recovery_max,
})

// The intensity as sent: a lone max or a metric without values is dropped.
export function cleanIntensity(i: Intensity): Intensity {
  const out = pickIntensity(i)
  if (out.intensity_value === null) out.intensity_max = null
  if (out.recovery_value === null) out.recovery_max = null
  if (out.intensity_value === null && out.recovery_value === null) out.intensity_metric = null
  return out
}

// The planned target as a phrase, e.g. "@ 1:55–2:00 /500 m · recovery 2:30 /500 m"; '' without one.
export function intensityText(i: Intensity): string {
  const parts: string[] = []
  if (i.intensity_value !== null) parts.push(`@ ${formatIntensity(i.intensity_metric, i.intensity_value, i.intensity_max)}`)
  if (i.recovery_value !== null) parts.push(`recovery ${formatIntensity(i.intensity_metric, i.recovery_value, i.recovery_max)}`)
  return parts.join(' · ')
}

// Checks a planned intensity like the server: a metric once there is a value, ranges rising.
export function intensityProblem(i: Intensity): string {
  if (i.intensity_value === null && i.recovery_value === null) return ''
  if (!i.intensity_metric) return 'Choose watts, cal/h or a pace for the target.'
  for (const [what, value, max] of [
    ['target', i.intensity_value, i.intensity_max],
    ['recovery', i.recovery_value, i.recovery_max],
  ] as const) {
    if (value !== null && value <= 0) return `The ${what} must be above 0.`
    if (value !== null && max !== null && max <= value) return `The upper end of the ${what} range must be higher than ${formatIntensityValue(value, i.intensity_metric)}.`
  }
  return ''
}
