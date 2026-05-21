import { api } from './client';

export interface Permission {
	id: string;
	key: string;
	description: string;
	created_at: string;
}

export interface RoleWithPermissions {
	id: string;
	org_id: string | null;
	name: string;
	is_system_role: boolean;
	permissions: Permission[];
	created_at: string;
}

export interface CreateRoleInput {
	name: string;
	permission_keys: string[];
}

export interface UpdateRoleInput {
	name: string;
	permission_keys: string[];
}

export const rolesApi = {
	list: () => api.get<RoleWithPermissions[]>('/api/v1/roles'),

	get: (id: string) => api.get<RoleWithPermissions>(`/api/v1/roles/${id}`),

	create: (input: CreateRoleInput) => api.post<RoleWithPermissions>('/api/v1/roles', input),

	update: (id: string, input: UpdateRoleInput) =>
		api.patch<RoleWithPermissions>(`/api/v1/roles/${id}`, input),

	delete: (id: string) => api.delete<void>(`/api/v1/roles/${id}`),

	listAllPermissions: () => api.get<Permission[]>('/api/v1/permissions')
};
