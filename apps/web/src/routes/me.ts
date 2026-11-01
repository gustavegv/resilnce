import { PUBLIC_BACKEND_BASE_URL } from '$env/static/public';
import { user, type Account } from '$lib/stores/appState';

export async function getMe(): Promise<boolean> {
  try {
    const res = await fetch(PUBLIC_BACKEND_BASE_URL + '/api/me', {
      method: 'GET',
      credentials: 'include',
      cache: 'no-store',
    });
    if (res.ok) {
      user.set((await res.json()) as Account);
      return true;
    }
  } catch {
    // The app cannot safely retain a previously selected account after a failed check.
  }
  user.set(null);
  return false;
}
