import { api } from './client';

export interface Member {
	id: string;
	email: string;
	full_name: string;
	avatar_url: string | null;
	locale: string;
	is_active: boolean;
	org_id: string;
	role_id: string;
	role_name: string;
	joined_at: string;
}

export interface Role {
	id: string;
	name: string;
	is_system_role: boolean;
}

export interface PermissionOverride {
	permission_key: string;
	is_granted: boolean;
}

export interface EffectivePermissions {
	role_permissions: string[];
	overrides: PermissionOverride[];
	effective: string[];
}

export interface CreateMemberInput {
	full_name: string;
	email: string;
	password: string;
	role_id: string;
	team_id?: string;
}

export interface UpdateMemberRoleInput {
	role_id: string;
}

export interface UpdateProfileInput {
	full_name: string;
	email: string;
}

export interface UpdatePasswordInput {
	new_password: string;
}

export const membersApi = {
	list: () => api.get<Member[]>('/api/v1/members'),

	create: (input: CreateMemberInput) => api.post<Member>('/api/v1/members', input),

	updateRole: (id: string, input: UpdateMemberRoleInput) =>
		api.patch<Member>(`/api/v1/members/${id}`, input),

	deactivate: (id: string) => api.delete<void>(`/api/v1/members/${id}`),

	activate: (id: string) => api.post<Member>(`/api/v1/members/${id}/activate`, {}),

	updateProfile: (id: string, input: UpdateProfileInput) =>
		api.patch<Member>(`/api/v1/members/${id}/profile`, input),

	updatePassword: (id: string, input: UpdatePasswordInput) =>
		api.patch<void>(`/api/v1/members/${id}/password`, input),

	getPermissions: (id: string) =>
		api.get<EffectivePermissions>(`/api/v1/members/${id}/permissions`),

	setPermissions: (id: string, overrides: PermissionOverride[]) =>
		api.put<void>(`/api/v1/members/${id}/permissions`, { overrides }),

	listRoles: () => api.get<Role[]>('/api/v1/roles')
};
