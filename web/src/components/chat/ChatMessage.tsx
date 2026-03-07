import { Box, Flex, Icon, Text } from "@chakra-ui/react";
import { LuBot, LuUser } from "react-icons/lu";
import "./chat.css";

interface ChatMessageProps {
  role: "user" | "agent";
  content: string;
  timestamp?: string;
}

export function ChatMessage({ role, content, timestamp }: ChatMessageProps) {
  const isUser = role === "user";

  return (
    <Flex gap={3} mb={4} alignItems="start">
      <Box
        flexShrink={0}
        w={8}
        h={8}
        rounded="full"
        display="flex"
        alignItems="center"
        justifyContent="center"
        bg={isUser ? "#1e3a8a" : "#065f46"}
      >
        <Icon color="white" w={5} h={5} as={isUser ? LuUser : LuBot} />
      </Box>

      <Box flex="1" minW={0}>
        <Box display="flex" alignItems="center" gap={2} mb={1}>
          <Box as="span" color="white" fontWeight="bold" fontSize="sm">
            {isUser ? "You" : "Agent"}
          </Box>
          {timestamp && (
            <Box as="span" color="#666" fontSize="xs">
              {timestamp}
            </Box>
          )}
        </Box>
        <Box
          rounded="lg"
          p={4}
          bg={isUser ? "rgba(30, 58, 138, 0.2)" : "rgba(6, 95, 70, 0.2)"}
        >
          <Text color="white" whiteSpace="pre-wrap" lineHeight="relaxed">
            {content}
          </Text>
        </Box>
      </Box>
    </Flex>
  );
}
