import {
  useTutorialSessionEventsSubscription,
  useTutorialSessionEventsUnsubscribe,
} from "@/contexts/TutorialSessionEventsContext";
import { ApiRequestError } from "@/lib/api";
import { useApi } from "@/lib/ApiContext";
import type {
  Artifact,
  ArtifactKind,
  TutorialSessionDetail,
  TutorialTurn,
} from "@/lib/types";
import {
  Badge,
  Box,
  Button,
  Card,
  Heading,
  HStack,
  Icon,
  IconButton,
  Spinner,
  Stack,
  Text,
  Textarea,
  VStack,
} from "@chakra-ui/react";
import { useCallback, useEffect, useRef, useState } from "react";
import { FaTrash } from "react-icons/fa";
import { useNavigate, useParams } from "react-router-dom";

const artifactKindColor: Record<string, string> = {
  summary: "blue",
  notes: "green",
  problem_set: "orange",
  diagnostic: "purple",
};

const ARTIFACT_KINDS = ["summary", "notes", "problem_set", "diagnostic"] as const satisfies readonly ArtifactKind[];

// ── Sub-components ────────────────────────────────────────────────────────────

function TutorialSessionHeader({
  detail,
  onBack,
}: {
  detail: TutorialSessionDetail;
  onBack: () => void;
}) {
  return (
    <HStack mb={4} justify="space-between" align="start" wrap="wrap" gap={2}>
      <Box>
        <Heading size="md">Tutorial Session</Heading>
        <Text fontSize="sm" color="gray.500">
          Started {new Date(detail.started_at).toLocaleString()}
        </Text>
        {detail.kind && (
          <Text fontSize="xs" color="gray.400" mt={0.5}>
            {detail.kind.replace(/_/g, " ")}
          </Text>
        )}
      </Box>
      <HStack gap={2} flexShrink={0}>
        <Badge
          colorScheme={
            detail.status === "complete"
              ? "green"
              : detail.status === "abandoned"
                ? "gray"
                : "blue"
          }
        >
          {detail.status}
        </Badge>
        <Button size="sm" variant="outline" onClick={onBack}>
          ← Back
        </Button>
      </HStack>
    </HStack>
  );
}

