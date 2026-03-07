import { DeleteButton, ExportButton } from "@/components/Button";
import { useEditSeminarDialog } from "@/contexts/EditSeminarDialogContext";
import { useNewSessionDialog } from "@/contexts/NewSessionDialogContext";
import { ApiRequestError } from "@/lib/api";
import { useApi } from "@/lib/ApiContext";
import type { Seminar, Session } from "@/lib/types";
import {
  Badge,
  Box,
  Button,
  Card,
  DataList,
  Heading,
  HStack,
  Spinner,
  Stack,
  Text,
} from "@chakra-ui/react";
import { useCallback, useEffect, useState } from "react";
import { LuPencil } from "react-icons/lu";
import { useNavigate, useParams } from "react-router-dom";

export default function SeminarDetail() {
  const { id } = useParams<{ id: string }>();
  const api = useApi();
  const navigate = useNavigate();

  const [seminar, setSeminar] = useState<Seminar | null>(null);
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Use edit dialog hook
  const { openDialog: openEditDialog, callbackRef: editCallbackRef } =
    useEditSeminarDialog();

  // Use new session dialog hook
  const { openDialog: openNewSessionDialog, seminarIdRef } =
    useNewSessionDialog();

  const load = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    setError(null);
    try {
      setSeminar(await api.getSeminar(id));
    } catch (e) {
      setError(e instanceof ApiRequestError ? e.message : String(e));
    } finally {
      setLoading(false);
    }
    // Sessions are non-fatal — load separately so a sessions error
    // doesn't prevent the seminar from rendering.
    try {
      setSessions(await api.listSessions(id));
    } catch {
      // non-fatal
    }
  }, [id, api]);

  useEffect(() => {
    void load();
  }, [load]);

  // Register callbacks for dialogs
  useEffect(() => {
    editCallbackRef.current = (updated) => {
      setSeminar(updated);
    };
    return () => {
      editCallbackRef.current = null;
    };
  }, [editCallbackRef]);

  useEffect(() => {
    if (id) {
      seminarIdRef.current = id;
    }
  }, [id, seminarIdRef]);

  const handleDelete = async () => {
    if (!id || !window.confirm("Delete this seminar? This cannot be undone."))
      return;
    try {
      await api.deleteSeminar(id);
      navigate("/seminars", { replace: true });
    } catch (e) {
      setError(String(e));
    }
  };

  const handleDeleteSession = async (
    sessionId: string,
    sectionLabel: string,
  ) => {
    if (
      !window.confirm(
        `Delete session "${sectionLabel}"? This cannot be undone.`,
      )
    )
      return;
    try {
      await api.deleteSession(sessionId);
      setSessions(sessions.filter((s) => s.id !== sessionId));
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

  if (!seminar) {
    return <Text color="red.500">{error ?? "Seminar not found."}</Text>;
  }

  const statusColor: Record<string, string> = {
    in_progress: "blue",
    complete: "green",
    abandoned: "gray",
  };

  return (
    <>
      <Box
        id="seminar"
        maxW={{ base: "100vw", md: "4xl" }}
        w={{ md: "full" }}
        mx={{ md: "auto" }}
        pt={6}
      >
        {/* Header */}
        <HStack
          id="seminarHeader"
          mb={2}
          justify="space-between"
          align="start"
          wrap="wrap"
          gap={2}
        >
          <Box minW={0} flex={1}>
            <DataList.Root>
              <DataList.Item>
                <DataList.ItemLabel>Thesis:</DataList.ItemLabel>
                <DataList.ItemValue>
                  {seminar.thesis_current}
                </DataList.ItemValue>
              </DataList.Item>
            </DataList.Root>
            <Heading size="lg" wordBreak="break-word">
              {seminar.title}
            </Heading>
            {seminar.author && (
              <Text color="gray.500" fontSize="sm">
                {seminar.author}
              </Text>
            )}
            {seminar.edition_notes && (
              <Text color="gray.500" fontSize="sm">
                Edition: {seminar.edition_notes}
              </Text>
            )}
          </Box>
          <HStack gap={2} wrap="wrap" flexShrink={0}>
            <Button
              className="grey"
              size="sm"
              variant="outline"
              onClick={() => seminar && openEditDialog(seminar)}
            >
              <LuPencil />
              Edit
            </Button>

            <ExportButton to={`/seminars/${id}/export`} />
            <DeleteButton onClick={handleDelete} />
          </HStack>
        </HStack>

        <Card.Root mb={6} p={4}>
          <Text
            fontSize="sm"
            fontStyle="italic"
            color="gray.600"
            _dark={{ color: "gray.400" }}
          >
            <strong>Thesis:</strong> {seminar.thesis_current}
          </Text>
        </Card.Root>

        {error && (
          <Text color="red.500" mb={4}>
            {error}
          </Text>
        )}

        {/* Sessions */}
        <Box id="seminarSessions">
          <HStack mb={4} justify="space-between">
            <Text fontWeight="medium">{sessions.length} session(s)</Text>
            <Button
              bg="#f59e0b"
              color="black"
              _hover={{ bg: "#fbbf24" }}
              size="sm"
              onClick={openNewSessionDialog}
            >
              New Session
            </Button>
          </HStack>

          {sessions.length === 0 ? (
            <Text color="gray.500">No sessions yet.</Text>
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
                        onClick={() =>
                          s.status === "in_progress"
                            ? navigate(`/sessions/${s.id}`)
                            : navigate(`/sessions/${s.id}/review`)
                        }
                      >
                        <Text fontWeight="medium" wordBreak="break-word">
                          {s.section_label}
                        </Text>
                        <Text fontSize="xs" color="gray.500">
                          {new Date(s.started_at).toLocaleDateString()}
                          {" — "}
                          {s.phase}
                        </Text>
                      </Box>
                      <HStack gap={2} flexShrink={0}>
                        <Badge colorScheme={statusColor[s.status] ?? "gray"}>
                          {s.status}
                        </Badge>
                        <ExportButton to={`/sessions/${s.id}/export`} />
                        <DeleteButton
                          onClick={(e) => {
                            e.stopPropagation();
                            handleDeleteSession(s.id, s.section_label);
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
      </Box>
    </>
  );
}
