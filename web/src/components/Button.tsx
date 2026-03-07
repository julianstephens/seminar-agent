import { prepareText } from "@/lib/utils";
import { Button, Icon, IconButton, type ButtonProps } from "@chakra-ui/react";
import { useState } from "react";
import {
  LuArrowLeft,
  LuCheck,
  LuCopy,
  LuDownload,
  LuTrash,
} from "react-icons/lu";
import { useNavigate } from "react-router-dom";

interface NavigateProps {
  to: string;
}

export const ExportButton = ({ to, ...props }: ButtonProps & NavigateProps) => {
  const navigate = useNavigate();
  return (
    <Button
      size="sm"
      className="grey"
      variant="outline"
      onClick={() => navigate(to)}
      {...props}
    >
      <LuDownload />
      Export
    </Button>
  );
};

export const BackButton = ({
  backPath,
  ...props
}: ButtonProps & { backPath: string }) => {
  const navigate = useNavigate();
  return (
    <Button
      className="grey"
      alignItems="center"
      size="sm"
      variant="ghost"
      onClick={() => navigate(backPath)}
      {...props}
    >
      <LuArrowLeft />
      Back
    </Button>
  );
};

export const DeleteButton = (props: ButtonProps) => {
  return (
    <IconButton size="sm" colorPalette="red" variant="subtle" {...props}>
      <Icon>
        <LuTrash />
      </Icon>
    </IconButton>
  );
};

export const CopyTextButton = ({ textToCopy }: { textToCopy: string }) => {
  const [isCopied, setIsCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(prepareText(textToCopy));
      setIsCopied(true);
      setTimeout(() => setIsCopied(false), 2000); // Reset "Copied!" message after 2 seconds
    } catch (err) {
      console.error("Failed to copy text: ", err);
    }
  };

  return (
    <IconButton variant="ghost" onClick={handleCopy}>
      <Icon as={isCopied ? LuCheck : LuCopy} />
    </IconButton>
  );
};
