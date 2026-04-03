export type PreviewLabelConfig = {
  appearAtPercent: number;
  label: string;
};

  export const config = {
    cardCount: 10,
    scrollTurns: 0.77,
    smoothing: 0.11,
    orbitX: 0,
    orbitY: 37,
    orbitZ: 0,
    phoneWidth: 200,
    shellSteps: 20,
  } as const;


export const previewLabelConfig: PreviewLabelConfig[] = [
  { appearAtPercent: 1, label: 'Your training, all in one place' },
  { appearAtPercent: 12, label: 'Create workouts in seconds' },
  { appearAtPercent: 23, label: 'Customize exercises your way' },
  { appearAtPercent: 36, label: 'Start sessions instantly' },
  { appearAtPercent: 48, label: 'Smart progression built in' },
  { appearAtPercent: 60, label: 'See your progress clearly' },
  { appearAtPercent: 73, label: 'Import notes fast with quick fill' },
  { appearAtPercent: 98, label: 'Built for better training' },

];

  export const carouselSources = [
    '/app_screenshots/blank.png',
    '/app_screenshots/logo.png',
    '/app_screenshots/blank.png',

    '/app_screenshots/create_q.png',
    '/app_screenshots/stats.png',
    '/app_screenshots/finish.png',
    '/app_screenshots/start.png',
    '/app_screenshots/manual.png',
    '/app_screenshots/create.png',
    '/app_screenshots/home.png',
  ];
