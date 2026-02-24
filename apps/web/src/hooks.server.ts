import type { Handle } from '@sveltejs/kit';

export const handle: Handle = async ({ event, resolve }) => {
  const session = event.cookies.get('zist_session');
  if (session) {
    try {
      // Decode JWT payload (Base64URL) — no signature verification needed here.
      // The token was issued by our own auth service and validated by the gateway
      // on every API call via propagateAuth middleware.
      const parts = session.split('.');
      if (parts.length === 3) {
        const padded = parts[1].replace(/-/g, '+').replace(/_/g, '/');
        const payload = JSON.parse(Buffer.from(padded, 'base64').toString('utf8'));
        const nowSec = Math.floor(Date.now() / 1000);

        if (payload.exp && payload.exp > nowSec) {
          event.locals.user = {
            user_id: payload.user_id || payload.sub || '',
            email:     payload.email || '',
            tenant_id: payload.tenant_id || '',
            scopes:    Array.isArray(payload.roles)
              ? payload.roles.join(',')
              : (payload.scope || ''),
          };
        }
      }
    } catch {
      // Malformed token — treat as anonymous
    }
  }
  return resolve(event);
};
