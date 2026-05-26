type Program = {
  id: string;
  slug: string;
  name: string;
  tagline: string;
  blockLength?: string;
  cadence: string;
  level: string;
  type: string;
  focus: string;
  tags: string[];
  accent: 'violet' | 'sky' | 'emerald' | 'amber';
  overview: string;
  howItRuns: string[];
  bestFor: string[];
  sessions?: string[];
  chartData: any;
};

export const hypertrophyPrograms: Program[] = [
  {
    id: 'upper-lower-booty',
    slug: 'upper-lower-booty',
    name: 'Booty Builder Upper Lower',
    tagline: 'Simple easy gains',
    cadence: '2-4 days / week',
    blockLength: '1 hour sessions',
    level: 'Beginner',
    type: 'Complete Program',
    focus: 'Hyperthrophy',
    tags: ['Basics', 'A/B sessions'],
    accent: 'violet',
    overview:
      'A simple upper/lower split for getting stronger in the lower body and building muscle without overthinking it. Rotate between two sessions, hit the basics hard, and train 2-4 days a week depending on your schedule. Easy to follow, beginner-friendly, and built around steady progress.',
    howItRuns: ['placeholder 2', 'placeholder 3', 'placeholder 4'],
    bestFor: ['Easy to follow program', 'Lifters who want a well-rounded program'],
    sessions: [
      'Booty Builder Upper A',
      'Booty Builder Upper B',
      'Booty Builder Lower A',
      'Booty Builder Lower B',
    ],
    chartData: [
      { muscleGroup: 'Quads', volume: 85.0 },
      { muscleGroup: 'Hamstrings', volume: 85.6 },
      { muscleGroup: 'Glutes', volume: 101.5 },
      { muscleGroup: 'Chest', volume: 25.5 },
      { muscleGroup: 'Back', volume: 76.8 },
      { muscleGroup: 'Triceps', volume: 67.5 },
      { muscleGroup: 'Biceps', volume: 52.5 },
      { muscleGroup: 'Shoulders', volume: 101.7 },
    ],
  },
  {
    id: 'compound-upper-lower',
    slug: 'compound-upper-lower',
    name: 'Quick Compound Upper Lower',
    tagline: 'Big lifts, balanced progress',
    cadence: '2-4 days / week',
    blockLength: '45 minute sessions',
    level: 'Intermediate',
    type: 'Complete Program',
    focus: 'Compound Hypertrophy',
    tags: ['Upper/Lower', 'Compound Lifts', 'Front Squat Bias', 'Quick'],
    accent: 'violet',
    overview:
      'A compound-focused upper/lower split built around pressing, pulling, front squats, hinges, single-leg work, carries, and direct core training. The program runs quick as it keeps most of the work on high-return exercises like bench, dips, rows, pull-ups, front squats, RDLs, Bulgarians, and belt squats, with just enough accessory work for arms, rear delts, and abs.',
    howItRuns: [
      'Rotate through Upper A, Lower A, Upper B, and Lower B, training 2-4 days per week depending on your schedule.',
      'Upper sessions focus on compound pressing and pulling with bench, dips, shoulder press, rows, pull-ups, and chin-ups.',
      'Lower sessions are front-squat biased, supported by RDLs, Bulgarians, belt squats, back extensions, carries, and core work.',
      'Accessory work is kept simple, with face pulls, biceps, leg raises, and ab wheel used to round out the main lifts.',
    ],
    bestFor: [
      'Lifters who want most of their training built around compound movements',
      'People who prefer front squats over back squats as their main squat pattern',
      'Anyone looking for a balanced upper/lower split with pressing, pulling, squatting, hinging, carries, and core work',
      'Beginners and early intermediates who are comfortable with basic barbell, machine, and bodyweight exercises',
    ],
    sessions: [
      'Quick Compound Upper A',
      'Quick Compound Upper B',
      'Quick Compound Lower A',
      'Quick Compound Lower B',
    ],
    chartData: [
      { muscleGroup: 'Quads', volume: 70.0 },
      { muscleGroup: 'Hamstrings', volume: 38.1 },
      { muscleGroup: 'Glutes', volume: 60.5 },
      { muscleGroup: 'Chest', volume: 48.5 },
      { muscleGroup: 'Back', volume: 93.6 },
      { muscleGroup: 'Triceps', volume: 55.6 },
      { muscleGroup: 'Biceps', volume: 66.2 },
      { muscleGroup: 'Shoulders', volume: 57.5 },
    ],
  },
  {
    id: 'balanced-upper-lower-hypertrophy',
    slug: 'balanced-upper-lower-hypertrophy',
    name: 'Balanced Upper Lower Hypertrophy',
    tagline: 'Even volume, complete development',
    cadence: '4 days / week',
    blockLength: '75 minute sessions',
    level: 'Intermediate',
    type: 'Complete Program',
    focus: 'Balanced Hypertrophy',
    tags: [
      'Upper/Lower',
      'Hypertrophy',
      'Balanced Volume',
      'Compound Lifts',
      'Accessories',
      '4 Day Split',
    ],
    accent: 'emerald',
    overview:
      'A balanced upper/lower hypertrophy split designed to bring every major muscle group close to its recommended weekly training target. The program combines squats, presses, rows, pulldowns, hinges, unilateral leg work, hip thrusts, curls, extensions, raises, and rear-delt work so volume is distributed evenly across quads, hamstrings, glutes, chest, back, arms, and shoulders.',
    howItRuns: [
      'Train four days per week using a Lower A, Upper A, Lower B, Upper B structure.',
      'Lower A builds the squat and hinge base with squats, leg press, RDLs, leg extensions, and hamstring curls.',
      'Upper A is push-biased, combining chest pressing, shoulder pressing, pulldowns, rows, lateral raises, and triceps work.',
      'Lower B adds glute and unilateral emphasis through front squats, Bulgarian split squats, hip thrusts, back extensions, and hip abduction.',
      'Upper B is pull-biased, using pulldowns, rows, chest work, rear delts, biceps, and triceps to round out the week.',
    ],
    bestFor: [
      'Lifters who want a complete four-day hypertrophy program with balanced weekly volume',
      'Intermediate trainees who want each major muscle group trained close to its ideal weekly set target',
      'People who like an upper/lower structure but want better balance than a heavily push-, pull-, or leg-biased split',
      'Anyone looking to combine compound lifts with enough accessory work to cover arms, delts, glutes, hamstrings, and chest properly',
    ],
    sessions: [
      'Balanced Lower A: Quad + hamstring',
      'Balanced Upper A: Push-biased upper',
      'Balanced Lower B: Glute + unilateral',
      'Balanced Upper B: Pull-biased',
    ],
    chartData: [
      { muscleGroup: 'Quads', volume: 107 },
      { muscleGroup: 'Hamstrings', volume: 104 },
      { muscleGroup: 'Glutes', volume: 104 },
      { muscleGroup: 'Chest', volume: 105 },
      { muscleGroup: 'Back', volume: 101 },
      { muscleGroup: 'Triceps', volume: 102 },
      { muscleGroup: 'Biceps', volume: 109 },
      { muscleGroup: 'Shoulders', volume: 103 },
    ],
  },
];

