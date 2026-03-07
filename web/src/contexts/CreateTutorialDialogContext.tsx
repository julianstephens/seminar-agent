import { createContext, useContext, useRef, useState } from "react";

interface CreateTutorialDialogContextType {
  isOpen: boolean;
  openDialog: () => void;
  closeDialog: () => void;
  titleRef: React.MutableRefObject<HTMLInputElement | null>;
  subjectRef: React.MutableRefObject<HTMLInputElement | null>;
  descriptionRef: React.MutableRefObject<HTMLInputElement | null>;
  difficulty: "beginner" | "intermediate" | "advanced";
  setDifficulty: (difficulty: "beginner" | "intermediate" | "advanced") => void;
  onCreateCallback: React.MutableRefObject<(() => void) | null>;
}

const CreateTutorialDialogContext = createContext<
  CreateTutorialDialogContextType | undefined
>(undefined);

export const CreateTutorialDialogProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [difficulty, setDifficulty] = useState<
    "beginner" | "intermediate" | "advanced"
  >("beginner");
  const titleRef = useRef<HTMLInputElement>(null);
  const subjectRef = useRef<HTMLInputElement>(null);
  const descriptionRef = useRef<HTMLInputElement>(null);
  const onCreateCallback = useRef<(() => void) | null>(null);

  return (
    <CreateTutorialDialogContext.Provider
      value={{
        isOpen,
        openDialog: () => setIsOpen(true),
        closeDialog: () => setIsOpen(false),
        titleRef,
        subjectRef,
        descriptionRef,
        difficulty,
        setDifficulty,
        onCreateCallback,
      }}
    >
      {children}
    </CreateTutorialDialogContext.Provider>
  );
};

// eslint-disable-next-line react-refresh/only-export-components
export const useCreateTutorialDialog = () => {
  const context = useContext(CreateTutorialDialogContext);
  if (!context) {
    throw new Error(
      "useCreateTutorialDialog must be used within CreateTutorialDialogProvider",
    );
  }
  return context;
};
