import type { ExerciseSuggestion } from './types';

// Both producers hand off untrusted suggestions, never ready-to-save intent.
export function readGeneratedDraft(text: string, catalog?: Set<string>): ExerciseSuggestion[] {
  const value: unknown = JSON.parse(text);
  if (!Array.isArray(value) || value.length < 1 || value.length > 100)
    throw new Error('Use a draft with 1–100 exercises. Correct the draft below.');
  return value.map((item, index) => {
    const fail = () =>
      new Error(
        `Suggestion ${index + 1}: use a name, 1–20 sets, a weight from 0–9999, a threshold from 1–99 and an increment from 0.25–99. Correct the draft below.`
      );
    if (!item || typeof item !== 'object' || Array.isArray(item)) throw fail();
    if (
      Object.keys(item).some(
        (key) =>
          !['name', 'sets', 'weight', 'repThreshold', 'autoIncrease', 'muscleGroupCodes'].includes(
            key
          )
      )
    )
      throw new Error(`Suggestion ${index + 1}: unsupported fields. Correct the draft below.`);
    const { name, sets, weight, repThreshold = 12, autoIncrease = 2.5 } = item;
    if (
      typeof name !== 'string' ||
      !name.trim() ||
      [...name.trim()].length > 100 ||
      !Number.isInteger(sets) ||
      sets < 1 ||
      sets > 20 ||
      typeof weight !== 'number' ||
      !Number.isFinite(weight) ||
      weight < 0 ||
      weight > 9999 ||
      !/^\d+(?:\.\d{1,2})?$/.test(String(weight)) ||
      !Number.isInteger(repThreshold) ||
      repThreshold < 1 ||
      repThreshold > 99 ||
      typeof autoIncrease !== 'number' ||
      !Number.isFinite(autoIncrease) ||
      autoIncrease < 0.25 ||
      autoIncrease > 99 ||
      !/^\d+(?:\.\d{1,2})?$/.test(String(autoIncrease))
    )
      throw fail();
    const codes: unknown = item.muscleGroupCodes ?? [];
    if (
      ('muscleGroupCodes' in item && item.muscleGroupCodes === null) ||
      !Array.isArray(codes) ||
      codes.some((code) => typeof code !== 'string' || !code || (catalog && !catalog.has(code))) ||
      new Set(codes).size !== codes.length
    )
      throw new Error(
        `Suggestion ${index + 1}: use known, unrepeated muscle-group codes or an empty array. Correct the draft below.`
      );
    return {
      name: name.trim(),
      muscleGroupCodes: codes,
      configuration: {
        prescribedSets: sets,
        initialWeight: String(weight),
        repThreshold,
        autoIncrement: String(autoIncrease),
        restSeconds: 0,
      },
    };
  });
}