export const advancedPrograms: Program[] = [
  {
    id: '531',
    slug: '531',
    name: '5/3/1',
    tagline: 'Steady long-term strength progression',
    blockLength: '4-week block',
    cadence: '3-4 days / week',
    level: 'Beginner',
    type: 'Complete',
    focus: 'Main lifts + sustainable progress',
    tags: ['Strength', 'Classic'],
    accent: 'violet',
    overview:
      'A classic strength template built around squat, bench, deadlift, and press using conservative training maxes and repeatable monthly progression.',
    howItRuns: [
      'Each block runs across four weeks, with the main work progressing through 5s, 3s, and 5/3/1 style loading before a lighter reset or transition week.',
      'You train one main lift each session, then add supplemental work and a small amount of assistance based on your goal.',
      'The pace is intentionally sustainable, which makes it easy to run for multiple blocks without burning out.',
    ],
    bestFor: [
      'Building strength over months, not just one peak',
      'Lifters who want structure without excessive complexity',
    ],
    chartData: [
      { muscleGroup: 'Quads', volume: 82 },
      { muscleGroup: 'Glutes', volume: 76 },
      { muscleGroup: 'Chest', volume: 88 },
      { muscleGroup: 'Back', volume: 94 },
      { muscleGroup: 'Triceps', volume: 64 },
      { muscleGroup: 'Biceps', volume: 58 },
      { muscleGroup: 'Shoulders', volume: 72 },
    ],
  },
  {
    id: 'bbb',
    slug: 'boring-but-big',
    name: '5/3/1 Boring But Big',
    tagline: '5/3/1 with a big hypertrophy push',
    blockLength: '12-week block',
    cadence: '4 days / week',
    level: 'Intermediate',
    type: 'Complete',
    focus: 'Strength + size',
    tags: ['Hypertrophy', 'High volume', '5/3/1'],
    accent: 'sky',
    overview:
      'A higher-volume 5/3/1 variation that keeps the main strength work and adds 5 sets of 10 on supplemental lifts to drive muscle gain.',
    howItRuns: [
      'Each session starts with the regular 5/3/1 main lift work.',
      'After that, you perform 5x10 on the same lift or a paired lift depending on the setup you choose.',
      'Accessories stay simple so the extra volume can do the heavy lifting for growth and work capacity.',
    ],
    bestFor: [
      'Lifters who want size without abandoning strength work',
      'People who tolerate moderate to high barbell volume well',
    ],
    chartData: [
      { muscleGroup: 'Quads', volume: 82 },
      { muscleGroup: 'Glutes', volume: 76 },
      { muscleGroup: 'Chest', volume: 88 },
      { muscleGroup: 'Back', volume: 94 },
      { muscleGroup: 'Triceps', volume: 64 },
      { muscleGroup: 'Biceps', volume: 58 },
      { muscleGroup: 'Shoulders', volume: 72 },
    ],
  },
  {
    id: 'russian-squat',
    slug: 'russian-squat',
    name: 'Russian Strength Program',
    tagline: 'Short overreach cycle focused on one main lift',
    blockLength: '6-week block',
    cadence: '3 sessions / week',
    level: 'Intermediate',
    type: 'Partial',
    focus: 'Rapid peak for a chosen lift',
    tags: ['Strength', 'Specialization', 'Intensity'],
    accent: 'amber',
    overview:
      'A concentrated six-week strength cycle that can be applied to a single main lift, such as squat or bench, starting with repeated work around 80% and then tapering volume while intensity climbs toward a max test.',
    howItRuns: [
      'The first half builds a lot of volume on one chosen lift, commonly through repeated sets at roughly the same percentage while reps increase session to session.',
      'The second half shifts toward heavier, lower-rep work so you can express the strength built during the volume wave.',
      'Because the frequency and specificity are high, the rest of training usually needs to stay simple and fatigue-aware.',
    ],
    bestFor: [
      'Short-term specialization on a single lift',
      'Lifters who recover well from frequent practice on the same movement',
    ],
    chartData: [
      { muscleGroup: 'Quads', volume: 82 },
      { muscleGroup: 'Glutes', volume: 76 },
      { muscleGroup: 'Chest', volume: 88 },
      { muscleGroup: 'Back', volume: 94 },
      { muscleGroup: 'Triceps', volume: 64 },
      { muscleGroup: 'Biceps', volume: 58 },
      { muscleGroup: 'Shoulders', volume: 72 },
    ],
  },
  {
    id: 'candito-6-week',
    slug: 'candito-6-week',
    name: 'Candito 6 Week',
    tagline: 'Block-style strength cycle with changing weekly emphasis',
    blockLength: '6-week block',
    cadence: '4 days / week',
    level: 'Intermediate',
    type: 'Complete',
    focus: 'Strength peak across the big three',
    tags: ['Powerlifting', 'Periodized', '6 weeks'],
    accent: 'emerald',
    overview:
      'A six-week strength cycle that rotates emphasis from conditioning and hypertrophy into heavier strength and peaking work.',
    howItRuns: [
      'Each week has a distinct role, moving from higher workload into heavier, more specific strength work.',
      'The structure is more block-oriented than a static weekly template, so the feel of training changes across the cycle.',
      'It works well when you want a defined short block with a clear performance focus at the end.',
    ],
    bestFor: ['Intermediates who want a sharper peak', 'Lifters who enjoy week-to-week variation'],
    chartData: [
      { muscleGroup: 'Quads', volume: 82 },
      { muscleGroup: 'Glutes', volume: 76 },
      { muscleGroup: 'Chest', volume: 88 },
      { muscleGroup: 'Back', volume: 94 },
      { muscleGroup: 'Triceps', volume: 64 },
      { muscleGroup: 'Biceps', volume: 58 },
      { muscleGroup: 'Shoulders', volume: 72 },
    ],
  },
];
