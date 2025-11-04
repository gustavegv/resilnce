const greetings = {
  morning: ['Good morning.', 'Ready to move?', 'Are you ready?', 'Starting strong today!'],
  day: ['Good day.', "Hope your day's going well!", 'Stay focused.'],
  evening: ['Good evening.', 'Hope your day was good.', 'Late night lift?'],
  general: [
    'Welcome back!',
    'Hello.',
    "Let's get moving!",
    'Great to see you.',
    "Glad you're here!",
    'Ready to train?',
    'Welcome!',
    'Hi there.',
    "Let's begin.",
    "Let's get started.",
  ],
  continue: ['Keep going', "Let's go", 'Good work'],
};

export function greet(name: string, existingSession: boolean): string {
  var punctuation;

  const message = getMessage(existingSession);

  var greeting = message.slice(0, -1);
  punctuation = message[message.length - 1];

  const greetMessage = greeting + ' ' + normalizeName(name) + punctuation;
  return greetMessage;
}

function getMessage(existingSession: boolean): string {
  var chosenPool: string[];
  const rng = seededRNG(hourlySeed());

  if (existingSession) {
    chosenPool = greetings['continue'];
  } else {
    const tod = getTimeOfDay();
    const pool = greetings[tod];
    const odds = 1 / pool.length;
    chosenPool = rng() > odds ? pool : greetings['general'];
  }

  const randomIndex = Math.floor(rng() * chosenPool.length);
  const message = chosenPool[randomIndex];
  return message;
}

function hourlySeed(): number {
  return Math.floor(Date.now() / (60 * 60 * 1000));
}

function seededRNG(seed: number) {
  return function rand() {
    let t = (seed += 0x6d2b79f5);
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

function getTimeOfDay(): 'morning' | 'day' | 'evening' | 'general' {
  const hour = new Date().getHours();
  let timeOfDay: 'morning' | 'day' | 'evening' | 'general' = 'general';

  if (hour >= 5 && hour < 12) {
    timeOfDay = 'morning';
  } else if (hour >= 12 && hour < 18) {
    timeOfDay = 'day';
  } else {
    timeOfDay = 'evening';
  }
  return timeOfDay;
}

function normalizeName(input: string): string {
  const firstN = input.split(' ')[0];
  const captial = String(firstN).charAt(0).toUpperCase() + String(firstN).slice(1);
  return captial;
}
