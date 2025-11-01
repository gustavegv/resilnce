import { writable } from 'svelte/store';

export const user = writable<{ name: string } | null>(null);
