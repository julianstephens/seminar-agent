import { createContext, useContext, useState } from "react";

interface SelectTutorialDialogContextType {
  isOpen: boolean;
  openDialog: () => void;
  closeDialog: () => void;
}

const SelectTutorialDialogContext = createContext<
  SelectTutorialDialogContextType | undefined
>(undefined);

export const SelectTutorialDialogProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <SelectTutorialDialogContext.Provider
      value={{
        isOpen,
        openDialog: () => setIsOpen(true),
        closeDialog: () => setIsOpen(false),
      }}
    >
      {children}
    </SelectTutorialDialogContext.Provider>
  );
};

// eslint-disable-next-line react-refresh/only-export-components
export const useSelectTutorialDialog = () => {
  const context = useContext(SelectTutorialDialogContext);
  if (!context) {
    throw new Error(
      "useSelectTutorialDialog must be used within SelectTutorialDialogProvider",
    );
  }
  return context;
};
