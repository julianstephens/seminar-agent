import { useApi } from "@/lib/ApiContext";
import type { ProblemSet } from "@/lib/types";
import {
  Badge,
  Button,
  Flex,
  Heading,
  HStack,
  Icon,
  Span,
  Spinner,
  Text,
  VStack,
} from "@chakra-ui/react";
import { useCallback, useEffect, useState } from "react";
import type { IconType } from "react-icons";
import {
  LuChevronRight,
  LuCircleAlert,
  LuCircleCheck,
  LuClock,
  LuPencilLine,
} from "react-icons/lu";
import { useNavigate, useParams } from "react-router-dom";

const statusColor: Record<string, string> = {
  reviewed: "green",
  assigned: "yellow",
  submitted: "blue",
  deleted: "red",
};

const statusIcon: Record<string, IconType> = {
  reviewed: LuCircleCheck,
  assigned: LuPencilLine,
  submitted: LuClock,
  deleted: LuCircleAlert,
};

const ProblemSetButton = ({ ps }: { ps: ProblemSet }) => {
  const navigate = useNavigate();
  const patterns = [
    ...new Set(
      ps.tasks.map((task) => task.pattern_code.replace("_", " ").toLowerCase()),
    ),
  ];

  return (
    <Button
      onClick={() =>
        navigate(`/tutorials/${ps.tutorial_id}/problem-sets/${ps.id}`)
      }
      id={`problemSet-${ps.id}`}
      key={ps.id}
      w="full"
      h="fit"
      justifyContent="start"
      alignItems="start"
      textAlign="left"
      bgColor="#1a1a1a"
      border="1px #333 solid"
      rounded="lg"
      color="white"
      _hover={{ bgColor: "#252525" }}
      transition="background-color 0.25s, color 0.25s"
      disabled={ps.status === "deleted"}
    >
      <VStack align="start" w="full">
        <HStack
          id="problemSetInfo"
          justify="space-between"
          align="start"
          w="full"
        >
          <VStack align="start">
            <Heading
              size="sm"
              fontWeight="bold"
            >{`Problem Set for ${ps.week_of}`}</Heading>
            <HStack color="fg.muted" fontSize="xs">
              <Text>
                Assigned {new Date(ps.created_at).toLocaleDateString()}
              </Text>
              <Span>•</Span>
              <Text>Assigned from {ps.assigned_from_session_id}</Text>
            </HStack>
          </VStack>
          <HStack>
            <Badge colorPalette={statusColor[ps.status] ?? ""}>
              <Icon w={4} h={4} as={statusIcon[ps.status] ?? LuCircleAlert} />
              {ps.status}
            </Badge>
            {ps.status !== "deleted" && (
              <Icon
                as={LuChevronRight}
                w={4}
                h={4}
                strokeWidth="1"
                color="fg.muted"
              />
            )}
          </HStack>
        </HStack>
        <VStack id="problemSetPatterns" w="full" mt={2} align="start">
          <Text fontSize="xs" color="fg.muted">
            Target Patterns:
          </Text>
          <HStack gap={2}>
            {patterns.map((pattern) => (
              <Badge
                size="xs"
                fontSize="xs"
                key={pattern}
                bgColor="#2a2a2a"
                color="#f59e0b"
                px={2}
                py={1}
              >
                {pattern}
              </Badge>
            ))}
          </HStack>
        </VStack>
      </VStack>
    </Button>
  );
};

const ProblemSetList = () => {
  const api = useApi();
  const params = useParams();
  const [problemSets, setProblemSets] = useState<ProblemSet[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Fetch problem sets for each tutorial
      const problemSetsArrays = await api
        .listTutorialProblemSets(params.id!)
        .catch((err) => {
          console.error("Error fetching problem sets:", err);
          return [] as ProblemSet[]; // Return empty array on error to prevent breaking the UI
        });

      // Flatten and sort by created_at (newest first)
      const allProblemSets = problemSetsArrays
        .flat()
        .sort(
          (a, b) =>
            new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
        );

      setProblemSets(allProblemSets);
    } catch (e) {
      setError(String(e));
    } finally {
      setLoading(false);
    }
  }, [api]);

  useEffect(() => {
    void load();
  }, [load]);

  if (loading) {
    return (
      <HStack justify="center" mt={20}>
        <Spinner size="xl" />
      </HStack>
    );
  }

  if (error) {
    return <Text color="red.500">Failed to load problem sets: {error}</Text>;
  }

  const assignedProblemSets = problemSets.filter(
    (ps) => ps.status === "assigned",
  ) as ProblemSet[];
  const submittedProblemSets = problemSets.filter(
    (ps) => ps.status === "submitted",
  ) as ProblemSet[];
  const reviewedProblemSets = problemSets.filter(
    (ps) => ps.status === "reviewed",
  ) as ProblemSet[];

  return (
    <>
      <Flex id="problemSetList" direction="column">
        <VStack
          id="problemSetListHeader"
          w="full"
          h="full"
          justify="start"
          align="start"
        >
          <Heading size="lg">Problem Sets</Heading>
          <Text color="fg.muted">
            Small corrective exercises designed to repair specific reasoning
            habits
          </Text>
        </VStack>

        {/* Assigned problem sets */}
        {assignedProblemSets.length > 0 && (
          <VStack
            id="assignedProblemSets"
            w="full"
            h="full"
            justify="start"
            align="start"
            gap={4}
            mt={10}
          >
            <Heading size="md">Active ({assignedProblemSets.length})</Heading>
            {assignedProblemSets.map((ps) => (
              <>
                <ProblemSetButton ps={ps} />
              </>
            ))}

            {/* Submitted problem sets */}
            {submittedProblemSets.length > 0 && (
              <VStack
                id="submittedProblemSets"
                w="full"
                h="full"
                justify="start"
                align="start"
                gap={4}
                mt={10}
              >
                <Heading size="md">
                  Pending Review ({submittedProblemSets.length})
                </Heading>
                {submittedProblemSets.map((ps) => (
                  <>
                    <ProblemSetButton ps={ps} />
                  </>
                ))}
              </VStack>
            )}

            {/* Reviewed problem sets */}
            {reviewedProblemSets.length > 0 && (
              <VStack
                id="reviewedProblemSets"
                w="full"
                h="full"
                justify="start"
                align="start"
                gap={4}
                mt={10}
              >
                <Heading size="md">
                  Complete ({reviewedProblemSets.length})
                </Heading>
                {reviewedProblemSets.map((ps) => (
                  <>
                    <ProblemSetButton key={ps.id} ps={ps} />
                  </>
                ))}
              </VStack>
            )}
          </VStack>
        )}
      </Flex>
    </>
  );
};

export default ProblemSetList;
