export function setCookie(name: string, value: string, days = 7) {
  const d = new Date();
  d.setTime(d.getTime() + days * 24 * 60 * 60 * 1000);
  document.cookie = `${encodeURIComponent(name)}=${encodeURIComponent(value)};expires=${d.toUTCString()};path=/`;
}

export function getCookie(name: string): string | null {
  const match = document.cookie.match(
    new RegExp("(^| )" + encodeURIComponent(name) + "=([^;]+)"),
  );
  return match ? decodeURIComponent(match[2]) : null;
}

export function deleteCookie(name: string) {
  setCookie(name, "", -1);
}
