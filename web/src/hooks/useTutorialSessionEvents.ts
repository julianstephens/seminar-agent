/**
 * useTutorialSessionEvents – SSE subscription hook for a single tutorial session.
 *
 * Uses fetch() + ReadableStream so an Authorization header can be injected;
 * the browser's native EventSource does not support custom headers.
 *
 * Tutorial-specific events (no phase/timer semantics from seminar runner):
 *   - tutorial_turn_added
 *   - tutorial_artifact_added
 *   - tutorial_artifact_deleted
 *   - tutorial_session_completed
 *   - error
 */

import { useAccessToken } from "@/auth/useAuth";
import type { Artifact, TutorialTurn } from "@/lib/types";
import { useEffect } from "react";

const BASE_URL =
  ((import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "") + "/v1";

// ── Payload types ─────────────────────────────────────────────────────────────

export interface TutorialTurnAddedPayload {
  turn: TutorialTurn;
}

export interface TutorialSessionCompletedPayload {
  session_id: string;
}

export interface TutorialArtifactAddedPayload {
  artifact: Artifact;
}

export interface TutorialArtifactDeletedPayload {
  artifact_id: string;
}

export interface TutorialSseErrorPayload {
  message: string;
}

// ── Options ───────────────────────────────────────────────────────────────────

export interface UseTutorialSessionEventsOptions {
  /** Fires when a new turn (user or agent) is persisted. */
  onTurnAdded?: (payload: TutorialTurnAddedPayload) => void;
  /** Fires when a new artifact is added to the session. */
  onArtifactAdded?: (payload: TutorialArtifactAddedPayload) => void;
  /** Fires when an artifact is removed from the session. */
  onArtifactDeleted?: (payload: TutorialArtifactDeletedPayload) => void;
  /** Fires when the session reaches a terminal state. */
  onSessionCompleted?: (payload: TutorialSessionCompletedPayload) => void;
  /** Fires on a non-fatal stream-level error emitted by the server. */
  onError?: (payload: TutorialSseErrorPayload) => void;
  /** Fires when the fetch connection itself fails or the stream ends unexpectedly. */
  onConnectionError?: (error: unknown) => void;
}

// ── Hook ──────────────────────────────────────────────────────────────────────

/**
 * Opens an SSE stream for `sessionId` and calls the supplied handlers as
 * events arrive. Cleans up (aborts the fetch) on unmount.
 */
export function useTutorialSessionEvents(
  sessionId: string | undefined,
  options: UseTutorialSessionEventsOptions,
): void {
  const getToken = useAccessToken();

  useEffect(() => {
    if (!sessionId) return;

    let cancelled = false;
    const controller = new AbortController();

    async function connect() {
      try {
        const token = await getToken();
        const res = await fetch(
          `${BASE_URL}/tutorial-sessions/${sessionId}/events`,
          {
            headers: { Authorization: `Bearer ${token}` },
            signal: controller.signal,
          },
        );

        if (!res.ok || !res.body) {
          options.onConnectionError?.(
            new Error(`SSE connect failed: ${res.status} ${res.statusText}`),
          );
          return;
        }

        const reader = res.body.getReader();
        const decoder = new TextDecoder();
        let buf = "";

        while (!cancelled) {
          const { done, value } = await reader.read();
          if (done) break;

          buf += decoder.decode(value, { stream: true });

          // SSE messages are delimited by blank lines ("event: …\ndata: …\n\n").
          const parts = buf.split("\n\n");
          buf = parts.pop() ?? "";

          for (const part of parts) {
            let eventType = "message";
            const dataLines: string[] = [];

            for (const line of part.split("\n")) {
              if (line.startsWith("event:")) {
                eventType = line.slice(6).trim();
              } else if (line.startsWith("data:")) {
                dataLines.push(line.slice(5).trim());
              }
            }

            if (dataLines.length === 0) continue;

            let payload: unknown;
            try {
              payload = JSON.parse(dataLines.join("\n")) as unknown;
            } catch {
              continue;
            }

            dispatch(eventType, payload, options);
          }
        }
      } catch (e) {
        if (
          !cancelled &&
          !(e instanceof DOMException && e.name === "AbortError")
        ) {
          options.onConnectionError?.(e);
        }
      }
    }

    void connect();

    return () => {
      cancelled = true;
      controller.abort();
    };
    // `options` is intentionally omitted from deps to avoid re-opening the stream.
    // `getToken` identity is stable across renders (Auth0 SDK contract).
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionId, getToken]);
}

// ── Internal dispatch ─────────────────────────────────────────────────────────

function dispatch(
  type: string,
  payload: unknown,
  opts: UseTutorialSessionEventsOptions,
): void {
  switch (type) {
    case "tutorial_turn_added":
      opts.onTurnAdded?.(payload as TutorialTurnAddedPayload);
      break;
    case "tutorial_artifact_added":
      opts.onArtifactAdded?.(payload as TutorialArtifactAddedPayload);
      break;
    case "tutorial_artifact_deleted":
      opts.onArtifactDeleted?.(payload as TutorialArtifactDeletedPayload);
      break;
    case "tutorial_session_completed":
      opts.onSessionCompleted?.(payload as TutorialSessionCompletedPayload);
      break;
    case "error":
      opts.onError?.(payload as TutorialSseErrorPayload);
      break;
    default:
      // Unknown event type – ignore.
      break;
  }
}
