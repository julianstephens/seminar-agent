import { ChatInput } from "@/components/chat/ChatInput";
import { ArtifactPanel } from "@/components/tutorials/ArtifactPanel";
import { TutorialTurnList } from "@/components/tutorials/TurnList";
import { TutorialSessionActions } from "@/components/tutorials/TutorialSessionActions";
import { TutorialSessionHeader } from "@/components/tutorials/TutorialSessionHeader";
import {
  useTutorialSessionEventsSubscription,
  useTutorialSessionEventsUnsubscribe,
} from "@/contexts/TutorialSessionEventsContext";
import { ApiRequestError } from "@/lib/api";
import { useApi } from "@/lib/ApiContext";
import type {
  Artifact,
  TutorialSessionDetail,
  TutorialTurn,
} from "@/lib/types";
import {
  Box,
  Card,
  Flex,
  Heading,
  HStack,
  Spinner,
  Text,
  VStack,
} from "@chakra-ui/react";
import { useCallback, useEffect, useRef, useState } from "react";
import { flushSync } from "react-dom";
import { useNavigate, useParams } from "react-router-dom";

// ── Main component ────────────────────────────────────────────────────────────

const TutorialSessionRunner = () => {
  const { id } = useParams<{ id: string }>();
  console.log("[TutorialSessionRunner] Component rendered, id:", id);
  const api = useApi();
  const navigate = useNavigate();
  const unsubscribe = useTutorialSessionEventsUnsubscribe();

  // Session + artifacts
  const [detail, setDetail] = useState<TutorialSessionDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Turns (own state so they can be appended incrementally)
  const [turns, setTurns] = useState<TutorialTurn[]>([]);
  const [, setSubmittingTurn] = useState(false);
  const [turnError, setTurnError] = useState<string | null>(null);
  const [streamError, setStreamError] = useState<string | null>(null);

  // Streaming agent responses (turn_id -> accumulated text)
  const [streamingTurns, setStreamingTurns] = useState<Map<string, string>>(
    new Map(),
  );

  // Session lifecycle
  const [completing, setCompleting] = useState(false);
  const [abandoning, setAbandoning] = useState(false);
  const [showCompleteForm, setShowCompleteForm] = useState(false);

  // Refs
  const composerRef = useRef<HTMLTextAreaElement | null>(null);
  const bottomRef = useRef<HTMLDivElement | null>(null);
  const notesRef = useRef<HTMLTextAreaElement | null>(null);

  const load = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    setError(null);
    try {
      const session = await api.getTutorialSession(id);
      setDetail(session);
      setTurns(session.turns ?? []);
    } catch (e) {
      setError(e instanceof ApiRequestError ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [id, api]);

  useEffect(() => {
    void load();
  }, [load]);

  // Auto-scroll to bottom when turns update or streaming chunks arrive.
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [turns, streamingTurns]);

  // ── SSE subscription ────────────────────────────────────────────────────────

  console.log(
    "[TutorialSessionRunner] Setting up SSE subscription for id:",
    id,
  );
  useTutorialSessionEventsSubscription(id, {
    onTurnAdded: ({ turn }) => {
      console.log("[TutorialSessionRunner] onTurnAdded callback:", turn);
      setTurns((prev) =>
        prev.some((t) => t.id === turn.id) ? prev : [...prev, turn],
      );
    },
    onAgentResponseChunk: ({ turn_id, chunk, is_final }) => {
      console.log("[TutorialSessionRunner] onAgentResponseChunk callback:", {
        turn_id,
        chunk: chunk.substring(0, 20),
        is_final,
      });
      if (is_final) {
        // Clear streaming state for this turn when final chunk arrives
        setStreamingTurns((prev) => {
          const next = new Map(prev);
          next.delete(turn_id);
          return next;
        });
      } else {
        // Accumulate chunk - use flushSync to force immediate render for streaming effect
        flushSync(() => {
          setStreamingTurns((prev) => {
            const next = new Map(prev);
            const current = next.get(turn_id) ?? "";
            next.set(turn_id, current + chunk);
            return next;
          });
        });
      }
    },
    onArtifactAdded: ({ artifact }) => {
      setDetail((prev) =>
        prev ? { ...prev, artifacts: [...prev.artifacts, artifact] } : prev,
      );
    },
    onArtifactDeleted: ({ artifact_id }) => {
      setDetail((prev) =>
        prev
          ? {
              ...prev,
              artifacts: prev.artifacts.filter((a) => a.id !== artifact_id),
            }
          : prev,
      );
    },
    onSessionCompleted: () => {
      setDetail((prev) => (prev ? { ...prev, status: "complete" } : prev));
      if (id) unsubscribe(id);
    },
    onError: ({ message }) => setStreamError(message),
    onConnectionError: (e) => {
      console.warn("[SSE] tutorial connection error", e);
    },
  });

  // ── Handlers ────────────────────────────────────────────────────────────────

  const handleSubmitTurn = async () => {
    if (!id || !composerRef.current) return;
    const text = composerRef.current.value.trim();
    if (!text) return;

    // Clear input and add optimistic user turn immediately
    composerRef.current.value = "";
    const optimisticId = `optimistic-${Date.now()}`;
    const optimisticUserTurn: TutorialTurn = {
      id: optimisticId,
      session_id: id,
      speaker: "user",
      text,
      created_at: new Date().toISOString(),
    };
    setTurns((prev) => [...prev, optimisticUserTurn]);

    setSubmittingTurn(true);
    setTurnError(null);

    try {
      const response = await api.submitTutorialTurn(id, text);
      setTurns((prev) => {
        // Replace optimistic turn with real user turn, add agent turn if present
        const withoutOptimistic = prev.filter((t) => t.id !== optimisticId);
        const newTurns = [response.user_turn];
        if (response.agent_turn) {
          newTurns.push(response.agent_turn);
        }
        // Only add turns that don't already exist
        const filtered = newTurns.filter(
          (t) => !withoutOptimistic.some((existing) => existing.id === t.id),
        );
        return [...withoutOptimistic, ...filtered];
      });
    } catch (e) {
      // Remove optimistic turn on error
      setTurns((prev) => prev.filter((t) => t.id !== optimisticId));
      setTurnError(e instanceof ApiRequestError ? e.message : String(e));
    } finally {
      setSubmittingTurn(false);
    }
  };

  const handleComplete = async () => {
    if (!id) return;
    setCompleting(true);
    try {
      const notes = notesRef.current?.value.trim() ?? "";
      const updated = await api.completeTutorialSession(id, notes);
      setDetail((prev) => (prev ? { ...prev, ...updated } : null));
      setShowCompleteForm(false);
      if (id) unsubscribe(id);
    } catch (e) {
      setError(e instanceof ApiRequestError ? e.message : String(e));
    } finally {
      setCompleting(false);
    }
  };

  const handleAbandon = async () => {
    if (!id || !window.confirm("Abandon this session?")) return;
    setAbandoning(true);
    try {
      const updated = await api.abandonTutorialSession(id);
      setDetail((prev) => (prev ? { ...prev, ...updated } : null));
      unsubscribe(id);
    } catch (e) {
      setError(e instanceof ApiRequestError ? e.message : String(e));
    } finally {
      setAbandoning(false);
    }
  };

  const handleExport = () => {
    if (!id) return;
    navigate(`/tutorial-sessions/${id}/export`);
  };

  const handleDeleteArtifact = async (artifact: Artifact) => {
    if (!id || !window.confirm(`Delete artifact "${artifact.title}"?`)) return;
    try {
      await api.deleteArtifact(id, artifact.id);
      setDetail((prev) =>
        prev
          ? {
              ...prev,
              artifacts: prev.artifacts.filter((a) => a.id !== artifact.id),
            }
          : null,
      );
    } catch (e) {
      setError(e instanceof ApiRequestError ? e.message : String(e));
    }
  };

  // ── Render ──────────────────────────────────────────────────────────────────

  if (loading) {
    return (
      <HStack justify="center" mt={20}>
        <Spinner size="xl" />
      </HStack>
    );
  }

  if (!detail) {
    return <Text color="red.500">{error ?? "Session not found."}</Text>;
  }

  const isTerminal =
    detail.status === "complete" || detail.status === "abandoned";

  return (
    <Flex w="full">
      <VStack flexGrow={1} h="full">
        {/* A. Header */}
        <TutorialSessionHeader
          detail={detail}
          onBack={() => navigate(`/tutorials/${detail.tutorial_id}`)}
          onExport={handleExport}
        />

        {/* E. Error banners */}
        {error && (
          <Text color="red.500" mb={4}>
            {error}
          </Text>
        )}
        {turnError && (
          <Text color="red.500" mb={4}>
            {turnError}
          </Text>
        )}
        {streamError && (
          <Text color="orange.500" fontSize="sm" mb={4}>
            ⚠ Stream error: {streamError}
          </Text>
        )}

        {/* Session notes (shown when complete) */}
        {detail.notes && (
          <Card.Root
            mb={4}
            p={4}
            borderLeft="4px solid"
            borderColor="green.400"
          >
            <Text fontSize="sm" fontStyle="italic">
              <strong>Session notes:</strong> {detail.notes}
            </Text>
          </Card.Root>
        )}

        {/* Abandoned banner */}
        {detail.status === "abandoned" && (
          <Text color="gray.500" fontSize="sm" mb={4} fontStyle="italic">
            This session has been abandoned.
          </Text>
        )}

        <Box
          id="conversationContainer"
          maxW={{ base: "100vw", md: "4xl" }}
          w={{ md: "full" }}
          mx={{ base: "4", md: "auto" }}
          display={{ md: "flex" }}
          flexDir="column"
          gap={6}
          alignItems="flex-start"
        >
          <Heading size="sm" mb={3}>
            Conversation
          </Heading>
          <TutorialTurnList
            turns={turns}
            streamingTurns={streamingTurns}
            bottomRef={bottomRef}
          />
        </Box>
        <ChatInput
          onSend={() => void handleSubmitTurn()}
          disabled={isTerminal}
        />
        {/* C. Artifact panel (side) */}
        {/* <Box w={{ base: "full", md: "340px" }} flexShrink={0}>
          <HStack mb={3} justify="space-between">
            <Heading size="sm">Artifacts ({detail.artifacts.length})</Heading>
            {!isTerminal && (
              <Button
                size="sm"
                bg="#f59e0b"
                color="black"
                _hover={{ bg: "#fbbf24" }}
                onClick={() => setShowArtifactForm((v) => !v)}
              >
                {showArtifactForm ? "Cancel" : "Add"}
              </Button>
            )}
          </HStack>

          {showArtifactForm && (
            <ArtifactComposer
              onSave={() => void handleCreateArtifact()}
              onCancel={() => setShowArtifactForm(false)}
              saving={creatingArtifact}
              artifactKind={artifactKind}
              setArtifactKind={setArtifactKind}
              titleRef={artifactTitleRef}
              contentRef={artifactContentRef}
            />
          )}

          <ArtifactList
            artifacts={detail.artifacts}
            isTerminal={isTerminal}
            onDelete={(a) => void handleDeleteArtifact(a)}
          />
        </Box> */}
        {/* D. Completion controls */}
        {!isTerminal && (
          <TutorialSessionActions
            onComplete={() => void handleComplete()}
            onAbandon={() => void handleAbandon()}
            completing={completing}
            abandoning={abandoning}
            showCompleteForm={showCompleteForm}
            onToggleCompleteForm={() => setShowCompleteForm((v) => !v)}
            notesRef={notesRef}
          />
        )}
      </VStack>
      <Box display={{ base: "none", lg: "block" }}>
        <ArtifactPanel
          artifacts={detail.artifacts}
          isTerminal={isTerminal}
          onAdd={() => {}}
          onDelete={handleDeleteArtifact}
        />
      </Box>
    </Flex>
  );
};

export default TutorialSessionRunner;
