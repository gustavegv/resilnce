export const promptStandard = `
    {
    "standard_prompt": {
        "description": "Your task is to take the user's unformatted workout session text and convert it into a clean, standardized JSON structure. 
        The JSON must contain the workout sessions's name and a list of exercises. Each exercise should follow the ExerciseInfo structure, 
        including name, muscle group (if provided), currentProgress, and optional autoIncrease. 
        
        Inside currentProgress, make sure to include:
        - sets: number of sets
        - repsPerSet: an array with the rep count for each set
        - weightPerSet: an array with the weight for each set (usually the same for each rep)
        - restSeconds: use -1 if the rest time is not provided
        
        If any exercise name is missing, leave it as an empty string instead of guessing.
        If the number of sets or reps is missing, default to 3 sets of 7 reps.
        If weight data is missing, set weightPerSet to [0, 0, 0] for 3 sets by default, UNLESS another "auto choose" prompt says otherwise.
        If rest time is not given, set restSeconds to -1.
        If muscleGroup is not mentioned, autofill it with the correct muscle group.
        If autoIncrease is not mentioned, simply omit that fields.

        The response must contain ONLY valid JSON.
        Do NOT include explanations, comments, or any additional text outside of the JSON object.",
        "expected_json_format": {
        "session_name": "string",
        "exercises": [
            {
            "name": "string",
            "muscleGroup": "string (optional)",
            "currentProgress": {
                "sets": "integer",
                "repsPerSet": ["integer", "integer", "..."],
                "weightPerSet": ["integer", "integer", "..."],
                "restSeconds": "integer"
            },
            "autoIncrease": "number (default to 2.5kg increase if not specified otherwise)"
            }
        ]
        },
        "example_input": "Push Day.\nBench press (70kg): reps (7,7,6)\nShoulder Press 30kg: reps8,9,10\nTricep Kickback 6kg reps:15,15,12",
        "example_output": {
        "day_name": "Push Day",
        "exercises": [
            {
            "name": "Bench Press",
            "currentProgress": {
                "sets": 3,
                "repsPerSet": [7, 7, 6],
                "weightPerSet": [70, 70, 70],
                "restSeconds": -1
            }
            },
            {
            "name": "Shoulder Press",
            "currentProgress": {
                "sets": 3,
                "repsPerSet": [8, 9, 10],
                "weightPerSet": [30, 30, 30],
                "restSeconds": -1
            }
            },
            {
            "name": "Tricep Kickback",
            "currentProgress": {
                "sets": 3,
                "repsPerSet": [15, 15, 12],
                "weightPerSet": [6, 6, 6],
                "restSeconds": -1
            }
            }
        ]
        }
    }
    }

    `;

export const promptSpell = `
    "spellcheck_exercises": {
        "description": "Before parsing and formatting the workout session, carefully spellcheck and normalize all exercise names.",
        "instructions": [
        "If the name is obviously misspelled or incomplete, correct it to the closest valid exercise name.",
        "Do not change the type of exercise — only fix spelling and casing.",
        "Use commonly accepted fitness terminology where possible.",
        "Preserve the original structure of sets, reps, and weights unless another prompt says otherwise."
        ],
        "example_input": "Push Day.\nBench press (70kg): reps (7,7,6)\nShoulderpres 30kg: reps8,9,10\nBicep curlss 12kg reps:10,10,9",
        "example_behavior": [
        "'Shoulderpres' → corrected to 'Shoulder Press'",
        "'Bicep curlss' → corrected to 'Bicep Curls'"
        ]
    },`;

export const promptRestart = `
    "restart_with_lighter_load": {
        "description": "If the user is returning from an injury or a long break, reduce all weights by 20% for a lighter starting load.",
        "instructions": [
        "Reduce all weights by 20% of the original value.",
        "Always round down to the nearest 2.5kg increment for realism.",
        "If no weight is specified, leave it blank.",
        "Reset all reps to 7. Do not alter set counts"
        ],
        "example_input": "Push Day.\nBench press (70kg): reps (9,9,8)\nShoulder Press 30kg: reps12,9,10\nLat pull dwn 55kg reps:10,9,8",
        "example_calculation": [
        "Bench Press: 70kg → 55kg (20% lighter, rounded to nearest 2.5kg), all reps set to 7",
        "Shoulder Press: 30kg → 25kg, all reps set to 7",
        "Lat Pull Down: 55kg → 45kg, all reps set to 7"
        ]
    },`;

export const promptAutoChoose = `
    "auto_choose_reps_and_weights": {
        "description": "Ignore the weights and reps provided in the user's input. Automatically assign new starting weights and reps based on a moderate difficulty program.",
        "instructions": [
        "Ignore all provided weights and reps.",
        "Use 3 sets by default unless the exercise implies otherwise.",
        "Assign 7-12 reps per set for hypertrophy-friendly programming.",
        "Choose weights appropriate for a lifter with ~6 months of training experience:",
        "  • For compound lifts (e.g., bench press, squat, deadlift, overhead press, barbell row) → set weights around 70% of an average intermediate working load.",
        "  • For accessory lifts (e.g., shoulder raises, tricep pushdowns, curls) → set moderate weights that allow completing the rep range with good form.",
        "  • For bodyweight or cardio-based exercises, keep them unweighted unless commonly loaded (e.g., weighted dips).",
        "Always output clean, standardized weights in kg."
        ],
        "example_input": "Push Day.\nBench press (70kg): reps (7,7,6)\nShoulder Press 30kg: reps8,9,10\nTricep kickbakc 6kg reps:15,15,12",
        "example_behavior": [
        "Bench Press → 50kg x 10 reps x 3 sets",
        "Shoulder Press → 20kg x 10 reps x 3 sets",
        "Tricep Kickback → 8kg x 12 reps x 3 sets"
        ]
    }
    `;
