import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent }) => {
  const { user } = await parent();
  if (!user) redirect(302, '/login');

  const res = await fetch('/api/admin/flags');
  const data = res.ok ? await res.json() : { flags: [] };
  return { flags: data.flags ?? [] };
};
