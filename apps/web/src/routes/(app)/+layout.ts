import { redirect } from '@sveltejs/kit';
import { browser } from '$app/environment';
import { get } from 'svelte/store';
import { isAuthenticated } from '$lib/stores/auth';

export function load() {
	// Runs on the client before the layout component renders.
	// With ssr:false the server never touches this, so `browser` is always true
	// at runtime — the guard is kept for type-safety / future-proofing.
	if (browser && !get(isAuthenticated)) {
		redirect(307, '/login');
	}
}
