import { redirect } from '@sveltejs/kit';
import { user } from '$lib/stores/appState';
import { get } from 'svelte/store';

export const load = async ({ fetch, url }) => {
  const value = get(user);
  const name = value?.name ?? '';
  if (name != '') {
    console.log('User logged in!');
    return;
  }

  const section = url.pathname.split('/')[1] || 'home';
  const section2debug = url.pathname.split('/')[2] || '';
};

export const prerender = true;
