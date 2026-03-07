/**
 * TutorialSessionEventsContext – manages tutorial SSE subscriptions globally.
 *
 * Mirrors the pattern used by SessionEventsContext for seminar sessions,
 * but uses tutorial-specific endpoints and event types only.
 * Never touches seminar components, seminar contexts, or seminar endpoints.
 */

import { useAccessToken } from "@/auth/useAuth";
import type {
  TutorialArtifactAddedPayload,
  TutorialArtifactDeletedPayload,
  TutorialSessionCompletedPayload,
  TutorialSseErrorPayload,
  TutorialTurnAddedPayload,
  UseTutorialSessionEventsOptions,
} from "@/hooks/useTutorialSessionEvents";
import { createContext, useContext, useEffect, useRef } from "react";

const BASE_URL =
  ((import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "") + "/v1";

interface TutorialSessionSubscription {
  controller: AbortController;
  handlers: UseTutorialSessionEventsOptions;
}

interface TutorialSessionEventsContextType {
  subscribe: (
    sessionId: string,
    handlers: UseTutorialSessionEventsOptions,
  ) => void;
  unsubscribe: (sessionId: string) => void;
}

const TutorialSessionEventsContext =
  createContext<TutorialSessionEventsContextType | null>(null);

/**
 * Provider that manages tutorial SSE connections globally.
 * Should be placed at a high level in the app (above all tutorial-session routes).
 */
export function TutorialSessionEventsProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const getToken = useAccessToken();
  const subscriptionsRef = useRef<Map<string, TutorialSessionSubscription>>(
    new Map(),
  );

  const subscribe = (
    sessionId: string,
    handlers: UseTutorialSessionEventsOptions,
  ) => {
    console.log("[SSE Context] subscribe called for session:", sessionId);
    const subs = subscriptionsRef.current;

    // If already subscribed, update handlers and return.
    if (subs.has(sessionId)) {
      console.log("[SSE Context] Already subscribed, updating handlers");
      const existing = subs.get(sessionId)!;
      existing.handlers = handlers;
      return;
    }

    console.log("[SSE Context] Creating new subscription");
    const controller = new AbortController();
    let cancelled = false;

    async function connect() {
      try {
        console.log(
          "[SSE Context] Connecting to:",
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

        console.log(
          "[SSE Context] Response status:",
          res.status,
          res.statusText,
        );
        if (!res.ok || !res.body) {
          console.error(
            "[SSE Context] Connection failed:",
            res.status,
            res.statusText,
          );
          handlers.onConnectionError?.(
            new Error(`SSE connect failed: ${res.status} ${res.statusText}`),
          );
          return;
        }

        console.log(
          "[SSE Context] Connection established, starting to read stream",
        );
        const reader = res.body.getReader();
        const decoder = new TextDecoder();
        let buf = "";

        while (!cancelled) {
          const { done, value } = await reader.read();
          if (done) {
            console.log("[SSE Context] Stream ended (done=true)");
            break;
          }

          buf += decoder.decode(value, { stream: true });

          const parts = buf.split("\n\n");
          buf = parts.pop() ?? "";

          for (const part of parts) {
            // Skip heartbeat comments
            if (part.trim().startsWith(":")) {
              console.log("[SSE Context] Received heartbeat");
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
              console.log("[SSE Context] Received event:", eventType, payload);
            } catch (e) {
              console.error(
                "[SSE Context] Failed to parse event data:",
                dataLines.join("\n"),
                e,
              );
              continue;
            }

            dispatch(
              eventType,
              payload,
              subscriptionsRef.current.get(sessionId)?.handlers || handlers,
            );
          }
        }
      } catch (e) {
        if (!(e instanceof DOMException && e.name === "AbortError")) {
          console.error("[SSE Context] Connection error:", e);
          handlers.onConnectionError?.(e);
        } else {
          console.log("[SSE Context] Connection aborted");
        }
      } finally {
        cancelled = true;
        console.log("[SSE Context] Connection cleanup");
      }
    }

    subs.set(sessionId, { controller, handlers });
    console.log("[SSE Context] Calling connect()");
    void connect();
  };

  const unsubscribe = (sessionId: string) => {
    console.log("[SSE Context] unsubscribe called for session:", sessionId);
    const subs = subscriptionsRef.current;
    const sub = subs.get(sessionId);
    if (sub) {
      sub.controller.abort();
      subs.delete(sessionId);
      console.log("[SSE Context] Subscription removed");
    }
  };

  // Cleanup all subscriptions on unmount.
  useEffect(() => {
    const subs = subscriptionsRef.current;
    return () => {
      subs.forEach((sub) => {
        sub.controller.abort();
      });
      subs.clear();
    };
  }, []);

  return (
    <TutorialSessionEventsContext.Provider value={{ subscribe, unsubscribe }}>
      {children}
    </TutorialSessionEventsContext.Provider>
  );
}

/**
 * Hook to subscribe to tutorial session events.
 * Updates handlers without re-opening the connection.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function useTutorialSessionEventsSubscription(
  sessionId: string | undefined,
  handlers: UseTutorialSessionEventsOptions,
): void {
  const ctx = useContext(TutorialSessionEventsContext);
  console.log(
    "[useTutorialSessionEventsSubscription] Hook called with sessionId:",
    sessionId,
    "ctx:",
    ctx,
  );

  useEffect(() => {
    console.log(
      "[useTutorialSessionEventsSubscription] useEffect called with sessionId:",
      sessionId,
      "ctx:",
      ctx,
    );
    if (!sessionId || !ctx) return;
    ctx.subscribe(sessionId, handlers);
    // No cleanup here; connection persists across navigation.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionId, ctx]);
}

/**
 * Hook to explicitly close the SSE connection for a tutorial session.
 * Call this when completing or abandoning a session.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function useTutorialSessionEventsUnsubscribe(): (
  sessionId: string,
) => void {
  const ctx = useContext(TutorialSessionEventsContext);
  return (sessionId: string) => {
    ctx?.unsubscribe(sessionId);
  };
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
