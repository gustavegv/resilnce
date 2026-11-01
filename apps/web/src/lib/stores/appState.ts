import { writable } from 'svelte/store';

export type ApplicationModel = 'legacy' | 'exercise-centric';
export type Account = {
  id: string;
  email: string;
  name: string;
  developmentMode: boolean;
  capabilities: {
    applicationModel: ApplicationModel;
    trainingWrites: boolean;
    writeMaintenance: boolean;
  };
};

// Only /api/me supplies account identity; browser cookies never choose an owner.
export const user = writable<Account | null>(null);

export const upgradeRequired = writable(false);
