import { ChatMessage } from "@/components/chat/ChatMessage";
import type { TutorialTurn } from "@/lib/types";
import { Box, HStack, Spinner, Text, VStack } from "@chakra-ui/react";

export const TutorialTurnList = ({
  turns,
  streamingTurns,
  bottomRef,
}: {
  turns: TutorialTurn[];
  streamingTurns: Map<string, string>;
  bottomRef: React.RefObject<HTMLDivElement | null>;
}) => {
  const agentThinking =
    turns.length > 0 && turns[turns.length - 1].speaker === "user";

  return (
    <Box
      id="turnList"
      minH={{ base: "180px", md: "300px" }}
      maxH={{ base: "40vh", md: "55vh" }}
      overflowY="auto"
      mb={4}
    >
      {turns.length === 0 ? (
        <Text color="gray.400" textAlign="center" mt={8}>
          Conversation will appear here. Submit a message to get started.
        </Text>
      ) : (
        <VStack align="stretch" gap={3}>
          {turns
            .filter((t) => t.text?.trim() || streamingTurns.has(t.id))
            .map((t) => {
              const isUser = t.speaker === "user";
              const streamingText = streamingTurns.get(t.id);
              const displayText = streamingText ?? t.text;
              const isStreaming = !!streamingText;

              return (
                <ChatMessage
                  role={isUser ? "user" : "agent"}
                  content={isStreaming ? streamingText : displayText}
                  timestamp={new Date(t.created_at).toLocaleTimeString()}
                />
                // <Box
                //     key={t.id}
                //     p={3}
                //     borderLeft="4px solid"
                //     borderLeftColor={
                //         isUser ? "blue.500" : isSystem ? "gray.400" : "teal.500"
                //     }
                //     bg={isUser ? "blue.50" : isSystem ? "gray.100" : "teal.50"}
                //     _dark={{
                //         bg: isUser
                //             ? "blue.900"
                //             : isSystem
                //                 ? "gray.700"
                //                 : "teal.900",
                //     }}
                //     rounded="md"
                //     opacity={isSystem ? 0.8 : 1}
                // >
                //     <HStack mb={2} gap={2} wrap="wrap">
                //         <Badge
                //             colorScheme={isUser ? "blue" : isSystem ? "gray" : "teal"}
                //             size="md"
                //             fontWeight="bold"
                //         >
                //             {isUser ? "👤 You" : isSystem ? "⚙ System" : "🤖 Agent"}
                //         </Badge>
                //         {isStreaming && (
                //             <Badge colorScheme="purple" size="sm">
                //                 streaming...
                //             </Badge>
                //         )}
                //     </HStack>
                //     <Text fontSize="sm" whiteSpace="pre-wrap" lineHeight="1.6">
                //         {displayText}
                //         {isStreaming && (
                //             <Box
                //                 as="span"
                //                 display="inline-block"
                //                 w="2px"
                //                 h="1em"
                //                 bg="currentColor"
                //                 ml={0.5}
                //                 css={css`
                //                     animation: blink 1s infinite;
                //                     @keyframes blink {
                //                         0%, 49% {
                //                             opacity: 1;
                //                         }
                //                         50%, 100% {
                //                             opacity: 0;
                //                         }
                //                     }
                //                 `}
                //             />
                //         )}
                //     </Text>
                //     <Text fontSize="xs" color="gray.400" mt={1}>
                //         {new Date(t.created_at).toLocaleTimeString()}
                //     </Text>
                // </Box>
              );
            })}
          {agentThinking && (
            <HStack gap={2} w="full" px={3} py={2}>
              <Spinner size="sm" />
              <Text fontSize="sm" color="gray.500" fontStyle="italic">
                Agent is thinking…
              </Text>
            </HStack>
          )}
        </VStack>
      )}
      <div ref={bottomRef} />
    </Box>
  );
};
