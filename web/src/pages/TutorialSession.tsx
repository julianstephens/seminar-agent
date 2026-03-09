import { ChatInput } from "@/components/chat/ChatInput";
import { ArtifactsDialog } from "@/components/dialogs/ArtifactsDialog";
import { CreateArtifactDialog } from "@/components/dialogs/CreateArtifactDialog";
import { ArtifactPanel } from "@/components/tutorials/ArtifactPanel";
import { TutorialTurnList } from "@/components/tutorials/TurnList";
import { TutorialSessionActions } from "@/components/tutorials/TutorialSessionActions";
import { TutorialSessionHeader } from "@/components/tutorials/TutorialSessionHeader";
import { useCreateArtifactDialog } from "@/contexts/CreateArtifactDialogContext";
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
import { useParams } from "react-router-dom";

// ── Main component ────────────────────────────────────────────────────────────

const TutorialSessionRunner = () => {
  const { id } = useParams<{ id: string }>();
  console.log("[TutorialSessionRunner] Component rendered, id:", id);
  const api = useApi();
  const unsubscribe = useTutorialSessionEventsUnsubscribe();

  // Artifact dialog
  const artifactDialog = useCreateArtifactDialog();
  const [creatingArtifact, setCreatingArtifact] = useState(false);
  const [artifactsDialogOpen, setArtifactsDialogOpen] = useState(false);

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

  // Failed turns (turn IDs that failed to get agent response)
  const [failedTurns, setFailedTurns] = useState<Set<string>>(new Set());

  // Session lifecycle
  const [completing, setCompleting] = useState(false);
  const [abandoning, setAbandoning] = useState(false);
  const [showCompleteForm, setShowCompleteForm] = useState(false);

  // Refs
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
      // Populate failedTurns from backend data
      const failed = new Set<string>();
      for (const turn of session.turns ?? []) {
        if (turn.failed) {
          failed.add(turn.id);
        }
      }
      setFailedTurns(failed);
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
      // Mark turn as failed if the backend indicates it
      if (turn.failed) {
        setFailedTurns((prev) => new Set(prev).add(turn.id));
      }
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
    onError: ({ message }) => {
      setStreamError(message);
      // Mark the last user turn as failed and clear any streaming state
      setTurns((prev) => {
        const lastUserTurn = [...prev]
          .reverse()
          .find((t) => t.speaker === "user");
        if (lastUserTurn) {
          setFailedTurns((failed) => new Set(failed).add(lastUserTurn.id));
        }
        return prev;
      });
      // Clear streaming state since the agent call failed
      setStreamingTurns(new Map());
    },
    onConnectionError: (e) => {
      console.warn("[SSE] tutorial connection error", e);
    },
  });

  // ── Handlers ────────────────────────────────────────────────────────────────

  const handleSubmitTurn = async (text: string) => {
    if (!id || !text) return;

    // Add optimistic user turn immediately
    const optimisticId = `optimistic-${Date.now()}`;
    const optimisticUserTurn: TutorialTurn = {
      id: optimisticId,
      session_id: id,
      speaker: "user",
      text,
      failed: false,
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
      // Keep the optimistic turn visible but mark it as failed.
      setFailedTurns((prev) => new Set(prev).add(optimisticId));
      if (e instanceof ApiRequestError) {
        // e.message is the error code (e.g. "validation_error"); the
        // human-readable detail is nested in e.detail.
        const body = e.detail as Record<string, unknown> | null | undefined;
        const details = body?.details as
          | Record<string, unknown>
          | null
          | undefined;
        const msg =
          (typeof details?.message === "string" && details.message) ||
          (typeof body?.message === "string" && body.message) ||
          e.message;
        setTurnError(msg);
      } else {
        setTurnError(String(e));
      }
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

  const handleCreateArtifact = async (problemSetId?: string) => {
    if (!id) return;
    const title = artifactDialog.titleRef.current?.value.trim();
    const content = artifactDialog.contentRef.current?.value.trim();

    if (!title || !content) {
      setError("Title and content are required");
      return;
    }

    setCreatingArtifact(true);
    setError(null);

    try {
      const artifact = await api.createArtifact(id, {
        kind: artifactDialog.kind,
        title,
        content,
        problem_set_id: problemSetId,
      });
      setDetail((prev) =>
        prev ? { ...prev, artifacts: [...prev.artifacts, artifact] } : prev,
      );

      // Reset form
      if (artifactDialog.titleRef.current) {
        artifactDialog.titleRef.current.value = "";
      }
      if (artifactDialog.contentRef.current) {
        artifactDialog.contentRef.current.value = "";
      }

      // Close dialog only if "create another" is not checked
      if (!artifactDialog.createAnother) {
        artifactDialog.closeDialog();
      }
    } catch (e) {
      setError(e instanceof ApiRequestError ? e.message : String(e));
    } finally {
      setCreatingArtifact(false);
    }
  };

  const handleDeleteProblemSet = async () => {
    if (!id || !detail?.problem_set) return;
    if (!window.confirm("Delete this problem set?")) return;

    try {
      await api.deleteSessionProblemSet(id);
      setDetail((prev) => (prev ? { ...prev, problem_set: undefined } : prev));
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
          toBack={`/tutorials/${detail.tutorial_id}`}
          toExport={`/tutorial-sessions/${id}/export`}
          onOpenArtifacts={() => setArtifactsDialogOpen(true)}
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
            failedTurns={failedTurns}
            bottomRef={bottomRef}
          />
        </Box>
        <ChatInput
          onSend={(message) => void handleSubmitTurn(message)}
          disabled={isTerminal}
          sessionKind={detail.kind}
          artifacts={detail.artifacts ?? []}
        />

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
          onAdd={artifactDialog.openDialog}
          onDelete={handleDeleteArtifact}
        />
      </Box>
      <CreateArtifactDialog
        isOpen={artifactDialog.isOpen}
        onClose={artifactDialog.closeDialog}
        titleRef={artifactDialog.titleRef}
        contentRef={artifactDialog.contentRef}
        kind={artifactDialog.kind}
        setKind={artifactDialog.setKind}
        creating={creatingArtifact}
        handleCreate={(problemSetId) => void handleCreateArtifact(problemSetId)}
        createAnother={artifactDialog.createAnother}
        setCreateAnother={artifactDialog.setCreateAnother}
        problemSet={detail.problem_set}
      />
      <ArtifactsDialog
        isOpen={artifactsDialogOpen}
        onClose={() => setArtifactsDialogOpen(false)}
        artifacts={detail.artifacts}
        isTerminal={isTerminal}
        onAdd={artifactDialog.openDialog}
        onDelete={handleDeleteArtifact}
        problemSet={detail.problem_set}
        onDeleteProblemSet={handleDeleteProblemSet}
      />
    </Flex>
  );
};

export default TutorialSessionRunner;
