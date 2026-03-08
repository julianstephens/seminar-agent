import {
  Box,
  Button,
  Flex,
  Heading,
  NativeSelect,
  NativeSelectField,
  NativeSelectRoot,
  Text,
  VStack,
} from "@chakra-ui/react";
import { useState } from "react";

interface Pattern {
  code: string;
  description: string;
}

const PATTERNS: Pattern[] = [
  {
    code: "auto",
    description: "Automatically select patterns based on recent diagnostics",
  },
  {
    code: "TEXT_DRIFT",
    description: "Claims drift from text evidence",
  },
  {
    code: "UNDEFINED_TERMS",
    description: "Uses terms without clear definition",
  },
  {
    code: "HIDDEN_PREMISES",
    description: "Arguments rely on unstated assumptions",
  },
  {
    code: "WEAK_STRUCTURE",
    description: "Organization lacks logical flow",
  },
  {
    code: "RHETORICAL_INFLATION",
    description: "Uses unnecessary dramatic language",
  },
  {
    code: "PREMATURE_SYNTHESIS",
    description: "Draws conclusions too early",
  },
];

interface ProblemSetCommandBuilderProps {
  onSelect: (command: string) => void;
  onCancel: () => void;
}

export function ProblemSetCommandBuilder({
  onSelect,
  onCancel,
}: ProblemSetCommandBuilderProps) {
  const [patterns, setPatterns] = useState("auto");
  const [difficulty, setDifficulty] = useState("intermediate");
  const [mode, setMode] = useState("commit");

  const handleBuild = () => {
    let command = "/problem-set";

    // Only add options if they differ from defaults
    if (patterns !== "auto") {
      command += ` /patterns ${patterns}`;
    }
    if (difficulty !== "intermediate") {
      command += ` /difficulty ${difficulty}`;
    }
    if (mode !== "commit") {
      command += ` /mode ${mode}`;
    }

    onSelect(command);
  };

  const selectedPattern = PATTERNS.find((p) => p.code === patterns);

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
          Build /problem-set command
        </Heading>

        {/* Patterns Section */}
        <Box>
          <Text fontSize="sm" fontWeight="bold" color="white" mb={2}>
            Patterns
          </Text>
          <NativeSelect.Root>
            <NativeSelect.Field
              value={patterns}
              onChange={(e) => setPatterns(e.target.value)}
              bgColor="#0a0a0a"
              color="white"
              border="1px solid #333"
              _hover={{ borderColor: "#555" }}
              _focus={{ borderColor: "#F59E0B", outline: "none" }}
            >
              {PATTERNS.map((pattern) => (
                <option key={pattern.code} value={pattern.code}>
                  {pattern.code === "auto" ? "Auto" : pattern.code}
                </option>
              ))}
            </NativeSelect.Field>
          </NativeSelect.Root>
          {selectedPattern && (
            <Text fontSize="xs" color="#999" mt={1}>
              {selectedPattern.description}
            </Text>
          )}
        </Box>

        {/* Difficulty Section */}
        <Box>
          <Text fontSize="sm" fontWeight="bold" color="white" mb={2}>
            Difficulty
          </Text>
          <NativeSelectRoot>
            <NativeSelectField
              value={difficulty}
              onChange={(e) => setDifficulty(e.target.value)}
              bgColor="#0a0a0a"
              color="white"
              border="1px solid #333"
              _hover={{ borderColor: "#555" }}
              _focus={{ borderColor: "#F59E0B", outline: "none" }}
            >
              <option value="beginner">Beginner</option>
              <option value="intermediate">Intermediate</option>
              <option value="advanced">Advanced</option>
            </NativeSelectField>
          </NativeSelectRoot>
        </Box>

        {/* Mode Section */}
        <Box>
          <Text fontSize="sm" fontWeight="bold" color="white" mb={2}>
            Mode
          </Text>
          <NativeSelectRoot>
            <NativeSelectField
              value={mode}
              onChange={(e) => setMode(e.target.value)}
              bgColor="#0a0a0a"
              color="white"
              border="1px solid #333"
              _hover={{ borderColor: "#555" }}
              _focus={{ borderColor: "#F59E0B", outline: "none" }}
            >
              <option value="commit">Commit (save to database)</option>
              <option value="preview">Preview (generate only)</option>
            </NativeSelectField>
          </NativeSelectRoot>
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
            _hover={{ bgColor: "#D97706" }}
          >
            Insert Command
          </Button>
        </Flex>
      </VStack>
    </Box>
  );
}
