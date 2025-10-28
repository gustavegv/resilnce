// src/lib/stores/user.ts
import { writable, type Writable } from 'svelte/store';
import { getCookie, setCookie, deleteCookie } from './cookies';

const COOKIE_KEY = 'username';

const initial = typeof document !== 'undefined' ? getCookie(COOKIE_KEY) : null;

export const user: Writable<string | null> = writable(initial);

user.subscribe((name) => {
  if (typeof document === 'undefined') return;
  if (name) setCookie(COOKIE_KEY, name, 7);
  else deleteCookie(COOKIE_KEY);
});
