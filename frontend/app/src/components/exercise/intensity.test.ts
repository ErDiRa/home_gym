import { describe, expect, it } from 'vitest'
import { cleanIntensity, formatIntensity, intensityProblem, intensityText, parseIntensity } from './intensity'

describe('parseIntensity', () => {
  it.each([
    ['1:55', 115],
    ['1.55', 115],
    ['1,55', 115],
    ['1:55.3', 115.3],
    ['2', 120],
    [' 0:59 ', 59],
  ])('reads the pace %s', (text, seconds) => {
    expect(parseIntensity(text, 'pace_500m')).toBe(seconds)
  })

  it.each(['', '1:75', 'fast', '1:5:5'])('rejects the pace "%s"', (text) => {
    expect(parseIntensity(text, 'pace_500m')).toBeNull()
  })

  it('reads watts and cal/h as numbers', () => {
    expect(parseIntensity('231', 'watts')).toBe(231)
    expect(parseIntensity('1250,5', 'cal_per_hour')).toBe(1250.5)
    expect(parseIntensity('1:55', 'watts')).toBeNull()
  })
})

describe('formatIntensity', () => {
  it('shows paces as m:ss with tenths and others as numbers', () => {
    expect(formatIntensity('pace_500m', 115, 120)).toBe('1:55–2:00 /500 m')
    expect(formatIntensity('pace_1000m', 225.4)).toBe('3:45.4 /1000 m')
    expect(formatIntensity('pace_500m', 65)).toBe('1:05 /500 m')
    expect(formatIntensity('watts', 220, 240)).toBe('220–240 W')
  })

  it('writes the planned target as a phrase', () => {
    const target = { intensity_metric: 'watts' as const, intensity_value: 220, intensity_max: null, recovery_value: 100, recovery_max: 120 }
    expect(intensityText(target)).toBe('@ 220 W · recovery 100–120 W')
  })
})

describe('the planned target', () => {
  const none = { intensity_metric: null, intensity_value: null, intensity_max: null, recovery_value: null, recovery_max: null }

  it('needs a metric and rising ranges', () => {
    expect(intensityProblem(none)).toBe('')
    expect(intensityProblem({ ...none, intensity_value: 220 })).toBe('Choose watts, cal/h or a pace for the target.')
    expect(intensityProblem({ ...none, intensity_metric: 'pace_500m', recovery_value: 150, recovery_max: 140 })).toBe(
      'The upper end of the recovery range must be higher than 2:30.',
    )
  })

  it('drops a metric without values and a lone max', () => {
    expect(cleanIntensity({ ...none, intensity_metric: 'watts', intensity_max: 240 })).toEqual(none)
  })
})
