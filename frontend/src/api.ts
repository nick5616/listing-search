import type { FieldError, SearchParams, SearchResponse } from "./types";

export class ApiError extends Error {
  constructor(
    message: string,
    public readonly status: number,
    public readonly details: FieldError[] = [],
  ) {
    super(message);
  }
}

// Builds the query string, leaving out blank fields so "no filter" is never sent as "".
export function buildQuery(params: SearchParams): URLSearchParams {
  const qs = new URLSearchParams();
  const { page, ...fields } = params;
  for (const [name, value] of Object.entries(fields)) {
    const trimmed = String(value).trim();
    if (trimmed !== "") qs.set(name, trimmed);
  }
  qs.set("page", String(page));
  return qs;
}

async function getJSON<T>(url: string, signal?: AbortSignal): Promise<T> {
  let res: Response;
  try {
    res = await fetch(url, { signal });
  } catch (err) {
    if ((err as Error).name === "AbortError") throw err;
    throw new ApiError("Can't reach the server. Is the backend running on port 8080?", 0);
  }
  // A non-JSON body becomes null, but an abort mid-read must still surface as an abort.
  const body = await res.json().catch((err: Error) => {
    if (err.name === "AbortError") throw err;
    return null;
  });
  if (!res.ok) {
    const details: FieldError[] = body?.details ?? [];
    throw new ApiError(body?.error ?? `Request failed (${res.status})`, res.status, details);
  }
  return body as T;
}

export function searchListings(params: SearchParams, signal?: AbortSignal) {
  return getJSON<SearchResponse>(`/api/listings/search?${buildQuery(params)}`, signal);
}

export async function fetchCities(signal?: AbortSignal): Promise<string[]> {
  const body = await getJSON<{ cities: string[] }>("/api/cities", signal);
  return body.cities;
}