function TutorialTurnList({
  turns,
  bottomRef,
}: {
  turns: TutorialTurn[];
  bottomRef: React.RefObject<HTMLDivElement | null>;
}) {
  const agentThinking =
    turns.length > 0 && turns[turns.length - 1].speaker === "user";

  return (
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
          Conversation will appear here. Submit a message to get started.
        </Text>
      ) : (
        <VStack align="stretch" gap={3}>
          {turns
            .filter((t) => t.text?.trim())
            .map((t) => {
              const isUser = t.speaker === "user";
              const isSystem = t.speaker === "system";
              return (
                <Box
                  key={t.id}
                  p={3}
                  borderLeft="4px solid"
                  borderLeftColor={
                    isUser ? "blue.500" : isSystem ? "gray.400" : "teal.500"
                  }
                  bg={
                    isUser ? "blue.50" : isSystem ? "gray.100" : "teal.50"
                  }
                  _dark={{
                    bg: isUser ? "blue.900" : isSystem ? "gray.700" : "teal.900",
                  }}
                  rounded="md"
                  opacity={isSystem ? 0.8 : 1}
                >
                  <HStack mb={2} gap={2} wrap="wrap">
                    <Badge
                      colorScheme={isUser ? "blue" : isSystem ? "gray" : "teal"}
                      size="md"
                      fontWeight="bold"
                    >
                      {isUser ? "👤 You" : isSystem ? "⚙ System" : "🤖 Agent"}
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
                  <Text fontSize="xs" color="gray.400" mt={1}>
                    {new Date(t.created_at).toLocaleTimeString()}
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
  );
}

function TutorialComposer({
  onSubmit,
  submitting,
  disabled,
  turnError,
  composerRef,
}: {
  onSubmit: () => void;
  submitting: boolean;
  disabled: boolean;
  turnError: string | null;
  composerRef: React.RefObject<HTMLTextAreaElement | null>;
}) {
  return (
    <VStack gap={2} align="stretch">
      {turnError && (
        <Text color="red.500" fontSize="sm">
          {turnError}
        </Text>
      )}
      <Textarea
        ref={composerRef}
        rows={3}
        placeholder="Your message…"
        disabled={submitting || disabled}
        onKeyDown={(e) => {
          if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
            onSubmit();
          }
        }}
      />
      <HStack justify="space-between" wrap="wrap" gap={2}>
        <Text fontSize="xs" color="gray.400">
          ⌘/Ctrl+Enter to submit
        </Text>
        <Button
          bg="#f59e0b"
          color="black"
          _hover={{ bg: "#fbbf24" }}
          loading={submitting}
          disabled={disabled}
          w={{ base: "full", sm: "auto" }}
          onClick={onSubmit}
        >
          Send
        </Button>
      </HStack>
    </VStack>
  );
}

function ArtifactList({
  artifacts,
  isTerminal,
  onDelete,
}: {
  artifacts: Artifact[];
  isTerminal: boolean;
  onDelete: (artifact: Artifact) => void;
}) {
  if (artifacts.length === 0) {
    return (
      <Text color="gray.500" fontSize="sm">
        No artifacts yet.
      </Text>
    );
  }

  return (
    <Stack gap={3}>
      {artifacts.map((a) => (
        <Card.Root key={a.id} _hover={{ shadow: "sm" }}>
          <Card.Body>
            <HStack justify="space-between" align="start" wrap="wrap" gap={2}>
              <Box minW={0} flex={1}>
                <HStack gap={2} mb={1}>
                  <Badge colorScheme={artifactKindColor[a.kind] ?? "gray"}>
                    {a.kind.replace(/_/g, " ")}
                  </Badge>
                  <Text fontWeight="medium" wordBreak="break-word">
                    {a.title}
                  </Text>
                </HStack>
                <Text
                  fontSize="sm"
                  color="gray.600"
                  _dark={{ color: "gray.400" }}
                  whiteSpace="pre-wrap"
                  lineClamp={4}
                >
                  {a.content}
                </Text>
                <Text fontSize="xs" color="gray.400" mt={1}>
                  {new Date(a.created_at).toLocaleString()}
                </Text>
              </Box>
              {!isTerminal && (
                <IconButton
                  size="xs"
                  colorScheme="red"
                  variant="outline"
                  flexShrink={0}
                  onClick={() => onDelete(a)}
                >
                  <Icon>
                    <FaTrash />
                  </Icon>
                </IconButton>
              )}
            </HStack>
          </Card.Body>
        </Card.Root>
      ))}
    </Stack>
  );
}

function ArtifactComposer({
  onSave,
  onCancel,
  saving,
  artifactKind,
  setArtifactKind,
  titleRef,
  contentRef,
}: {
  onSave: () => void;
  onCancel: () => void;
  saving: boolean;
  artifactKind: ArtifactKind;
  setArtifactKind: (k: ArtifactKind) => void;
  titleRef: React.RefObject<HTMLInputElement | null>;
  contentRef: React.RefObject<HTMLTextAreaElement | null>;
}) {
  return (
    <Card.Root mb={4} p={4}>
      <VStack align="stretch" gap={3}>
        <select
          value={artifactKind}
          onChange={(e) => setArtifactKind(e.target.value as ArtifactKind)}
          style={{ padding: "6px 10px", border: "1px solid #ccc", borderRadius: 4 }}
        >
          {ARTIFACT_KINDS.map((k) => (
            <option key={k} value={k}>
              {k.replace(/_/g, " ")}
            </option>
          ))}
        </select>
        <input
          ref={titleRef}
          placeholder="Title *"
          style={{ padding: "6px 10px", border: "1px solid #ccc", borderRadius: 4 }}
        />
        <Textarea ref={contentRef} placeholder="Content *" rows={6} />
        <HStack gap={2}>
          <Button
            bg="#f59e0b"
            color="black"
            _hover={{ bg: "#fbbf24" }}
            loading={saving}
            onClick={onSave}
          >
            Save Artifact
          </Button>
          <Button variant="outline" onClick={onCancel}>
            Cancel
          </Button>
        </HStack>
      </VStack>
    </Card.Root>
  );
}

function TutorialSessionActions({
  onComplete,
  onAbandon,
  completing,
  abandoning,
  showCompleteForm,
  onToggleCompleteForm,
  notesRef,
}: {
  onComplete: () => void;
  onAbandon: () => void;
  completing: boolean;
  abandoning: boolean;
  showCompleteForm: boolean;
  onToggleCompleteForm: () => void;
  notesRef: React.RefObject<HTMLTextAreaElement | null>;
}) {
  return (
    <Box mt={4}>
      <HStack gap={3} wrap="wrap">
        <Button
          bg="#f59e0b"
          color="black"
          _hover={{ bg: "#fbbf24" }}
          onClick={onToggleCompleteForm}
        >
          {showCompleteForm ? "Cancel" : "Complete Session"}
        </Button>
        <Button
          variant="outline"
          colorScheme="red"
          loading={abandoning}
          onClick={onAbandon}
        >
          Abandon
        </Button>
      </HStack>

      {showCompleteForm && (
        <Card.Root mt={4} p={4}>
          <VStack align="stretch" gap={3}>
            <Text fontWeight="medium">Session Notes (optional)</Text>
            <Textarea ref={notesRef} placeholder="Add any final notes..." rows={4} />
            <Button
              bg="#f59e0b"
              color="black"
              _hover={{ bg: "#fbbf24" }}
              loading={completing}
              onClick={onComplete}
            >
              Confirm Complete
            </Button>
          </VStack>
        </Card.Root>
      )}
    </Box>
  );
}

// ── Main component ────────────────────────────────────────────────────────────

export default function TutorialSessionRunner() {
  const { id } = useParams<{ id: string }>();
  const api = useApi();
  const navigate = useNavigate();
  const unsubscribe = useTutorialSessionEventsUnsubscribe();

  // Session + artifacts
  const [detail, setDetail] = useState<TutorialSessionDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Turns (own state so they can be appended incrementally)
  const [turns, setTurns] = useState<TutorialTurn[]>([]);
  const [submittingTurn, setSubmittingTurn] = useState(false);
  const [turnError, setTurnError] = useState<string | null>(null);
  const [streamError, setStreamError] = useState<string | null>(null);

  // Session lifecycle
  const [completing, setCompleting] = useState(false);
  const [abandoning, setAbandoning] = useState(false);
  const [showCompleteForm, setShowCompleteForm] = useState(false);

  // Artifact creation
  const [showArtifactForm, setShowArtifactForm] = useState(false);
  const [artifactKind, setArtifactKind] = useState<ArtifactKind>("notes");
  const [creatingArtifact, setCreatingArtifact] = useState(false);

  // Refs
  const composerRef = useRef<HTMLTextAreaElement | null>(null);
  const bottomRef = useRef<HTMLDivElement | null>(null);
  const notesRef = useRef<HTMLTextAreaElement | null>(null);
  const artifactTitleRef = useRef<HTMLInputElement | null>(null);
  const artifactContentRef = useRef<HTMLTextAreaElement | null>(null);

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

  // Auto-scroll to bottom when turns update.
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [turns]);

  // ── SSE subscription ────────────────────────────────────────────────────────

  useTutorialSessionEventsSubscription(id, {
    onTurnAdded: ({ turn }) => {
      setTurns((prev) =>
        prev.some((t) => t.id === turn.id) ? prev : [...prev, turn],
      );
    },
    onArtifactAdded: ({ artifact }) => {
      setDetail((prev) =>
        prev
          ? { ...prev, artifacts: [...prev.artifacts, artifact] }
          : prev,
      );
    },
    onArtifactDeleted: ({ artifact_id }) => {
      setDetail((prev) =>
        prev
          ? { ...prev, artifacts: prev.artifacts.filter((a) => a.id !== artifact_id) }
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

    setSubmittingTurn(true);
    setTurnError(null);

    try {
      const agentTurn = await api.submitTutorialTurn(id, text);
      setTurns((prev) => {
        const exists = prev.some((t) => t.id === agentTurn.id);
        return exists ? prev : [...prev, agentTurn];
      });
      composerRef.current.value = "";
    } catch (e) {
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

  const handleCreateArtifact = async () => {
    if (!id) return;
    const title = artifactTitleRef.current?.value.trim() ?? "";
    const content = artifactContentRef.current?.value.trim() ?? "";
    if (!title || !content) return;

    setCreatingArtifact(true);
    try {
      const artifact = await api.createArtifact(id, {
        kind: artifactKind,
        title,
        content,
      });
      setDetail((prev) =>
        prev ? { ...prev, artifacts: [...prev.artifacts, artifact] } : null,
      );
      setShowArtifactForm(false);
      if (artifactTitleRef.current) artifactTitleRef.current.value = "";
      if (artifactContentRef.current) artifactContentRef.current.value = "";
    } catch (e) {
      setError(e instanceof ApiRequestError ? e.message : String(e));
    } finally {
      setCreatingArtifact(false);
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
    <Box maxW="5xl" mx="auto" w="full">
      {/* A. Header */}
      <TutorialSessionHeader detail={detail} onBack={() => navigate(-1)} />

      {/* E. Error banners */}
      {error && (
        <Text color="red.500" mb={4}>
          {error}
        </Text>
      )}
      {streamError && (
        <Text color="orange.500" fontSize="sm" mb={4}>
          ⚠ Stream error: {streamError}
        </Text>
      )}

      {/* Session notes (shown when complete) */}
      {detail.notes && (
        <Card.Root mb={4} p={4} borderLeft="4px solid" borderColor="green.400">
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
        display={{ md: "flex" }}
        gap={6}
        alignItems="flex-start"
      >
        {/* B. Conversation panel (primary) */}
        <Box flex={1} minW={0}>
          <Heading size="sm" mb={3}>
            Conversation
          </Heading>
          <TutorialTurnList turns={turns} bottomRef={bottomRef} />

          {/* Composer */}
          {!isTerminal && (
            <TutorialComposer
              onSubmit={() => void handleSubmitTurn()}
              submitting={submittingTurn}
              disabled={isTerminal}
              turnError={turnError}
              composerRef={composerRef}
            />
          )}
        </Box>

        {/* C. Artifact panel (side) */}
        <Box w={{ base: "full", md: "340px" }} flexShrink={0}>
          <HStack mb={3} justify="space-between">
            <Heading size="sm">
              Artifacts ({detail.artifacts.length})
            </Heading>
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
        </Box>
      </Box>

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
    </Box>
  );
}
