import type { ProblemSet } from "@/lib/types";
import {
  Badge,
  Box,
  Card,
  Flex,
  Heading,
  Icon,
  IconButton,
  Text,
  VStack,
} from "@chakra-ui/react";
import { FaTrash } from "react-icons/fa";

const patternCodeColor: Record<string, string> = {
  TEXT_DRIFT: "blue",
  CONCEPT_DRIFT: "purple",
  FEEDBACK_LOOP: "green",
  HALLUCINATION: "red",
  PROMPT_INJECTION: "orange",
  CONTEXT_OVERFLOW: "yellow",
};

const statusColor: Record<string, string> = {
  assigned: "blue",
  submitted: "green",
  reviewed: "purple",
};

interface ProblemSetPanelProps {
  problemSet: ProblemSet | null | undefined;
  isTerminal: boolean;
  onDelete: (ps: ProblemSet) => void;
}

export const ProblemSetPanel = ({
  problemSet,
  isTerminal,
  onDelete,
}: ProblemSetPanelProps) => {
  // Don't show deleted problem sets
  if (problemSet?.status === "deleted") {
    problemSet = null;
  }

  return (
    <Flex
      id="problemSetPanel"
      flexDirection="column"
      w="80"
      h="full"
      bgColor="#0a0a0a"
      borderStart="1px #333 solid"
    >
      <Flex
        id="problemSetPanelHeader"
        p={4}
        borderBottom="1px #333 solid"
        align="center"
        justify="space-between"
      >
        <Heading size="md" color="white" fontWeight="bold">
          Problem Set
        </Heading>
      </Flex>

      {!problemSet ? (
        <Box p={4}>
          <Text color="gray.500" fontSize="sm">
            No problem set assigned for this session.
          </Text>
        </Box>
      ) : (
        <VStack id="problemSetContent" p={3} gap={3} overflowY="auto">
          {/* Problem Set Header */}
          <Card.Root w="full">
            <Card.Body>
              <Flex justify="space-between" align="center" mb={2}>
                <Badge colorPalette={statusColor[problemSet.status] ?? "gray"}>
                  {problemSet.status}
                </Badge>
                <Text fontSize="xs" color="gray.400">
                  Week of {new Date(problemSet.week_of).toLocaleDateString()}
                </Text>
              </Flex>
              {problemSet.review_notes && (
                <Text fontSize="sm" color="gray.600" mt={2}>
                  <strong>Review:</strong> {problemSet.review_notes}
                </Text>
              )}
              {!isTerminal && problemSet.status !== "deleted" && (
                <IconButton
                  mt={4}
                  size="xs"
                  colorPalette="red"
                  variant="outline"
                  onClick={() => onDelete(problemSet)}
                >
                  <Icon>
                    <FaTrash />
                  </Icon>
                </IconButton>
              )}
            </Card.Body>
          </Card.Root>

          {/* Task Cards */}
          {problemSet.tasks.map((task, idx) => (
            <Card.Root key={idx} w="full">
              <Card.Body>
                <VStack align="start" gap={2}>
                  <Flex justify="space-between" w="full">
                    <Text fontWeight="bold" fontSize="md">
                      {task.title}
                    </Text>
                    <Badge
                      colorPalette={
                        patternCodeColor[task.pattern_code] ?? "gray"
                      }
                    >
                      {task.pattern_code}
                    </Badge>
                  </Flex>
                  {task.description && (
                    <Text fontSize="sm" color="gray.600">
                      {task.description}
                    </Text>
                  )}
                  {task.prompt && (
                    <Box
                      p={3}
                      bg="gray.50"
                      _dark={{ bg: "gray.900" }}
                      borderRadius="md"
                      w="full"
                    >
                      <Text fontSize="sm" whiteSpace="pre-wrap">
                        {task.prompt}
                      </Text>
                    </Box>
                  )}
                </VStack>
              </Card.Body>
            </Card.Root>
          ))}
        </VStack>
      )}
    </Flex>
  );
};
