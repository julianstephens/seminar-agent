import {
  Button,
  Dialog,
  Field,
  Input,
  NativeSelectField,
  NativeSelectRoot,
} from "@chakra-ui/react";

interface CreateTutorialDialogProps {
  isOpen: boolean;
  onClose: () => void;
  titleRef: React.RefObject<HTMLInputElement | null>;
  subjectRef: React.RefObject<HTMLInputElement | null>;
  descriptionRef: React.RefObject<HTMLInputElement | null>;
  difficulty: "beginner" | "intermediate" | "advanced";
  setDifficulty: (difficulty: "beginner" | "intermediate" | "advanced") => void;
  creating: boolean;
  handleCreate: () => void;
}

export const CreateTutorialDialog = ({
  isOpen,
  onClose,
  titleRef,
  subjectRef,
  descriptionRef,
  difficulty,
  setDifficulty,
  creating,
  handleCreate,
}: CreateTutorialDialogProps) => {
  return (
    <Dialog.Root open={isOpen} onOpenChange={(d) => !d.open && onClose()}>
      <Dialog.Backdrop />
      <Dialog.Positioner>
        <Dialog.Content mt={0}>
          <Dialog.Header>
            <Dialog.Title>Create Tutorial</Dialog.Title>
          </Dialog.Header>
          <Dialog.Body>
            <Field.Root required mb={3}>
              <Field.Label>Title</Field.Label>
              <Input ref={titleRef} placeholder="Tutorial title" />
            </Field.Root>
            <Field.Root required mb={3}>
              <Field.Label>Subject</Field.Label>
              <Input ref={subjectRef} placeholder="Tutorial subject" />
            </Field.Root>
            <Field.Root mb={3}>
              <Field.Label>Description</Field.Label>
              <Input ref={descriptionRef} placeholder="Optional description" />
            </Field.Root>
            <Field.Root>
              <Field.Label>Difficulty</Field.Label>
              <NativeSelectRoot>
                <NativeSelectField
                  value={difficulty}
                  onChange={(e) =>
                    setDifficulty(e.target.value as typeof difficulty)
                  }
                >
                  <option value="beginner">Beginner</option>
                  <option value="intermediate">Intermediate</option>
                  <option value="advanced">Advanced</option>
                </NativeSelectField>
              </NativeSelectRoot>
            </Field.Root>
          </Dialog.Body>
          <Dialog.Footer>
            <Dialog.CloseTrigger asChild>
              <Button variant="ghost">Cancel</Button>
            </Dialog.CloseTrigger>
            <Button
              bg="#f59e0b"
              color="black"
              _hover={{ bg: "#fbbf24" }}
              loading={creating}
              onClick={handleCreate}
            >
              Create
            </Button>
          </Dialog.Footer>
        </Dialog.Content>
      </Dialog.Positioner>
    </Dialog.Root>
  );
};
