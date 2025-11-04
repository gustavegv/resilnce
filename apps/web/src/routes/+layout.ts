import { redirect } from '@sveltejs/kit';
import { user } from '$lib/stores/appState';
import { get } from 'svelte/store';

export const load = async ({ fetch, url }) => {
  const section = url.pathname.split('/')[1] || 'home';

  switch (section) {
    case 'account':
      console.log('On account');
      return;

    case 'home':
      console.log('On home');
      return;

    case 'oauth':
      console.log('On oauth');
      return;

    default:
      const value = get(user);
      const name = value?.name ?? '';
      if (name == '') {
        console.log('No user stored, redirecting');
        redirect(302, '/account');
      }
      break;
  }
};

export const prerender = true;
