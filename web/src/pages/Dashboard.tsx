import { useSelectSeminarDialog } from "@/contexts/SelectSeminarDialogContext";
import { useSelectTutorialDialog } from "@/contexts/SelectTutorialDialogContext";
import { useApi } from "@/lib/ApiContext";
import type { Seminar, Session, Tutorial, TutorialSession } from "@/lib/types";
import {
  Badge,
  Button,
  Card,
  Flex,
  Heading,
  HStack,
  Spinner,
  Text,
  VStack,
} from "@chakra-ui/react";
import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

interface UnifiedSession {
  id: string;
  type: "seminar" | "tutorial";
  title: string;
  status: string;
  started_at: string;
  kind?: string; // For tutorial sessions
  seminar_id?: string;
  tutorial_id?: string;
}

const statusColorMap: Record<string, string> = {
  in_progress: "yellow",
  complete: "green",
  abandoned: "gray",
};

const statusLabelMap: Record<string, string> = {
  in_progress: "In Progress",
  complete: "Complete",
  abandoned: "Abandoned",
};

const Dashboard = () => {
  const api = useApi();
  const navigate = useNavigate();
  const { openDialog: openSelectSeminar } = useSelectSeminarDialog();
  const { openDialog: openSelectTutorial } = useSelectTutorialDialog();

  const [sessions, setSessions] = useState<UnifiedSession[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Fetch all seminars and tutorials
      const [seminars, tutorials] = await Promise.all([
        api.listSeminars(),
        api.listTutorials(),
      ]);

      // Fetch sessions for each seminar and tutorial
      const [seminarSessionsArrays, tutorialSessionsArrays] = await Promise.all(
        [
          Promise.all(
            seminars.map((s) =>
              api.listSessions(s.id).catch(() => [] as Session[]),
            ),
          ),
          Promise.all(
            tutorials.map((t) =>
              api
                .listTutorialSessions(t.id)
                .catch(() => [] as TutorialSession[]),
            ),
          ),
        ],
      );

      // Create maps for easy lookup
      const seminarMap = new Map<string, Seminar>(
        seminars.map((s) => [s.id, s]),
      );
      const tutorialMap = new Map<string, Tutorial>(
        tutorials.map((t) => [t.id, t]),
      );

      // Flatten and transform seminar sessions
      const unifiedSeminarSessions: UnifiedSession[] = seminarSessionsArrays
        .flat()
        .map((session) => {
          const seminar = seminarMap.get(session.seminar_id);
          return {
            id: session.id,
            type: "seminar" as const,
            title: seminar
              ? `${seminar.title} - ${session.section_label}`
              : session.section_label,
            status: session.status,
            started_at: session.started_at,
            seminar_id: session.seminar_id,
          };
        });

      // Flatten and transform tutorial sessions
      const unifiedTutorialSessions: UnifiedSession[] = tutorialSessionsArrays
        .flat()
        .map((session) => {
          const tutorial = tutorialMap.get(session.tutorial_id);
          return {
            id: session.id,
            type: "tutorial" as const,
            title: tutorial ? `${tutorial.title}` : `Tutorial Session`,
            status: session.status,
            started_at: session.started_at,
            tutorial_id: session.tutorial_id,
            kind: session.kind,
          };
        });

      // Combine and sort: in_progress first (by recent date), then complete/abandoned at bottom (by recent date)
      const combined = [...unifiedSeminarSessions, ...unifiedTutorialSessions]
        .sort((a, b) => {
          // Priority: in_progress > complete/abandoned
          const aIsActive = a.status === "in_progress";
          const bIsActive = b.status === "in_progress";

          if (aIsActive && !bIsActive) return -1;
          if (!aIsActive && bIsActive) return 1;

          // Within same priority group, sort by date (most recent first)
          return (
            new Date(b.started_at).getTime() - new Date(a.started_at).getTime()
          );
        })
        .slice(0, 5);

      setSessions(combined);
    } catch (e) {
      setError(String(e));
    } finally {
      setLoading(false);
    }
  }, [api]);

  useEffect(() => {
    void load();
  }, [load]);

  const handleSessionClick = (session: UnifiedSession) => {
    if (session.type === "seminar") {
      navigate(`/sessions/${session.id}`);
    } else {
      navigate(`/tutorial-sessions/${session.id}`);
    }
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleString("en-US", {
      month: "2-digit",
      day: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      hour12: true,
    });
  };

  return (
    <Flex direction="column" w="full" h="full">
      <HStack id="dashboardHeader" justify="space-between" w="full">
        <Heading size="lg">Recent Sessions</Heading>
        <HStack>
          <Button
            bg="#f59e0b"
            color="black"
            _hover={{ bg: "#fbbf24" }}
            onClick={openSelectTutorial}
          >
            Start Tutorial
          </Button>
          <Button
            bg="#f59e0b"
            color="black"
            _hover={{ bg: "#fbbf24" }}
            onClick={openSelectSeminar}
          >
            Start Seminar
          </Button>
        </HStack>
      </HStack>
      <VStack id="sessions" mt="6">
        {loading ? (
          <Spinner size="xl" mt={8} />
        ) : error ? (
          <Text color="red.500" mt={4}>
            {error}
          </Text>
        ) : sessions.length === 0 ? (
          <Text color="gray.500" mt={8}>
            No sessions yet. Start your first tutorial or seminar!
          </Text>
        ) : (
          sessions.map((session) => {
            const isClickable = session.status === "in_progress";
            return (
              <Card.Root
                key={session.id}
                maxW="lg"
                w="full"
                cursor={isClickable ? "pointer" : "default"}
                opacity={isClickable ? 1 : 0.6}
                _hover={isClickable ? { shadow: "md" } : {}}
                onClick={
                  isClickable ? () => handleSessionClick(session) : undefined
                }
              >
                <Card.Header
                  display="flex"
                  flexDir="row"
                  justifyContent="space-between"
                  alignItems="center"
                  w="full"
                >
                  <HStack gap={2}>
                    <Heading size="sm">{session.title}</Heading>
                    <Badge
                      textTransform="lowercase"
                      colorScheme={
                        session.type === "seminar" ? "purple" : "blue"
                      }
                    >
                      {session.type === "seminar" ? "Seminar" : "Tutorial"}
                    </Badge>
                    {session.kind && (
                      <Badge textTransform="lowercase" colorScheme="teal">
                        {session.kind}
                      </Badge>
                    )}
                  </HStack>
                  <Badge
                    colorPalette={statusColorMap[session.status] || "gray"}
                    w="fit"
                  >
                    {statusLabelMap[session.status] || session.status}
                  </Badge>
                </Card.Header>
                <Card.Body>
                  <Text fontSize="sm" color="fg.muted">
                    Started {formatDate(session.started_at)}
                  </Text>
                </Card.Body>
              </Card.Root>
            );
          })
        )}
      </VStack>
    </Flex>
  );
};

export default Dashboard;
