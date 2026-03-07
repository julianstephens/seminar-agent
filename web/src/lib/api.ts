/**
 * Thin fetch wrapper that injects Auth0 bearer tokens into every request and
 * normalizes error responses.  All methods throw an `ApiError` on non-2xx
 * responses so call-sites can use a consistent error type.
 *
 * Usage:
 *   const api = createApiClient(getAccessToken);
 *   const seminars = await api.listSeminars();
 */

import type {
  Artifact,
  CreateArtifactInput,
  CreateSeminarInput,
  CreateSessionInput,
  CreateTutorialInput,
  CreateTutorialSessionInput,
  Seminar,
  Session,
  SessionDetail,
  SubmitTutorialTurnResponse,
  Turn,
  Tutorial,
  TutorialSession,
  TutorialSessionDetail,
  TutorialTurn,
  UpdateSeminarInput,
  UpdateTutorialInput,
} from "./types";

const BASE_URL =
  ((import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "") + "/v1";

// ── Error ─────────────────────────────────────────────────────────────────────

export class ApiRequestError extends Error {
  status: number;
  detail?: unknown;

  constructor(status: number, message: string, detail?: unknown) {
    super(message);
    this.name = "ApiRequestError";
    this.status = status;
    this.detail = detail;
  }
}

// ── Core fetch helper ─────────────────────────────────────────────────────────

async function request<T>(
  getToken: () => Promise<string>,
  method: string,
  path: string,
  body?: unknown,
): Promise<T> {
  const token = await getToken();
  const headers: HeadersInit = {
    Authorization: `Bearer ${token}`,
    ...(body !== undefined ? { "Content-Type": "application/json" } : {}),
  };

  const res = await fetch(`${BASE_URL}${path}`, {
    method,
    headers,
    ...(body !== undefined ? { body: JSON.stringify(body) } : {}),
  });

  if (!res.ok) {
    let message = res.statusText;
    let detail: unknown;
    try {
      const json = (await res.json()) as { error?: string };
      message = json.error ?? message;
      detail = json;
    } catch {
      // leave defaults
    }
    throw new ApiRequestError(res.status, message, detail);
  }

  // 204 No Content
  if (res.status === 204) return undefined as unknown as T;

  return res.json() as Promise<T>;
}

// ── Client factory ────────────────────────────────────────────────────────────

export function createApiClient(getToken: () => Promise<string>) {
  const get = <T>(path: string) => request<T>(getToken, "GET", path);
  const post = <T>(path: string, body?: unknown) =>
    request<T>(getToken, "POST", path, body ?? {});
  const patch = <T>(path: string, body: unknown) =>
    request<T>(getToken, "PATCH", path, body);
  const del = <T>(path: string) => request<T>(getToken, "DELETE", path);

  return {
    // ── Seminars ────────────────────────────────────────────────────────────
    listSeminars: () => get<Seminar[]>("/seminars"),

    getSeminar: (id: string) => get<Seminar>(`/seminars/${id}`),

    createSeminar: (input: CreateSeminarInput) =>
      post<Seminar>("/seminars", input),

    updateSeminar: (id: string, input: UpdateSeminarInput) =>
      patch<Seminar>(`/seminars/${id}`, input),

    deleteSeminar: (id: string) => del<void>(`/seminars/${id}`),

    // ── Sessions ─────────────────────────────────────────────────────────────
    createSession: (seminarId: string, input: CreateSessionInput) =>
      post<Session>(`/seminars/${seminarId}/sessions`, input),

    getSession: (sessionId: string) =>
      get<SessionDetail>(`/sessions/${sessionId}`),

    listSessions: (seminarId: string) =>
      get<Session[]>(`/seminars/${seminarId}/sessions`),

    abandonSession: (sessionId: string) =>
      post<Session>(`/sessions/${sessionId}/abandon`),

    deleteSession: (sessionId: string) => del<void>(`/sessions/${sessionId}`),

    submitResidue: (sessionId: string, residueText: string) =>
      post<Session>(`/sessions/${sessionId}/residue`, {
        residue_text: residueText,
      }),

    // ── Turns ─────────────────────────────────────────────────────────────────
    submitTurn: (sessionId: string, text: string) =>
      post<Turn>(`/sessions/${sessionId}/turns`, { text }),

    // ── Exports ───────────────────────────────────────────────────────────────
    exportSeminar: (seminarId: string, format: "json" | "md" = "json") =>
      get<{ url: string }>(`/seminars/${seminarId}/export?format=${format}`),

    exportSession: (sessionId: string, format: "json" | "md" = "json") =>
      get<{ url: string }>(`/sessions/${sessionId}/export?format=${format}`),

    exportTutorial: (tutorialId: string, format: "json" | "md" = "json") =>
      get<{ url: string }>(`/tutorials/${tutorialId}/export?format=${format}`),

    exportTutorialSession: (
      sessionId: string,
      format: "json" | "md" = "json",
    ) =>
      get<{ url: string }>(
        `/tutorial-sessions/${sessionId}/export?format=${format}`,
      ),

    // ── Tutorials ─────────────────────────────────────────────────────────────
    listTutorials: () => get<Tutorial[]>("/tutorials"),

    getTutorial: (id: string) => get<Tutorial>(`/tutorials/${id}`),

    createTutorial: (input: CreateTutorialInput) =>
      post<Tutorial>("/tutorials", input),

    updateTutorial: (id: string, input: UpdateTutorialInput) =>
      patch<Tutorial>(`/tutorials/${id}`, input),

    deleteTutorial: (id: string) => del<void>(`/tutorials/${id}`),

    // ── Tutorial Sessions ─────────────────────────────────────────────────────
    createTutorialSession: (
      tutorialId: string,
      input?: CreateTutorialSessionInput,
    ) =>
      post<TutorialSession>(`/tutorials/${tutorialId}/sessions`, input ?? {}),

    listTutorialSessions: (tutorialId: string) =>
      get<TutorialSession[]>(`/tutorials/${tutorialId}/sessions`),

    getTutorialSession: (sessionId: string) =>
      get<TutorialSessionDetail>(`/tutorial-sessions/${sessionId}`),

    completeTutorialSession: (sessionId: string, notes?: string) =>
      post<TutorialSession>(`/tutorial-sessions/${sessionId}/complete`, {
        notes: notes ?? "",
      }),

    abandonTutorialSession: (sessionId: string) =>
      post<TutorialSession>(`/tutorial-sessions/${sessionId}/abandon`),

    deleteTutorialSession: (sessionId: string) =>
      del<void>(`/tutorial-sessions/${sessionId}`),

    // ── Artifacts ─────────────────────────────────────────────────────────────
    listArtifacts: (sessionId: string) =>
      get<Artifact[]>(`/tutorial-sessions/${sessionId}/artifacts`),

    createArtifact: (sessionId: string, input: CreateArtifactInput) =>
      post<Artifact>(`/tutorial-sessions/${sessionId}/artifacts`, input),

    deleteArtifact: (sessionId: string, artifactId: string) =>
      del<void>(`/tutorial-sessions/${sessionId}/artifacts/${artifactId}`),

    // ── Tutorial Turns ────────────────────────────────────────────────────────
    submitTutorialTurn: (sessionId: string, text: string) =>
      post<SubmitTutorialTurnResponse>(
        `/tutorial-sessions/${sessionId}/turns`,
        { text },
      ),

    listTutorialTurns: (sessionId: string) =>
      get<TutorialTurn[]>(`/tutorial-sessions/${sessionId}/turns`),

    tutorialSessionEventsUrl: (sessionId: string) =>
      `${BASE_URL}/tutorial-sessions/${sessionId}/events`,
  };
}

export type ApiClient = ReturnType<typeof createApiClient>;
