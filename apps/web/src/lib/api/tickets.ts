import { api } from './client';

export type TicketStatus = 'open' | 'pending' | 'on_hold' | 'resolved' | 'closed';
export type TicketPriority = 'low' | 'normal' | 'high' | 'urgent';

export interface Ticket {
	id: string;
	number: number;
	title: string;
	description: string | null;
	status: TicketStatus;
	priority: TicketPriority;
	assignee_id: string | null;
	assignee_name: string | null;
	assignee_avatar: string | null;
	team_id: string | null;
	team_name: string | null;
	requester_id: string | null;
	requester_name: string | null;
	category_id: string | null;
	category_name: string | null;
	tags: string[];
	sla_breached: boolean;
	sla_response_due_at: string | null;
	sla_resolution_due_at: string | null;
	created_at: string;
	updated_at: string;
}

export interface ListTicketsParams {
	status?: TicketStatus;
	priority?: TicketPriority;
	assignee_id?: string;
	team_id?: string;
	q?: string;
	page?: number;
	per_page?: number;
}

export interface CreateTicketInput {
	title: string;
	description?: string;
	priority?: TicketPriority;
	category_id?: string;
	assignee_id?: string;
	team_id?: string;
	sla_policy_id?: string;
	tags?: string[];
}

export interface UpdateTicketInput {
	title?: string;
	description?: string;
	status?: TicketStatus;
	priority?: TicketPriority;
	assignee_id?: string;
	team_id?: string;
	category_id?: string;
	tags?: string[];
}

function buildQuery(params: Record<string, unknown>): string {
	const q = Object.entries(params)
		.filter(([, v]) => v !== undefined && v !== null && v !== '')
		.map(([k, v]) => `${k}=${encodeURIComponent(String(v))}`)
		.join('&');
	return q ? `?${q}` : '';
}

export const ticketsApi = {
	list: (params: ListTicketsParams = {}) =>
		api.get<Ticket[]>(`/api/v1/tickets${buildQuery(params as Record<string, unknown>)}`),

	get: (id: string) => api.get<Ticket>(`/api/v1/tickets/${id}`),

	create: (input: CreateTicketInput) => api.post<Ticket>('/api/v1/tickets', input),

	update: (id: string, input: UpdateTicketInput) =>
		api.patch<Ticket>(`/api/v1/tickets/${id}`, input),

	delete: (id: string) => api.delete<void>(`/api/v1/tickets/${id}`)
};
