/**
 * SessionRunner – interactive turn-submission page.
 *
 * Implements Phase 12:
 *   - SSE subscription via useSessionEvents (backend-driven countdown + events)
 *   - Live MM:SS countdown sourced from timer_tick events
 *   - Phase-lock (disables input during phase transitions)
 *   - Appends turns received via turn_added without a manual refresh
 *   - Redirects on session_completed event
 *   - Inline MISSING_LOCATOR error with locator format helper
 *
 * Pre-existing behaviour retained:
 *   - Loading/submitting user turns and appending agent replies
 *   - Residue submission when phase === "residue_required"
 *   - Abandoning a session
 */

import {
  useSessionEventsSubscription,
  useSessionEventsUnsubscribe,
} from "@/contexts/SessionEventsContext";
import type {
  PhaseChangedPayload,
  TimerTickPayload,
  TurnAddedPayload,
} from "@/hooks/useSessionEvents";
import { ApiRequestError } from "@/lib/api";
import { useApi } from "@/lib/ApiContext";
import type { SessionDetail, SessionPhase, Turn } from "@/lib/types";
import {
  Alert,
  Badge,
  Box,
  Button,
  Card,
  Code,
  Heading,
  HStack,
  Spinner,
  Text,
  Textarea,
  VStack,
} from "@chakra-ui/react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

const PHASE_LABELS: Record<SessionPhase, string> = {
  reconstruction: "Reconstruction",
  opposition: "Opposition",
  reversal: "Reversal",
  residue_required: "Residue Required",
  done: "Done",
};

const PHASE_COLOR: Record<SessionPhase, string> = {
  reconstruction: "blue",
  opposition: "orange",
  reversal: "purple",
  residue_required: "red",
  done: "green",
};

