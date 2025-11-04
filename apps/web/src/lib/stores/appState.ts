import { writable } from 'svelte/store';

export const user = writable<{ name: string; id: string } | null>(null);
