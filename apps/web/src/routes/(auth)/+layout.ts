import { redirect } from '@sveltejs/kit';
import { browser } from '$app/environment';
import { get } from 'svelte/store';
import { isAuthenticated } from '$lib/stores/auth';

export function load() {
	// If the user is already authenticated, skip the auth screens.
	if (browser && get(isAuthenticated)) {
		redirect(307, '/dashboard');
	}
}