const SeminarSessionRunner = () => {
  const { id } = useParams<{ id: string }>();
  const api = useApi();
  const navigate = useNavigate();
  const unsubscribe = useSessionEventsUnsubscribe();

  const [session, setSession] = useState<SessionDetail | null>(null);
  const [turns, setTurns] = useState<Turn[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  /** Seconds remaining in the current phase, driven by SSE timer_tick. */
  const [secondsRemaining, setSecondsRemaining] = useState<number | null>(null);
  /** True while a phase_changed event is being processed; disables the input. */
  const [phaseLocked, setPhaseLocked] = useState(false);
  /** Set when the backend returns missing_locator (400) on a turn submit. */
  const [locatorError, setLocatorError] = useState<string | null>(null);

  const turnRef = useRef<HTMLTextAreaElement>(null);
  const residueRef = useRef<HTMLTextAreaElement>(null);
  const bottomRef = useRef<HTMLDivElement>(null);

  const load = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    setError(null);
    try {
      const detail = await api.getSession(id);
      setSession(detail);
      setTurns(detail.turns ?? []);
    } catch (e) {
      setError(e instanceof ApiRequestError ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [id, api]);

  useEffect(() => {
    void load();
  }, [load]);

  // Auto-scroll to bottom when turns update.
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [turns]);

  // Redirect to review if session is already completed/abandoned.
  useEffect(() => {
    if (session && session.status !== "in_progress") {
      navigate(`/sessions/${id}/review`, { replace: true });
    }
  }, [session, id, navigate]);

  // ── SSE subscription ────────────────────────────────────────────────────────

  useSessionEventsSubscription(id, {
    onTimerTick: (payload: TimerTickPayload) => {
      setSecondsRemaining(Math.ceil(payload.seconds_remaining));
    },

    onPhaseChanged: (payload: PhaseChangedPayload) => {
      // Lock input briefly while we refresh the session state.
      setPhaseLocked(true);
      setSecondsRemaining(
        payload.phase_ends_at
          ? Math.max(
              0,
              Math.ceil(
                (new Date(payload.phase_ends_at).getTime() - Date.now()) / 1000,
              ),
            )
          : null,
      );
      setSession((prev) =>
        prev
          ? {
              ...prev,
              phase: payload.phase,
              phase_ends_at: payload.phase_ends_at ?? prev.phase_ends_at,
            }
          : prev,
      );
      setPhaseLocked(false);
      setLocatorError(null);
    },

    onTurnAdded: (payload: TurnAddedPayload) => {
      setTurns((prev) => {
        if (prev.some((t) => t.id === payload.turn.id)) return prev;
        return [...prev, payload.turn];
      });
    },

    onSessionCompleted: () => {
      if (id) unsubscribe(id); // Close SSE connection when session completes
      navigate(`/sessions/${id}/review`, { replace: true });
    },

    onError: (payload) => {
      setError(payload.message);
    },

    onConnectionError: (e) => {
      console.warn("[SSE] connection error", e);
    },
  });

  const handleSubmitTurn = async () => {
    if (!id || !turnRef.current) return;
    const text = turnRef.current.value.trim();
    if (!text) return;
    setSubmitting(true);
    setError(null);
    setLocatorError(null);
    try {
      const agentTurn = await api.submitTurn(id, text);
      // SSE turn_added will append both user and agent turns; clear the field.
      // Still do a local dedup-append as a fallback for clients without SSE.
      setTurns((prev) => {
        const alreadyHas = prev.some((t) => t.id === agentTurn.id);
        return alreadyHas ? prev : [...prev, agentTurn];
      });
      turnRef.current.value = "";
    } catch (e) {
      if (e instanceof ApiRequestError && e.message === "missing_locator") {
        const detail = e.detail as { message?: string } | undefined;
        setLocatorError(
          detail?.message ??
            "Claim must include a text locator or be marked UNANCHORED.",
        );
      } else {
        const msg = e instanceof ApiRequestError ? e.message : String(e);
        setError(msg);
      }
    } finally {
      setSubmitting(false);
    }
  };

  const handleSubmitResidue = async () => {
    if (!id || !residueRef.current) return;
    const text = residueRef.current.value.trim();
    if (!text) return;
    setSubmitting(true);
    setError(null);
    try {
      await api.submitResidue(id, text);
      navigate(`/sessions/${id}/review`, { replace: true });
    } catch (e) {
      setError(e instanceof ApiRequestError ? e.message : String(e));
    } finally {
      setSubmitting(false);
    }
  };

  const handleAbandon = async () => {
    if (!id || !window.confirm("Abandon this session?")) return;
    try {
      await api.abandonSession(id);
      unsubscribe(id); // Close SSE connection when abandoning
      navigate(`/sessions/${id}/review`, { replace: true });
    } catch (e) {
      setError(String(e));
    }
  };

  if (loading) {
    return (
      <HStack justify="center" mt={20}>
        <Spinner size="xl" />
      </HStack>
    );
  }

  if (!session) {
    return <Text color="red.500">{error ?? "Session not found."}</Text>;
  }

  const phase = session.phase;
  const isResiduePhase = phase === "residue_required";
  const isDone = phase === "done";
  const canSubmitTurns = !isResiduePhase && !isDone;
  // Derive thinking state from turns array so it persists across navigation
  const agentThinking =
    turns.length > 0 && turns[turns.length - 1].speaker === "user";

  return (
    <Box maxW="3xl" mx="auto" w="full">
      {/* Phase banner */}
      <Card.Root
        mb={4}
        bg={`${PHASE_COLOR[phase]}.50`}
        _dark={{ bg: `${PHASE_COLOR[phase]}.900` }}
      >
        <Card.Body>
          <HStack justify="space-between" wrap="wrap" gap={2}>
            <HStack gap={3}>
              <Heading size="md">{session.section_label}</Heading>
              <Badge
                colorScheme={PHASE_COLOR[phase]}
                fontSize="sm"
                px={2}
                py={1}
              >
                {PHASE_LABELS[phase]}
              </Badge>
            </HStack>
            <HStack gap={2}>
              <Button
                size="xs"
                variant="outline"
                colorScheme="red"
                onClick={handleAbandon}
              >
                Abandon
              </Button>
            </HStack>
          </HStack>
          {/* Backend-driven countdown from SSE timer_tick events */}
          {!isResiduePhase && !isDone && (
            <Text
              fontSize="sm"
              fontVariantNumeric="tabular-nums"
              color="gray.600"
              mt={1}
            >
              {secondsRemaining !== null
                ? `${String(Math.floor(secondsRemaining / 60)).padStart(2, "0")}:${String(secondsRemaining % 60).padStart(2, "0")} remaining`
                : session.phase_ends_at
                  ? `Phase ends ${new Date(session.phase_ends_at).toLocaleTimeString()}`
                  : null}
            </Text>
          )}
          {phaseLocked && (
            <HStack mt={1} gap={1}>
              <Spinner size="xs" />
              <Text fontSize="xs" color="gray.400">
                Phase transitioning…
              </Text>
            </HStack>
          )}
        </Card.Body>
      </Card.Root>

      {error && (
        <Text color="red.500" mb={3}>
          {error}
        </Text>
      )}

      {/* MISSING_LOCATOR inline error + locator format guide */}
      {locatorError && (
        <Alert.Root status="warning" mb={3} rounded="md">
          <Alert.Indicator />
          <Alert.Content>
            <Alert.Title>Locator required</Alert.Title>
            <Alert.Description>
              {locatorError} Include one of the following formats in your claim,
              or prefix the sentence with <Code>UNANCHORED</Code>:
              <HStack mt={1} gap={2} wrap="wrap">
                {[
                  "p. 12",
                  "pp. 12–15",
                  "ch. 3",
                  "§4",
                  "scene 3",
                  "para. 7",
                  "¶7",
                  "l. 12",
                ].map((ex) => (
                  <Code key={ex} fontSize="xs">
                    {ex}
                  </Code>
                ))}
              </HStack>
            </Alert.Description>
          </Alert.Content>
        </Alert.Root>
      )}

      {/* Transcript */}
      <Box
        borderWidth={1}
        rounded="md"
        p={4}
        minH={{ base: "180px", md: "300px" }}
        maxH={{ base: "40vh", md: "55vh" }}
        overflowY="auto"
        mb={4}
        bg="gray.50"
        _dark={{ bg: "gray.800" }}
      >
        {turns.length === 0 ? (
          <Text color="gray.400" textAlign="center" mt={8}>
            Transcript will appear here as the session progresses.
          </Text>
        ) : (
          <VStack align="stretch" gap={3}>
            {turns
              .filter((t) => t.text?.trim())
              .map((t) => {
                const isUser = t.speaker === "user";
                return (
                  <Box
                    key={t.id}
                    p={3}
                    borderLeft="4px solid"
                    borderLeftColor={isUser ? "blue.500" : "teal.500"}
                    bg={isUser ? "blue.50" : "teal.50"}
                    _dark={{ bg: isUser ? "blue.900" : "teal.900" }}
                    rounded="md"
                  >
                    <HStack mb={2} gap={2} wrap="wrap">
                      <Badge
                        colorScheme={isUser ? "blue" : "teal"}
                        size="md"
                        fontWeight="bold"
                      >
                        {isUser ? "👤 You" : "🤖 Agent"}
                      </Badge>
                      <Badge colorScheme="gray" size="sm">
                        {t.phase}
                      </Badge>
                      {t.flags?.length > 0 &&
                        t.flags.map((f) => (
                          <Badge key={f} colorScheme="red" size="sm">
                            {f}
                          </Badge>
                        ))}
                    </HStack>
                    <Text fontSize="sm" whiteSpace="pre-wrap" lineHeight="1.6">
                      {t.text}
                    </Text>
                  </Box>
                );
              })}
            {agentThinking && (
              <HStack gap={2} w="full" px={3} py={2}>
                <Spinner size="sm" />
                <Text fontSize="sm" color="gray.500" fontStyle="italic">
                  Agent is thinking…
                </Text>
              </HStack>
            )}
          </VStack>
        )}
        <div ref={bottomRef} />
      </Box>

      {/* Input area */}
      {isResiduePhase ? (
        <VStack gap={3} align="stretch">
          <Text fontWeight="medium">
            Submit your residue reflection (5–7 sentences with
            thesis/objection/tension components):
          </Text>
          <Textarea ref={residueRef} rows={6} placeholder="Your residue…" />
          <Button
            colorScheme="red"
            loading={submitting}
            onClick={handleSubmitResidue}
          >
            Submit Residue
          </Button>
        </VStack>
      ) : isDone ? (
        <Button
          colorScheme="green"
          w="full"
          onClick={() => navigate(`/sessions/${id}/review`)}
        >
          View Review
        </Button>
      ) : (
        canSubmitTurns && (
          <VStack gap={2} align="stretch">
            <Textarea
              ref={turnRef}
              rows={3}
              placeholder="Your turn…"
              disabled={submitting || phaseLocked}
              onKeyDown={(e) => {
                if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
                  void handleSubmitTurn();
                }
              }}
            />
            <HStack justify="space-between" wrap="wrap" gap={2}>
              <Text fontSize="xs" color="gray.400">
                ⌘/Ctrl+Enter to submit
              </Text>
              <Button
                className="primary"
                loading={submitting}
                disabled={phaseLocked}
                w={{ base: "full", sm: "auto" }}
                onClick={handleSubmitTurn}
              >
                Submit
              </Button>
            </HStack>
          </VStack>
        )
      )}
    </Box>
  );
};

export default SeminarSessionRunner;
