import { browser } from '$app/environment';
import { env as pub } from '$env/dynamic/public';

let _db: import('firebase/firestore').Firestore | undefined;

export async function getDb() {
  if (!browser) return undefined; 

  if (_db) return _db;

  const [{ initializeApp, getApps }, { getFirestore }] = await Promise.all([
    import('firebase/app'),
    import('firebase/firestore')
  ]);

  const app =
    getApps().length > 0
      ? getApps()[0]
      : initializeApp({
          apiKey: pub.PUBLIC_FIREBASE_API_KEY,
          authDomain: pub.PUBLIC_FIREBASE_AUTH_DOMAIN,
          projectId: pub.PUBLIC_FIREBASE_PROJECT_ID,
          storageBucket: pub.PUBLIC_FIREBASE_STORAGE_BUCKET,
          messagingSenderId: pub.PUBLIC_FIREBASE_MESSAGING_SENDER_ID,
          appId: pub.PUBLIC_FIREBASE_APP_ID
        });

  _db = getFirestore(app);
  return _db;
}
