import { keycloak } from './auth';
import { PUBLIC_API_URL } from '$env/static/public';

async function request(path: string, options: RequestInit = {}): Promise<Response> {
	await keycloak.updateToken(30);
	return fetch(`${PUBLIC_API_URL}${path}`, {
		...options,
		headers: {
			'Content-Type': 'application/json',
			Authorization: `Bearer ${keycloak.token}`,
			...options.headers
		}
	});
}

export const api = {
	get: (path: string) => request(path),
	post: (path: string, body: unknown) =>
		request(path, { method: 'POST', body: JSON.stringify(body) }),
	patch: (path: string, body: unknown) =>
		request(path, { method: 'PATCH', body: JSON.stringify(body) }),
	delete: (path: string) => request(path, { method: 'DELETE' })
};

export type BookResponse = {
	id: number;
	title: string;
	authors: string;
	subjects: string | null;
	description: string | null;
	cover_art_url: string | null;
	work_id: string;
	status: 'want_to_read' | 'reading' | 'finished' | 'abandoned';
	rating: number | null;
	notes: string | null;
};

export type SearchResult = {
	title: string;
	authors: string[];
	first_publish_year: number;
	work_id: string;
};
