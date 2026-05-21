import { browser } from '$app/environment';
import { get } from 'svelte/store';
import { redirect } from '@sveltejs/kit';
import { authStore } from '$lib/stores/auth';
import { can } from '$lib/permissions';

export function load() {
	if (!browser) return;
	const auth = get(authStore);
	const allowed =
		can('teams:view') &&
		(auth.role === 'owner' ||
			auth.role === 'admin' ||
			(auth.teamIds?.length ?? 0) > 0);
	if (!allowed) redirect(307, '/dashboard');
}
