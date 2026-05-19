import { get } from 'svelte/store';
import { authStore } from '$lib/stores/auth';

export type Permission =
	| 'tickets:create'
	| 'tickets:view'
	| 'tickets:update_own'
	| 'tickets:update_any'
	| 'tickets:delete'
	| 'tickets:assign'
	| 'tickets:change_status'
	| 'tickets:set_priority'
	| 'tickets:add_tags'
	| 'comments:create_public'
	| 'comments:create_internal'
	| 'comments:edit_own'
	| 'comments:delete_own'
	| 'teams:view'
	| 'teams:manage'
	| 'users:view'
	| 'users:invite'
	| 'users:manage_roles'
	| 'users:deactivate'
	| 'organization:view_settings'
	| 'organization:manage_settings'
	| 'sla:view'
	| 'sla:manage'
	| 'reports:view'
	| 'api_keys:manage';

/**
 * Returns true if the current authenticated user has the given permission.
 * Elements should be conditionally rendered — never hidden with CSS.
 */
export function can(permission: Permission): boolean {
	const auth = get(authStore);
	if (!auth.user) return false;
	return auth.permissions.includes(permission);
}

/**
 * Returns true if the user holds ANY of the given permissions.
 */
export function canAny(...permissions: Permission[]): boolean {
	return permissions.some(can);
}

/**
 * Returns true if the user holds ALL of the given permissions.
 */
export function canAll(...permissions: Permission[]): boolean {
	return permissions.every(can);
}
