import {
  Box,
  Button,
  Card,
  HStack,
  Icon,
  Text,
  Textarea,
  VStack,
} from "@chakra-ui/react";
import { LuBan } from "react-icons/lu";

export const TutorialSessionActions = ({
  onComplete,
  onAbandon,
  completing,
  abandoning,
  showCompleteForm,
  onToggleCompleteForm,
  notesRef,
}: {
  onComplete: () => void;
  onAbandon: () => void;
  completing: boolean;
  abandoning: boolean;
  showCompleteForm: boolean;
  onToggleCompleteForm: () => void;
  notesRef: React.RefObject<HTMLTextAreaElement | null>;
}) => {
  return (
    <Box
      maxW={{ base: "100vw", md: "4xl" }}
      w={{ md: "full" }}
      mx={{ md: "auto" }}
      py={4}
      px={{ base: 4, md: 0 }}
    >
      <HStack w="full" gap={3} wrap="wrap">
        <Button size="sm" className="primary" onClick={onToggleCompleteForm}>
          {showCompleteForm ? "Cancel" : "Complete Session"}
        </Button>
        <Button
          size="sm"
          variant="subtle"
          colorPalette="red"
          loading={abandoning}
          onClick={onAbandon}
        >
          <Icon as={LuBan} />
          Abandon
        </Button>
      </HStack>

      {showCompleteForm && (
        <Card.Root mt={4} p={4}>
          <VStack align="stretch" gap={3}>
            <Text fontWeight="medium">Session Notes (optional)</Text>
            <Textarea
              ref={notesRef}
              placeholder="Add any final notes..."
              rows={4}
            />
            <Button
              className="primary"
              loading={completing}
              onClick={onComplete}
            >
              Confirm Complete
            </Button>
          </VStack>
        </Card.Root>
      )}
    </Box>
  );
};
