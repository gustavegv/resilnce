import { PUBLIC_BACKEND_BASE_URL } from '$env/static/public';
import { user } from '$lib/stores/appState';

function backendAdress(dir: string): string {
  const baseURL: string = PUBLIC_BACKEND_BASE_URL;
  return baseURL + dir;
}

export async function getMe(): Promise<boolean> {
  const res = await fetch(backendAdress('/api/me'), {
    method: 'GET',
    credentials: 'include',
  });
  if (res.ok) {
    const data = await res.json();
    console.log(data);
    user.set(data);
    return true;
  } else {
    user.set(null);
    console.error('User not logged in.');
    return false;
  }
}
