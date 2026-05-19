import { writable, derived } from 'svelte/store';
import type { Permission } from '$lib/permissions';

export interface AuthUser {
	id: string;
	email: string;
	fullName: string;
	avatarUrl: string | null;
	locale: string;
}

interface AuthState {
	user: AuthUser | null;
	orgId: string | null;
	orgSlug: string | null;
	role: string | null;
	permissions: Permission[];
	teamIds: string[];
	accessToken: string | null;
}

interface SessionData {
	user: AuthUser;
	orgId: string;
	orgSlug: string;
	role: string;
	permissions: Permission[];
	teamIds: string[];
	accessToken: string;
}

const STORAGE_KEY = 'nd_session';

const empty: AuthState = {
	user: null,
	orgId: null,
	orgSlug: null,
	role: null,
	permissions: [],
	teamIds: [],
	accessToken: null
};

function decodeJwt(token: string): Record<string, unknown> | null {
	try {
		const [, b64] = token.split('.');
		return JSON.parse(atob(b64.replace(/-/g, '+').replace(/_/g, '/')));
	} catch {
		return null;
	}
}

function readStorage(): AuthState {
	if (typeof localStorage === 'undefined') return empty;
	try {
		// New format
		const raw = localStorage.getItem(STORAGE_KEY);
		if (raw) {
			const s: SessionData = JSON.parse(raw);
			if (s.accessToken && s.user) {
				return {
					user: s.user,
					orgId: s.orgId,
					orgSlug: s.orgSlug,
					role: s.role,
					permissions: s.permissions ?? [],
					teamIds: s.teamIds ?? [],
					accessToken: s.accessToken
				};
			}
		}

		// Old format migration — decode what we can from the JWT
		const token = localStorage.getItem('nd_access_token');
		const orgSlug = localStorage.getItem('nd_org_slug');
		if (token && orgSlug) {
			const payload = decodeJwt(token);
			if (payload?.uid) {
				return {
					user: {
						id: payload.uid as string,
						email: '',
						fullName: '',
						avatarUrl: null,
						locale: 'pt'
					},
					orgId: (payload.oid as string) ?? null,
					orgSlug,
					role: (payload.role as string) ?? null,
					permissions: (payload.perms as Permission[]) ?? [],
					teamIds: (payload.team_ids as string[]) ?? [],
					accessToken: token
				};
			}
		}

		return empty;
	} catch {
		return empty;
	}
}

function createAuthStore() {
	const { subscribe, set, update } = writable<AuthState>(readStorage());

	return {
		subscribe,

		setSession(data: SessionData) {
			update((s) => ({ ...s, ...data }));
			try {
				localStorage.setItem(STORAGE_KEY, JSON.stringify(data));
			} catch { /* quota exceeded — ignore */ }
		},

		clear() {
			try {
				localStorage.removeItem(STORAGE_KEY);
			} catch { /* ignore */ }
			set(empty);
		},

		getToken(): string | null {
			let token: string | null = null;
			const unsub = subscribe((s) => {
				token = s.accessToken;
			});
			unsub();
			return token;
		}
	};
}

export const authStore = createAuthStore();

export const isAuthenticated = derived(authStore, ($auth) => !!$auth.user);
export const currentUser = derived(authStore, ($auth) => $auth.user);
