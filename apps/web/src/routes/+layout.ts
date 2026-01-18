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

  switch (section) {
    // todo: Remove this logic to unlock app at some point
    case 'account':
      console.log('On account');
      return;

    case 'home':
      console.log('On home');
      return;

    case 'oauth':
      console.log('On oauth');
      return;

    case 'create':
      console.log('On create');
      return;

    default:
      const value = get(user);
      const name = value?.name ?? '';
      if (name == '') {
        console.log('Redirect from:', section, '/', section2debug);
        redirect(302, '/');
      }
      break;
  }
};

export const prerender = true;
