import { CopyTextButton } from "@/components/Button";
import type { Artifact, ProblemSet } from "@/lib/types";
import { prepareText } from "@/lib/utils";
import {
  Badge,
  Box,
  Button,
  Card,
  Dialog,
  Heading,
  HStack,
  Icon,
  IconButton,
  Text,
  VStack,
} from "@chakra-ui/react";
import { FaTrash } from "react-icons/fa";
import { LuPlus } from "react-icons/lu";

const artifactKindColor: Record<string, string> = {
  summary: "blue",
  notes: "green",
  problem_set: "orange",
  problem_set_response: "teal",
  diagnostic: "purple",
};

const ArtifactList = ({
  artifacts,
  isTerminal,
  onDelete,
}: {
  artifacts: Artifact[];
  isTerminal: boolean;
  onDelete: (artifact: Artifact) => void;
}) => {
  if (artifacts.length === 0) {
    return (
      <Text color="gray.500" fontSize="sm">
        No artifacts yet.
      </Text>
    );
  }

  return (
    <VStack id="artifactList" p={3} gap={3} overflowY="auto">
      {artifacts.map((a) => (
        <Card.Root id={`artifact-${a.id}`} key={a.id}>
          <Card.Body>
            <Box w="full" id={`artifact-${a.id}`} minW={0} flex={1}>
              <VStack
                id={`artifactHeader-${a.id}`}
                align="start"
                justify="start"
                mb={1}
              >
                <HStack justify="space-between" w="full">
                  <Badge colorPalette={artifactKindColor[a.kind] ?? "gray"}>
                    {a.kind.replace(/_/g, " ")}
                  </Badge>
                  <CopyTextButton textToCopy={a.content} />
                </HStack>
                <Text fontWeight="bold" wordBreak="break-word">
                  {a.title}
                </Text>
              </VStack>
              <Text
                fontSize="sm"
                color="gray.600"
                _dark={{ color: "gray.400" }}
                whiteSpace="pre-wrap"
                lineClamp={4}
              >
                {prepareText(a.content)}
              </Text>
              <HStack w="full" justify="space-between">
                <Text fontSize="xs" color="gray.400" mt={1}>
                  {new Date(a.created_at).toLocaleString()}
                </Text>
              </HStack>
            </Box>
            {!isTerminal && (
              <IconButton
                mt="4"
                size="xs"
                colorPalette="red"
                variant="outline"
                flexShrink={0}
                onClick={() => onDelete(a)}
              >
                <Icon>
                  <FaTrash />
                </Icon>
              </IconButton>
            )}
          </Card.Body>
        </Card.Root>
      ))}
    </VStack>
  );
};

interface ArtifactsDialogProps {
  isOpen: boolean;
  onClose: () => void;
  artifacts: Artifact[];
  isTerminal: boolean;
  onAdd: VoidFunction;
  onDelete: (artifact: Artifact) => void;
  problemSet?: ProblemSet | null;
  onDeleteProblemSet?: () => void;
}

export const ArtifactsDialog = ({
  isOpen,
  onClose,
  artifacts,
  isTerminal,
  onAdd,
  onDelete,
  problemSet,
  onDeleteProblemSet,
}: ArtifactsDialogProps) => {
  // Filter out deleted problem sets
  const visibleProblemSet =
    problemSet?.status === "deleted" ? null : problemSet;

  return (
    <Dialog.Root
      open={isOpen}
      onOpenChange={(d) => !d.open && onClose()}
      size={{ base: "cover", sm: "lg" }}
    >
      <Dialog.Backdrop />
      <Dialog.Positioner>
        <Dialog.Content
          h={{ base: "100vh", sm: "fit-content" }}
          maxH={{ base: "100vh", sm: "80vh" }}
        >
          <Dialog.Header borderBottom="1px" borderColor="gray.700">
            <HStack justify="space-between" w="full">
              <Heading size="md" fontWeight="bold">
                Session Details
              </Heading>
            </HStack>
          </Dialog.Header>
          <Dialog.Body p={3} h="fit-content" overflowY="auto">
            {/* Problem Set Section */}
            {visibleProblemSet && (
              <Box mb={4}>
                <Heading size="sm" mb={2}>
                  Problem Set
                </Heading>
                <Card.Root>
                  <Card.Body>
                    <VStack align="start" gap={2}>
                      <HStack justify="space-between" w="full">
                        <Badge colorPalette="blue">
                          {visibleProblemSet.status}
                        </Badge>
                        <Text fontSize="xs" color="gray.400">
                          Week of{" "}
                          {new Date(
                            visibleProblemSet.week_of,
                          ).toLocaleDateString()}
                        </Text>
                      </HStack>
                      <Text fontSize="sm" fontWeight="bold">
                        {visibleProblemSet.tasks.length} task
                        {visibleProblemSet.tasks.length !== 1 ? "s" : ""}{" "}
                        assigned
                      </Text>
                      {!isTerminal && onDeleteProblemSet && (
                        <IconButton
                          size="xs"
                          colorPalette="red"
                          variant="outline"
                          onClick={onDeleteProblemSet}
                        >
                          <Icon>
                            <FaTrash />
                          </Icon>
                        </IconButton>
                      )}
                    </VStack>
                  </Card.Body>
                </Card.Root>
              </Box>
            )}

            {/* Artifacts Section */}
            <Box>
              <HStack justify="space-between" mb={2}>
                <Heading size="sm">
                  Artifacts {artifacts.length > 0 && `(${artifacts.length})`}
                </Heading>
                {!isTerminal && (
                  <Button onClick={onAdd} className="primary" size="sm">
                    <Icon as={LuPlus} w={4} h={4} />
                    Add
                  </Button>
                )}
              </HStack>
              <ArtifactList
                artifacts={artifacts}
                isTerminal={isTerminal}
                onDelete={onDelete}
              />
            </Box>
          </Dialog.Body>
        </Dialog.Content>
      </Dialog.Positioner>
    </Dialog.Root>
  );
};
