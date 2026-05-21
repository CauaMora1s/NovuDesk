import { authStore } from '$lib/stores/auth';

export interface Attachment {
	id: string;
	ticket_id: string;
	comment_id?: string;
	uploader_id?: string;
	filename: string;
	mime_type: string;
	size_bytes: number;
	url: string;
	created_at: string;
}

const API_BASE = import.meta.env.VITE_API_URL ?? '';

export const attachmentsApi = {
	list: async (ticketId: string): Promise<Attachment[]> => {
		const token = authStore.getToken();
		const headers: Record<string, string> = {};
		if (token) headers['Authorization'] = `Bearer ${token}`;

		const res = await fetch(`${API_BASE}/api/v1/tickets/${ticketId}/attachments`, {
			headers,
			credentials: 'include'
		});
		if (!res.ok) throw new Error('Failed to load attachments');
		const body = await res.json() as { data: Attachment[] };
		return body.data ?? [];
	},

	upload: async (ticketId: string, file: File, commentId?: string): Promise<Attachment> => {
		const token = authStore.getToken();
		const headers: Record<string, string> = {};
		if (token) headers['Authorization'] = `Bearer ${token}`;

		const form = new FormData();
		form.append('file', file);
		if (commentId) form.append('comment_id', commentId);

		const res = await fetch(`${API_BASE}/api/v1/tickets/${ticketId}/attachments`, {
			method: 'POST',
			headers,
			credentials: 'include',
			body: form
		});

		if (!res.ok) {
			const err = await res.json().catch(() => ({})) as { error?: { message?: string } };
			throw new Error(err.error?.message ?? res.statusText);
		}

		const body = await res.json() as { data: Attachment };
		return body.data;
	}
};
