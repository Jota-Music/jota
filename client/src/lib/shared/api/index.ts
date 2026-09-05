type Headers = Record<string, string>;

interface GetParams {
	query?: Record<string, string>;
	headers?: Headers;
}

interface PostParams {
	body?: object;
	headers?: Headers;
	responseType?: "json" | "text" | "blob" | "arrayBuffer";
}

const DEFAULT_HEADERS: Headers = {
	"Content-Type": "application/json",
	Accept: "application/json",
};

function getHeaders(headers?: Headers): Headers {
	return {
		...DEFAULT_HEADERS,
		...(headers ?? {}),
	};
}

export async function get<T>(
	route: string,
	params: GetParams = {},
): Promise<T> {
	try {
		const url = new URL(`/api${route}`, location.origin);

		if (params.query) {
			Object.entries(params.query).forEach(([key, value]) => {
				url.searchParams.set(key, value);
			});
		}

		const response = await fetch(url, {
			method: "GET",
			credentials: "include",
			headers: getHeaders(params.headers),
		});

		if (!response.ok) {
			const body = await response.text().catch(() => "");
			const detail = body || response.statusText;
			throw new Error(`Failed to fetch ${route}: ${detail}`);
		}

		return response.json() as T;
	} catch (error) {
		console.error(error);
		throw error;
	}
}

export async function post<T>(
	route: string,
	body: object,
	params: PostParams = {},
): Promise<T> {
	try {
		const url = new URL(`/api${route}`, location.origin);

		const response = await fetch(url, {
			method: "POST",
			credentials: "include",
			headers: getHeaders(params.headers),
			body: JSON.stringify(body),
		});

		if (!response.ok) {
			const body = await response.text().catch(() => "");
			const detail = body || response.statusText;
			throw new Error(`Failed to post ${route}: ${detail}`);
		}

		return response.json() as T;
	} catch (error) {
		console.error(error);
		throw error;
	}
}
