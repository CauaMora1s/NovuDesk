import { api } from './client';

export interface Category {
	id: string;
	org_id: string;
	name: string;
	description: string | null;
	created_at: string;
	updated_at: string;
}

export interface CreateCategoryInput {
	name: string;
	description?: string;
}

export const categoriesApi = {
	list: () => api.get<Category[]>('/api/v1/categories'),

	create: (input: CreateCategoryInput) => api.post<Category>('/api/v1/categories', input),

	update: (id: string, input: CreateCategoryInput) =>
		api.patch<Category>(`/api/v1/categories/${id}`, input),

	remove: (id: string) => api.delete<void>(`/api/v1/categories/${id}`)
};
