import { useSelectSeminarDialog } from "@/contexts/SelectSeminarDialogContext";
import { useApi } from "@/lib/ApiContext";
import type { Seminar } from "@/lib/types";
import {
  Badge,
  Box,
  Button,
  Card,
  Dialog,
  Spinner,
  Stack,
  Text,
} from "@chakra-ui/react";
import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

export const SelectSeminarDialog = () => {
  const { isOpen, closeDialog } = useSelectSeminarDialog();
  const api = useApi();
  const navigate = useNavigate();
  const [seminars, setSeminars] = useState<Seminar[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      setSeminars(await api.listSeminars());
    } catch (e) {
      setError(String(e));
    } finally {
      setLoading(false);
    }
  }, [api]);

  useEffect(() => {
    if (isOpen) {
      void load();
    }
  }, [isOpen, load]);

  const handleSelectSeminar = (seminarId: string) => {
    closeDialog();
    navigate(`/seminars/${seminarId}`);
  };

  return (
    <Dialog.Root open={isOpen} onOpenChange={(d) => !d.open && closeDialog()}>
      <Dialog.Backdrop />
      <Dialog.Positioner>
        <Dialog.Content mt={0}>
          <Dialog.Header>
            <Dialog.Title>Select Seminar</Dialog.Title>
          </Dialog.Header>
          <Dialog.Body>
            {loading ? (
              <Box textAlign="center" py={4}>
                <Spinner size="lg" />
              </Box>
            ) : error ? (
              <Text color="red.500">{error}</Text>
            ) : seminars.length === 0 ? (
              <Text color="gray.500">
                No seminars available. Create one first!
              </Text>
            ) : (
              <Stack gap={2}>
                {seminars.map((seminar) => (
                  <Card.Root
                    key={seminar.id}
                    cursor="pointer"
                    _hover={{ shadow: "md", bg: "gray.50", color: "black" }}
                    onClick={() => handleSelectSeminar(seminar.id)}
                  >
                    <Card.Body py={3}>
                      <Stack
                        direction="row"
                        justify="space-between"
                        align="center"
                      >
                        <Box>
                          <Text fontWeight="semibold">{seminar.title}</Text>
                          {seminar.author && (
                            <Text fontSize="sm" color="gray.600">
                              {seminar.author}
                            </Text>
                          )}
                        </Box>
                        <Badge colorScheme="purple">
                          {seminar.default_mode}
                        </Badge>
                      </Stack>
                    </Card.Body>
                  </Card.Root>
                ))}
              </Stack>
            )}
          </Dialog.Body>
          <Dialog.Footer>
            <Dialog.CloseTrigger asChild>
              <Button variant="ghost">Cancel</Button>
            </Dialog.CloseTrigger>
          </Dialog.Footer>
        </Dialog.Content>
      </Dialog.Positioner>
    </Dialog.Root>
  );
};
