import { createContext, useContext, useState } from "react";

interface SelectSeminarDialogContextType {
  isOpen: boolean;
  openDialog: () => void;
  closeDialog: () => void;
}

const SelectSeminarDialogContext = createContext<
  SelectSeminarDialogContextType | undefined
>(undefined);

export const SelectSeminarDialogProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <SelectSeminarDialogContext.Provider
      value={{
        isOpen,
        openDialog: () => setIsOpen(true),
        closeDialog: () => setIsOpen(false),
      }}
    >
      {children}
    </SelectSeminarDialogContext.Provider>
  );
};

// eslint-disable-next-line react-refresh/only-export-components
export const useSelectSeminarDialog = () => {
  const context = useContext(SelectSeminarDialogContext);
  if (!context) {
    throw new Error(
      "useSelectSeminarDialog must be used within SelectSeminarDialogProvider",
    );
  }
  return context;
};
