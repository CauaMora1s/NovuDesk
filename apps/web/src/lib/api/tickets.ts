import { authStore } from '$lib/stores/auth';
import { goto } from '$app/navigation';
import { api } from './client';

export type TicketStatus = 'open' | 'pending' | 'on_hold' | 'resolved' | 'closed';
export type TicketPriority = 'low' | 'normal' | 'high' | 'urgent';
export type TicketSort = 'created_at' | 'updated_at' | 'sla_due';

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

export interface TicketListResult {
	data: Ticket[];
	total: number;
}

export interface ListTicketsParams {
	status?: TicketStatus | TicketStatus[];
	priority?: TicketPriority | TicketPriority[];
	assignee_id?: string;
	team_id?: string;
	q?: string;
	number?: number;
	sla_breached?: boolean;
	sort?: TicketSort;
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

function buildQuery(params: ListTicketsParams): string {
	const parts: string[] = [];

	for (const [key, val] of Object.entries(params)) {
		if (val === undefined || val === null || val === '') continue;
		if (Array.isArray(val)) {
			for (const item of val) {
				parts.push(`${key}=${encodeURIComponent(String(item))}`);
			}
		} else {
			parts.push(`${key}=${encodeURIComponent(String(val))}`);
		}
	}

	return parts.length ? `?${parts.join('&')}` : '';
}

// Internal raw request that returns the full response envelope.
const API_BASE = import.meta.env.VITE_API_URL ?? '';

async function listRaw(params: ListTicketsParams): Promise<TicketListResult> {
	const token = authStore.getToken();
	const headers: Record<string, string> = {
		'Content-Type': 'application/json',
		'X-Requested-With': 'XMLHttpRequest'
	};
	if (token) headers['Authorization'] = `Bearer ${token}`;

	const res = await fetch(`${API_BASE}/api/v1/tickets${buildQuery(params)}`, {
		headers,
		credentials: 'include'
	});

	if (res.status === 401) {
		authStore.clear();
		goto('/login');
		throw new Error('Session expired');
	}

	const body = await res.json();
	const data: Ticket[] = Array.isArray(body.data) ? body.data : [];
	const total: number = body.meta?.total ?? data.length;
	return { data, total };
}

export const ticketsApi = {
	list: (params: ListTicketsParams = {}): Promise<TicketListResult> => listRaw(params),

	get: (id: string) => api.get<Ticket>(`/api/v1/tickets/${id}`),

	create: (input: CreateTicketInput) => api.post<Ticket>('/api/v1/tickets', input),

	update: (id: string, input: UpdateTicketInput) =>
		api.patch<Ticket>(`/api/v1/tickets/${id}`, input),

	delete: (id: string) => api.delete<void>(`/api/v1/tickets/${id}`)
};
