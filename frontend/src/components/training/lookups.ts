import { ref } from 'vue'
import { createLoadUnit, createRepUnit, listBlockKinds, listLoadUnits, listRepUnits } from '../../api'
import type { BlockKind, Unit } from '../../types'

// The block kinds and units the training editor picks from, shared by its components.
// loadTrainingLookups fills them when the editor opens.
export const blockKinds = ref<BlockKind[]>([])
export const repUnits = ref<Unit[]>([])
export const loadUnits = ref<Unit[]>([])
// The load unit picked last, used as the default for the next load (coaches tend to plan a
// whole training in kg or in % 1RM).
export const lastLoadUnit = ref<string | null>(null)

export async function loadTrainingLookups() {
  ;[blockKinds.value, repUnits.value, loadUnits.value] = await Promise.all([
    listBlockKinds(),
    listRepUnits(),
    listLoadUnits(),
  ])
}

export const kindOf = (code: string) => blockKinds.value.find((k) => k.code === code)

export type AmountPart = 'reps' | 'load'

export const unitsOf = (part: AmountPart) => (part === 'reps' ? repUnits : loadUnits)

// Creates a unit and adds it to the list, so every picker offers it right away.
export async function addUnit(part: AmountPart, name: string): Promise<Unit> {
  const unit = await (part === 'reps' ? createRepUnit(name) : createLoadUnit(name))
  unitsOf(part).value.push(unit)
  return unit
}

// The unit a new value starts in: reps for reps, the last load unit for load.
export function defaultUnit(part: AmountPart): string | null {
  if (part === 'load' && lastLoadUnit.value && loadUnits.value.some((u) => u.id === lastLoadUnit.value))
    return lastLoadUnit.value
  return unitsOf(part).value[0]?.id ?? null
}
