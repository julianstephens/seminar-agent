/**
 * SessionEventsContext – manages SSE subscriptions globally per session.
 *
 * Keeps SSE connections alive across navigation so agent responses
 * aren't interrupted when the user moves to the review page.
 *
 * The connection is only aborted on explicit abandonment or session completion.
 */

import { useAccessToken } from "@/auth/useAuth";
import type {
  PhaseChangedPayload,
  SessionCompletedPayload,
  SseErrorPayload,
  TimerTickPayload,
  TurnAddedPayload,
  UseSessionEventsOptions,
} from "@/hooks/useSessionEvents";
import { createContext, useContext, useEffect, useRef } from "react";

const BASE_URL =
  ((import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "") + "/v1";

interface SessionSubscription {
  controller: AbortController;
  handlers: UseSessionEventsOptions;
}

interface SessionEventsContextType {
  subscribe: (sessionId: string, handlers: UseSessionEventsOptions) => void;
  unsubscribe: (sessionId: string) => void;
}

const SessionEventsContext = createContext<SessionEventsContextType | null>(
  null,
);

/**
 * Provider that manages SSE connections globally.
 * Should be placed at a high level in the app (above all session-related routes).
 */
export function SessionEventsProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const getToken = useAccessToken();
  const subscriptionsRef = useRef<Map<string, SessionSubscription>>(new Map());

  const subscribe = (sessionId: string, handlers: UseSessionEventsOptions) => {
    const subs = subscriptionsRef.current;

    // If already subscribed, update handlers and return.
    if (subs.has(sessionId)) {
      const existing = subs.get(sessionId)!;
      existing.handlers = handlers;
      return;
    }

    const controller = new AbortController();
    const cancelled = false;

    async function connect() {
      try {
        const token = await getToken();
        const res = await fetch(`${BASE_URL}/sessions/${sessionId}/events`, {
          headers: { Authorization: `Bearer ${token}` },
          signal: controller.signal,
        });

        if (!res.ok || !res.body) {
          handlers.onConnectionError?.(
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
          // Split on double-newline; retain the trailing incomplete chunk.
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
              // Ignore ": heartbeat" comments and blank lines.
            }

            if (dataLines.length === 0) continue;

            let payload: unknown;
            try {
              payload = JSON.parse(dataLines.join("\n")) as unknown;
            } catch {
              // Malformed JSON – skip this message.
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
        if (
          !cancelled &&
          !(e instanceof DOMException && e.name === "AbortError")
        ) {
          handlers.onConnectionError?.(e);
        }
      }
    }

    subs.set(sessionId, { controller, handlers });
    void connect();
  };

  const unsubscribe = (sessionId: string) => {
    const subs = subscriptionsRef.current;
    const sub = subs.get(sessionId);
    if (sub) {
      sub.controller.abort();
      subs.delete(sessionId);
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
    <SessionEventsContext.Provider value={{ subscribe, unsubscribe }}>
      {children}
    </SessionEventsContext.Provider>
  );
}

/**
 * Hook to subscribe to session events.
 * Updates handlers without re-opening the connection.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function useSessionEventsSubscription(
  sessionId: string | undefined,
  handlers: UseSessionEventsOptions,
): void {
  const ctx = useContext(SessionEventsContext);

  useEffect(() => {
    if (!sessionId || !ctx) return;
    ctx.subscribe(sessionId, handlers);
    // No cleanup here; connection persists across navigation.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionId, ctx]);
}

/**
 * Hook to explicitly close the SSE connection for a session.
 * Call this when abandoning a session.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function useSessionEventsUnsubscribe(): (sessionId: string) => void {
  const ctx = useContext(SessionEventsContext);
  return (sessionId: string) => {
    ctx?.unsubscribe(sessionId);
  };
}

// ── Internal dispatch ─────────────────────────────────────────────────────────

function dispatch(
  type: string,
  payload: unknown,
  opts: UseSessionEventsOptions,
): void {
  switch (type) {
    case "phase_changed":
      opts.onPhaseChanged?.(payload as PhaseChangedPayload);
      break;
    case "timer_tick":
      opts.onTimerTick?.(payload as TimerTickPayload);
      break;
    case "turn_added":
      opts.onTurnAdded?.(payload as TurnAddedPayload);
      break;
    case "session_completed":
      opts.onSessionCompleted?.(payload as SessionCompletedPayload);
      break;
    case "error":
      opts.onError?.(payload as SseErrorPayload);
      break;
    default:
      // Unknown event type – ignore.
      break;
  }
}
