import type { ArtifactKind } from "@/lib/types";
import { createContext, useContext, useRef, useState } from "react";

interface CreateArtifactDialogContextType {
  isOpen: boolean;
  openDialog: () => void;
  closeDialog: () => void;
  titleRef: React.MutableRefObject<HTMLInputElement | null>;
  contentRef: React.MutableRefObject<HTMLTextAreaElement | null>;
  kind: ArtifactKind;
  setKind: (kind: ArtifactKind) => void;
  createAnother: boolean;
  setCreateAnother: (value: boolean) => void;
  onCreateCallback: React.MutableRefObject<(() => void) | null>;
}

const CreateArtifactDialogContext = createContext<
  CreateArtifactDialogContextType | undefined
>(undefined);

export const CreateArtifactDialogProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [kind, setKind] = useState<ArtifactKind>("notes");
  const [createAnother, setCreateAnother] = useState(false);
  const titleRef = useRef<HTMLInputElement>(null);
  const contentRef = useRef<HTMLTextAreaElement>(null);
  const onCreateCallback = useRef<(() => void) | null>(null);

  return (
    <CreateArtifactDialogContext.Provider
      value={{
        isOpen,
        openDialog: () => setIsOpen(true),
        closeDialog: () => setIsOpen(false),
        titleRef,
        contentRef,
        kind,
        setKind,
        createAnother,
        setCreateAnother,
        onCreateCallback,
      }}
    >
      {children}
    </CreateArtifactDialogContext.Provider>
  );
};

// eslint-disable-next-line react-refresh/only-export-components
export const useCreateArtifactDialog = () => {
  const context = useContext(CreateArtifactDialogContext);
  if (!context) {
    throw new Error(
      "useCreateArtifactDialog must be used within CreateArtifactDialogProvider",
    );
  }
  return context;
};
