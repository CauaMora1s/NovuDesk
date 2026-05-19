import { redirect } from '@sveltejs/kit';
import { browser } from '$app/environment';
import { get } from 'svelte/store';
import { isAuthenticated } from '$lib/stores/auth';

export function load() {
	if (browser) {
		redirect(307, get(isAuthenticated) ? '/dashboard' : '/login');
	}
}
