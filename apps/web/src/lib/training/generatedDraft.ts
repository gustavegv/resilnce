import type { ExerciseSuggestion } from './types';

// Both producers hand off untrusted suggestions, never ready-to-save intent.
export function readGeneratedDraft(text: string, catalog?: Set<string>): ExerciseSuggestion[] {
  let value: unknown;
  try {
    value = JSON.parse(text);
  } catch {
    throw new Error(
      'The generated session could not be read. Generate a new session or start manually.'
    );
  }
  if (!Array.isArray(value) || value.length < 1 || value.length > 100)
    throw new Error(
      'The generated session needs 1–100 exercises. Generate a new session or start manually.'
    );
  return value.map((item, index) => {
    const fail = () =>
      new Error(
        `Suggestion ${index + 1}: the suggested exercise has incomplete or invalid training settings. Generate a new session or start manually.`
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
      throw new Error(
        `Suggestion ${index + 1}: the suggested exercise contains unsupported information. Generate a new session or start manually.`
      );
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
        `Suggestion ${index + 1}: the suggested muscle groups could not be used. Generate a new session or start manually.`
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
