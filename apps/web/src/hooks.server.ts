import type { Handle } from '@sveltejs/kit';

export const handle: Handle = async ({ event, resolve }) => {
  if (event.cookies.get('zist_session')) {
    const resp = await event.fetch('/api/auth/me');
    if (resp.ok) {
      event.locals.user = await resp.json();
    }
  }
  return resolve(event);
};
