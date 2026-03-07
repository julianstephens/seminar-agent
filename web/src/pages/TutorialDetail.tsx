import { DeleteButton, ExportButton } from "@/components/Button";
import { NewTutorialSessionDialog } from "@/components/dialogs/NewTutorialSessionDialog";
import { ApiRequestError } from "@/lib/api";
import { useApi } from "@/lib/ApiContext";
import type {
  Tutorial,
  TutorialSession,
  TutorialSessionKind,
} from "@/lib/types";
import {
  Badge,
  Box,
  Button,
  Card,
  Heading,
  HStack,
  Spinner,
  Stack,
  Text,
} from "@chakra-ui/react";
import { useCallback, useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

const statusColor: Record<string, string> = {
  in_progress: "blue",
  complete: "green",
  abandoned: "gray",
};

export default function TutorialDetail() {
  const { id } = useParams<{ id: string }>();
  const api = useApi();
  const navigate = useNavigate();

  const [tutorial, setTutorial] = useState<Tutorial | null>(null);
  const [sessions, setSessions] = useState<TutorialSession[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [starting, setStarting] = useState(false);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [selectedKind, setSelectedKind] = useState<TutorialSessionKind | null>(
    null,
  );

  const load = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    setError(null);
    try {
      setTutorial(await api.getTutorial(id));
    } catch (e) {
      setError(e instanceof ApiRequestError ? e.message : String(e));
    } finally {
      setLoading(false);
    }
    try {
      setSessions(await api.listTutorialSessions(id));
    } catch {
      // non-fatal
    }
  }, [id, api]);

  useEffect(() => {
    void load();
  }, [load]);

  const handleDelete = async () => {
    if (!id || !window.confirm("Delete this tutorial? This cannot be undone."))
      return;
    try {
      await api.deleteTutorial(id);
      navigate("/tutorials", { replace: true });
    } catch (e) {
      setError(String(e));
    }
  };

  const handleOpenDialog = () => {
    setSelectedKind(null);
    setDialogOpen(true);
  };

  const handleStartSession = async () => {
    if (!id) return;
    setStarting(true);
    setDialogOpen(false);
    try {
      const sess = await api.createTutorialSession(id, {
        kind: selectedKind ?? undefined,
      });
      navigate(`/tutorial-sessions/${sess.id}`);
    } catch (e) {
      setError(String(e));
    } finally {
      setStarting(false);
    }
  };

  const handleDeleteSession = async (sessionId: string) => {
    if (!window.confirm("Delete this session? This cannot be undone.")) return;
    try {
      await api.deleteTutorialSession(sessionId);
      setSessions((prev) => prev.filter((s) => s.id !== sessionId));
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

  if (!tutorial) {
    return <Text color="red.500">{error ?? "Tutorial not found."}</Text>;
  }

  return (
    <Box
      id="tutorial"
      maxW={{ base: "100vw", md: "4xl" }}
      w={{ md: "full" }}
      mx={{ md: "auto" }}
      pt={6}
    >
      {/* Header */}
      <HStack
        id="tutorialHeader"
        mb={2}
        justify="space-between"
        align="start"
        wrap="wrap"
        gap={2}
      >
        <Box minW={0} flex={1}>
          <Heading size="lg" wordBreak="break-word">
            {tutorial.title}
          </Heading>
          <Text color="gray.500" fontSize="sm">
            {tutorial.subject}
          </Text>
        </Box>
        <HStack gap={2} wrap="wrap" flexShrink={0}>
          <Badge colorPalette="purple">{tutorial.difficulty}</Badge>
          <ExportButton to={`/tutorials/${id}/export`} />
          <DeleteButton onClick={handleDelete} />
        </HStack>
      </HStack>

      {tutorial.description && (
        <Card.Root mb={6} p={4}>
          <Text fontSize="sm" color="gray.600" _dark={{ color: "gray.400" }}>
            {tutorial.description}
          </Text>
        </Card.Root>
      )}

      {error && (
        <Text color="red.500" mb={4}>
          {error}
        </Text>
      )}

      {/* Sessions */}
      <Box id="tutorialSessions">
        <HStack mb={4} justify="space-between">
          <Text fontWeight="medium">{sessions.length} session(s)</Text>
          <Button className="primary" size="sm" onClick={handleOpenDialog}>
            Start Session
          </Button>
        </HStack>

        {sessions.length === 0 ? (
          <Text color="gray.500">No sessions yet. Start your first one!</Text>
        ) : (
          <Stack gap={3}>
            {sessions.map((s) => (
              <Card.Root key={s.id} _hover={{ shadow: "sm" }}>
                <Card.Body>
                  <HStack justify="space-between" wrap="wrap" gap={2}>
                    <Box
                      minW={0}
                      flex={1}
                      cursor="pointer"
                      onClick={() => navigate(`/tutorial-sessions/${s.id}`)}
                    >
                      <HStack gap={2}>
                        <Text
                          fontWeight="medium"
                          textTransform="capitalize"
                          wordBreak="break-word"
                        >
                          {s.kind ? `${s.kind} Tutorial` : "Tutorial"}
                        </Text>
                      </HStack>
                      <Text fontSize="xs" color="gray.500">
                        {new Date(s.started_at).toLocaleDateString()}
                        {s.ended_at &&
                          ` → ${new Date(s.ended_at).toLocaleDateString()}`}
                      </Text>
                    </Box>
                    <HStack gap={2} flexShrink={0}>
                      <Badge colorScheme={statusColor[s.status] ?? "gray"}>
                        {s.status}
                      </Badge>
                      <ExportButton to={`/tutorial-sessions/${s.id}/export`} />
                      <DeleteButton
                        onClick={(e) => {
                          e.stopPropagation();
                          void handleDeleteSession(s.id);
                        }}
                      />
                    </HStack>
                  </HStack>
                </Card.Body>
              </Card.Root>
            ))}
          </Stack>
        )}
      </Box>

      <NewTutorialSessionDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        selectedKind={selectedKind}
        onKindChange={setSelectedKind}
        starting={starting}
        onStart={handleStartSession}
      />
    </Box>
  );
}
