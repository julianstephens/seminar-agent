import { CopyTextButton } from "@/components/Button";
import type { Artifact } from "@/lib/types";
import { prepareText } from "@/lib/utils";
import {
  Badge,
  Box,
  Button,
  Card,
  Flex,
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
    <VStack id="artifactList" p={3} gap={3}>
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
interface ArtifactPanelProps {
  artifacts: Artifact[];
  isTerminal: boolean;
  onAdd: VoidFunction;
  onDelete: (artifact: Artifact) => void;
}
export const ArtifactPanel = ({
  artifacts,
  isTerminal,
  onAdd,
  onDelete,
}: ArtifactPanelProps) => {
  return (
    <>
      <Flex
        id="artifactsPanel"
        flexDirection="column"
        w="80"
        h="full"
        bgColor="#0a0a0a"
        borderStart="1px #333 solid"
      >
        <Flex
          id="artifactsPanelHeader"
          p={4}
          borderBottom="1px #333 solid"
          align="center"
          justify="space-between"
        >
          <Heading size="md" color="white" fontWeight="bold">
            Artifacts {artifacts.length > 0 && `(${artifacts.length})`}
          </Heading>
          <Button onClick={onAdd} className="primary">
            <Icon as={LuPlus} w={4} h={4} />
            Add
          </Button>
        </Flex>
        <ArtifactList
          artifacts={artifacts}
          isTerminal={isTerminal}
          onDelete={onDelete}
        />
      </Flex>
    </>
  );
};
