import { api } from './client';
import type { Category } from './categories';

export interface Team {
	id: string;
	org_id: string;
	name: string;
	description: string | null;
	created_at: string;
	updated_at: string;
}

export interface TeamMember {
	team_id: string;
	user_id: string;
	full_name: string;
	email: string;
	avatar_url: string | null;
	added_at: string;
}

export interface TeamDetail {
	team: Team;
	members: TeamMember[];
	categories: Category[];
}

export interface CreateTeamInput {
	name: string;
	description?: string;
}

export interface UpdateTeamInput {
	name?: string;
	description?: string;
}

export const teamsApi = {
	list: () => api.get<Team[]>('/api/v1/teams'),

	create: (input: CreateTeamInput) => api.post<Team>('/api/v1/teams', input),

	get: (id: string) => api.get<TeamDetail>(`/api/v1/teams/${id}`),

	update: (id: string, input: UpdateTeamInput) =>
		api.patch<Team>(`/api/v1/teams/${id}`, input),

	remove: (id: string) => api.delete<void>(`/api/v1/teams/${id}`),

	listMembers: (id: string) => api.get<TeamMember[]>(`/api/v1/teams/${id}/members`),

	addMember: (id: string, userId: string) =>
		api.post<void>(`/api/v1/teams/${id}/members`, { user_id: userId }),

	removeMember: (id: string, userId: string) =>
		api.delete<void>(`/api/v1/teams/${id}/members/${userId}`),

	listCategories: (id: string) =>
		api.get<Category[]>(`/api/v1/teams/${id}/categories`),

	addCategory: (id: string, categoryId: string) =>
		api.post<void>(`/api/v1/teams/${id}/categories`, { category_id: categoryId }),

	removeCategory: (id: string, categoryId: string) =>
		api.delete<void>(`/api/v1/teams/${id}/categories/${categoryId}`)
};
