import { authStore } from '$lib/stores/auth';
import { goto } from '$app/navigation';

// In dev the Vite proxy forwards /api → http://api:8080/api (same-origin, no CORS).
// In prod VITE_API_URL is set to the deployed API base (e.g. https://api.example.com).
const API_BASE = import.meta.env.VITE_API_URL ?? '';

export interface ApiError {
	code: string;
	message: string;
	details?: unknown;
}

export class HttpError extends Error {
	constructor(
		public status: number,
		public error: ApiError
	) {
		super(error.message);
	}
}

async function request<T>(
	path: string,
	options: RequestInit = {}
): Promise<T> {
	const token = authStore.getToken();

	const headers: Record<string, string> = {
		'Content-Type': 'application/json',
		'X-Requested-With': 'XMLHttpRequest',
		...(options.headers as Record<string, string>)
	};

	if (token) {
		headers['Authorization'] = `Bearer ${token}`;
	}

	const controller = new AbortController();
	const timer = setTimeout(() => controller.abort(), 15000);

	let res: Response;
	try {
		res = await fetch(`${API_BASE}${path}`, {
			...options,
			headers,
			credentials: 'include',
			signal: controller.signal
		});
	} catch (err: unknown) {
		if (err instanceof DOMException && err.name === 'AbortError') {
			throw new HttpError(504, { code: 'TIMEOUT', message: 'Request timed out' });
		}
		throw err;
	} finally {
		clearTimeout(timer);
	}

	if (res.status === 401) {
		authStore.clear();
		goto('/login');
		throw new HttpError(401, { code: 'UNAUTHORIZED', message: 'Session expired' });
	}

	if (!res.ok) {
		const body = await res.json().catch(() => ({ error: { code: 'UNKNOWN', message: 'Request failed' } }));
		throw new HttpError(res.status, body.error ?? body);
	}

	if (res.status === 204) return undefined as T;

	return res.json().then((body) => body.data ?? body);
}

export const api = {
	get: <T>(path: string) => request<T>(path),
	post: <T>(path: string, body: unknown) =>
		request<T>(path, { method: 'POST', body: JSON.stringify(body) }),
	patch: <T>(path: string, body: unknown) =>
		request<T>(path, { method: 'PATCH', body: JSON.stringify(body) }),
	delete: <T>(path: string) => request<T>(path, { method: 'DELETE' })
};
