import { ChatMessage } from "@/components/chat/ChatMessage";
import type { TutorialTurn } from "@/lib/types";
import { Box, HStack, Spinner, Text, VStack } from "@chakra-ui/react";

export const TutorialTurnList = ({
  turns,
  streamingTurns,
  failedTurns,
  bottomRef,
}: {
  turns: TutorialTurn[];
  streamingTurns: Map<string, string>;
  failedTurns: Set<string>;
  bottomRef: React.RefObject<HTMLDivElement | null>;
}) => {
  // Show thinking spinner if:
  // 1. There are active streaming turns (agent is currently responding), OR
  // 2. Last turn is from user (waiting for agent response)
  const lastTurn = turns.length > 0 ? turns[turns.length - 1] : null;

  const agentThinking =
    streamingTurns.size > 0 || (lastTurn && lastTurn.speaker === "user");

  // Find streaming turns that don't have a corresponding turn in the turns array yet
  const streamingOnlyTurnIds = Array.from(streamingTurns.keys()).filter(
    (turnId) => !turns.some((t) => t.id === turnId),
  );

  return (
    <Box
      id="turnList"
      minH={{ base: "180px", md: "300px" }}
      maxH={{ base: "40vh", md: "55vh" }}
      overflowY="auto"
      mb={4}
    >
      {turns.length === 0 && streamingOnlyTurnIds.length === 0 ? (
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
                  key={t.id}
                  role={isUser ? "user" : "agent"}
                  content={isStreaming ? streamingText : displayText}
                  timestamp={new Date(t.created_at).toLocaleTimeString()}
                  failed={failedTurns.has(t.id)}
                />
              );
            })}
          {/* Render streaming-only agent responses (not yet in turns array) */}
          {streamingOnlyTurnIds.map((turnId) => (
            <ChatMessage
              key={turnId}
              role="agent"
              content={streamingTurns.get(turnId) ?? ""}
              timestamp={new Date().toLocaleTimeString()}
              failed={false}
            />
          ))}
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
