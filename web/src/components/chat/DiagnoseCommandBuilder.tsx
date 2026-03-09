import type { Artifact } from "@/lib/types";
import {
  Box,
  Button,
  Checkbox,
  Flex,
  Heading,
  Text,
  VStack,
} from "@chakra-ui/react";
import { useState } from "react";

interface DiagnoseCommandBuilderProps {
  artifacts: Artifact[];
  onSelect: (command: string) => void;
  onCancel: () => void;
}

export function DiagnoseCommandBuilder({
  artifacts,
  onSelect,
  onCancel,
}: DiagnoseCommandBuilderProps) {
  const [selectedIds, setSelectedIds] = useState<string[]>([]);

  const toggleId = (id: string, checked: boolean) => {
    if (checked) {
      setSelectedIds((prev) => [...prev, id]);
    } else {
      setSelectedIds((prev) => prev.filter((x) => x !== id));
    }
  };

  const handleBuild = () => {
    let command = "/diagnose";
    if (selectedIds.length > 0) {
      command += ` /artifacts ${selectedIds.join(",")}`;
    }
    onSelect(command);
  };

  return (
    <Box
      position="absolute"
      bottom="100%"
      left={0}
      mb={2}
      w="full"
      maxW="600px"
      bgColor="#1a1a1a"
      border="1px solid #333"
      borderRadius="lg"
      boxShadow="0 4px 12px rgba(0, 0, 0, 0.5)"
      p={4}
      zIndex={10}
    >
      <VStack align="stretch" gap={4}>
        <Heading size="sm" color="white">
          Build /diagnose command
        </Heading>

        {/* Artifacts Section */}
        <Box>
          <Text fontSize="sm" fontWeight="bold" color="white" mb={2}>
            Artifacts{" "}
            <Text as="span" fontSize="xs" color="#999" fontWeight="normal">
              (leave all unchecked to diagnose all artifacts)
            </Text>
          </Text>
          {artifacts.length === 0 ? (
            <Text fontSize="xs" color="#999">
              No artifacts in this session yet.
            </Text>
          ) : (
            <VStack align="stretch" gap={2}>
              {artifacts.map((art) => (
                <Checkbox.Root
                  key={art.id}
                  checked={selectedIds.includes(art.id)}
                  onCheckedChange={(details) =>
                    toggleId(art.id, !!details.checked)
                  }
                >
                  <Checkbox.HiddenInput />
                  <Checkbox.Control
                    borderColor="#555"
                    _checked={{ bgColor: "#F59E0B", borderColor: "#F59E0B" }}
                  >
                    <Checkbox.Indicator />
                  </Checkbox.Control>
                  <Checkbox.Label color="white" fontSize="sm">
                    <Text as="span" color="#F59E0B" fontFamily="mono">
                      {art.kind}
                    </Text>
                    {" — "}
                    {art.title}
                  </Checkbox.Label>
                </Checkbox.Root>
              ))}
            </VStack>
          )}
        </Box>

        {/* Action Buttons */}
        <Flex gap={2} justify="flex-end">
          <Button variant="ghost" size="sm" onClick={onCancel} color="white">
            Cancel
          </Button>
          <Button
            bgColor="#F59E0B"
            color="black"
            size="sm"
            onClick={handleBuild}
          >
            Run /diagnose
          </Button>
        </Flex>
      </VStack>
    </Box>
  );
}
