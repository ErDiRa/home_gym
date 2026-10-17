import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  createFocus as apiCreateFocus,
  createLoadUnit,
  createRepUnit,
  listBlockKinds,
  listFocuses,
  listJoints,
  listLoadUnits,
  listMuscleGroups,
  listRepUnits,
} from '../api'
import type { BlockKind, Focus, Option, Unit } from '../types'

export type AmountPart = 'reps' | 'load'

// The lookup lists forms pick from: joints, muscle groups and focuses for exercises, block
// kinds and units for trainings.
export const useLookupsStore = defineStore('lookups', () => {
  const joints = ref<Option[]>([])
  const muscleGroups = ref<Option[]>([])
  const focuses = ref<Focus[]>([])
  const blockKinds = ref<BlockKind[]>([])
  const repUnits = ref<Unit[]>([])
  const loadUnits = ref<Unit[]>([])
  // The load unit picked last, used as the default for the next load (coaches tend to plan a
  // whole training in kg or in % 1RM).
  const lastLoadUnit = ref<string | null>(null)

  async function loadExerciseLookups() {
    // The leading semicolon keeps a line starting with [ from joining the previous statement.
    ;[joints.value, muscleGroups.value, focuses.value] = await Promise.all([
      listJoints(),
      listMuscleGroups(),
      listFocuses(),
    ])
  }

  async function loadTrainingLookups() {
    ;[blockKinds.value, repUnits.value, loadUnits.value] = await Promise.all([
      listBlockKinds(),
      listRepUnits(),
      listLoadUnits(),
    ])
  }

  // Creates a focus and adds it to the list, kept in name order.
  async function createFocus(name: string, alias: string): Promise<Focus> {
    const focus = await apiCreateFocus(name, alias)
    focuses.value = [...focuses.value, focus].sort((a, b) => a.name.localeCompare(b.name))
    return focus
  }

  // The kind with the given code; undefined until the lookups are loaded.
  const kindOf = (code: string) => blockKinds.value.find((k) => k.code === code)

  // The unit list for reps or load.
  const unitsOf = (part: AmountPart) => (part === 'reps' ? repUnits.value : loadUnits.value)

  // Creates a unit and adds it to the list, so every picker offers it right away.
  async function addUnit(part: AmountPart, name: string): Promise<Unit> {
    const unit = await (part === 'reps' ? createRepUnit(name) : createLoadUnit(name))
    unitsOf(part).push(unit)
    return unit
  }

  // The unit a new value starts in: reps for reps, the last load unit for load.
  function defaultUnit(part: AmountPart): string | null {
    if (part === 'load' && lastLoadUnit.value && loadUnits.value.some((u) => u.id === lastLoadUnit.value))
      return lastLoadUnit.value
    return unitsOf(part)[0]?.id ?? null
  }

  return {
    joints,
    muscleGroups,
    focuses,
    blockKinds,
    repUnits,
    loadUnits,
    lastLoadUnit,
    loadExerciseLookups,
    loadTrainingLookups,
    createFocus,
    kindOf,
    unitsOf,
    addUnit,
    defaultUnit,
  }
})
