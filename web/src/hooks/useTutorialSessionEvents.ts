/**
 * useTutorialSessionEvents – SSE subscription hook for a single tutorial session.
 *
 * Uses fetch() + ReadableStream so an Authorization header can be injected;
 * the browser's native EventSource does not support custom headers.
 *
 * Tutorial-specific events:
 *   - turn_added
 *   - agent_response_chunk
 *   - tutorial_artifact_added
 *   - tutorial_artifact_deleted
 *   - session_completed
 *   - error
 */

import { useAccessToken } from "@/auth/useAuth";
import type {
  AgentResponseChunkPayload,
  Artifact,
  TutorialTurn,
} from "@/lib/types";
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
  /** Fires when a streaming agent response chunk arrives. */
  onAgentResponseChunk?: (payload: AgentResponseChunkPayload) => void;
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
        console.log(
          "[SSE] Connecting to:",
          `${BASE_URL}/tutorial-sessions/${sessionId}/events`,
        );
        const token = await getToken();
        const res = await fetch(
          `${BASE_URL}/tutorial-sessions/${sessionId}/events`,
          {
            headers: { Authorization: `Bearer ${token}` },
            signal: controller.signal,
          },
        );

        console.log("[SSE] Response status:", res.status, res.statusText);
        if (!res.ok || !res.body) {
          console.error("[SSE] Connection failed:", res.status, res.statusText);
          options.onConnectionError?.(
            new Error(`SSE connect failed: ${res.status} ${res.statusText}`),
          );
          return;
        }

        console.log("[SSE] Connection established, starting to read stream");
        const reader = res.body.getReader();
        const decoder = new TextDecoder();
        let buf = "";

        while (!cancelled) {
          let readResult;
          try {
            readResult = await reader.read();
          } catch (readError) {
            // Stream read error (e.g., network issue, stream abort)
            if (!cancelled) {
              console.warn("[SSE] stream read error", readError);
            }
            break;
          }

          const { done, value } = readResult;
          if (done) {
            console.log("[SSE] Stream ended (done=true)");
            break;
          }

          buf += decoder.decode(value, { stream: true });

          // SSE messages are delimited by blank lines ("event: …\ndata: …\n\n").
          const parts = buf.split("\n\n");
          buf = parts.pop() ?? "";

          for (const part of parts) {
            // Skip heartbeat comments
            if (part.trim().startsWith(":")) {
              console.log("[SSE] Received heartbeat");
              continue;
            }

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
              console.log("[SSE] Received event:", eventType, payload);
            } catch (e) {
              console.error(
                "[SSE] Failed to parse event data:",
                dataLines.join("\n"),
                e,
              );
              continue;
            }

            dispatch(eventType, payload, options);
          }
        }
      } catch (e) {
        if (!(e instanceof DOMException && e.name === "AbortError")) {
          console.error("[SSE] Connection error:", e);
          options.onConnectionError?.(e);
        } else {
          console.log("[SSE] Connection aborted");
        }
      } finally {
        cancelled = true;
        console.log("[SSE] Cleaning up connection");
        try {
          reader.cancel();
        } catch {
          // Ignore errors when cancelling reader
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
    case "turn_added":
      opts.onTurnAdded?.(payload as TutorialTurnAddedPayload);
      break;
    case "agent_response_chunk":
      opts.onAgentResponseChunk?.(payload as AgentResponseChunkPayload);
      break;
    case "tutorial_artifact_added":
      opts.onArtifactAdded?.(payload as TutorialArtifactAddedPayload);
      break;
    case "tutorial_artifact_deleted":
      opts.onArtifactDeleted?.(payload as TutorialArtifactDeletedPayload);
      break;
    case "session_completed":
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
