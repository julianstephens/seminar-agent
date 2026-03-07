import { Box, Button, Flex, Icon, Span, Textarea } from "@chakra-ui/react";
import { type KeyboardEvent, useState } from "react";
import { LuSend } from "react-icons/lu";

interface ChatInputProps {
  onSend: (message: string) => void;
  disabled?: boolean;
  placeholder?: string;
}

export function ChatInput({
  onSend,
  disabled,
  placeholder = "Your message...",
}: ChatInputProps) {
  const [message, setMessage] = useState("");

  const handleSubmit = () => {
    if (message.trim() && !disabled) {
      onSend(message.trim());
      setMessage("");
    }
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      handleSubmit();
    }
  };

  return (
    <Box w="full" borderY="1px solid #333" bgColor="transparent" p={4}>
      <Box maxW="4xl" mx="auto">
        <Box position="relative">
          <Textarea
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={placeholder}
            disabled={disabled}
            w="full"
            bgColor="#1a1a1a"
            border="none"
            color="white"
            rounded="lg"
            pr={24}
            resize="none"
            _focus={{ outline: "none", ring: "2px", ringColor: "#F59E0B" }}
            minH="80px"
            maxH="200px"
            _placeholder={{ color: "#666" }}
            rows={2}
          />
          <Flex
            position="absolute"
            bottom={4}
            right={4}
            alignItems="center"
            gap={3}
          >
            <Span color="#666" fontSize="xs">
              ⌘/Ctrl + Enter to submit
            </Span>
            <Button
              onClick={handleSubmit}
              disabled={!message.trim() || disabled}
              bgColor="#F59E0B"
              color="black"
              px={5}
              py={2}
              rounded="md"
              fontWeight="bold"
              transition="background-color 0.2s"
              display="flex"
              alignItems="center"
              gap={2}
              _hover={{ bgColor: "#D97706" }}
              _disabled={{ bgColor: "#666", cursor: "not-allowed" }}
            >
              <Icon w={4} h={4}>
                <LuSend />
              </Icon>
              Send
            </Button>
          </Flex>
        </Box>
      </Box>
    </Box>
  );
}
