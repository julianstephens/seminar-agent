import { useAuthState } from "@/auth/useAuth";
import { CreateSeminarDialog } from "@/components/dialogs/CreateSeminarDialog";
import { EditSeminarDialog } from "@/components/dialogs/EditSeminarDialog";
import { NewSessionDialog } from "@/components/dialogs/NewSessionDialog";
import { SelectSeminarDialog } from "@/components/dialogs/SelectSeminarDialog";
import { SelectTutorialDialog } from "@/components/dialogs/SelectTutorialDialog";
import { useEditSeminarDialog } from "@/contexts/EditSeminarDialogContext";
import { useNewSessionDialog } from "@/contexts/NewSessionDialogContext";
import { useSeminarDialog } from "@/contexts/SeminarDialogContext";
import { useApi } from "@/lib/ApiContext";
import type {
  CreateSeminarInput,
  CreateSessionInput,
  UpdateSeminarInput,
} from "@/lib/types";
import { Box, Button, Flex, HStack, Spacer, Text } from "@chakra-ui/react";
import React, { useState } from "react";
import { Outlet, NavLink as RouterLink, useNavigate } from "react-router-dom";

/**
 * Top-level layout shell: nav bar + page content area.
 * All authenticated routes render inside this layout via <Outlet />.
 */
const BaseLayout = ({ children }: React.PropsWithChildren) => {
  const { logout, user } = useAuthState();
  const navigate = useNavigate();
  const api = useApi();

  // Create seminar dialog
  const { isOpen, closeDialog, callbackRef } = useSeminarDialog();

  // Edit seminar dialog
  const {
    isOpen: editIsOpen,
    closeDialog: closeEditDialog,
    seminar,
    titleRef,
    authorRef,
    editionNotesRef,
    callbackRef: editCallbackRef,
  } = useEditSeminarDialog();
  const [saving, setSaving] = useState(false);

  // New session dialog
  const {
    isOpen: sessionIsOpen,
    closeDialog: closeSessionDialog,
    sectionLabelRef,
    seminarIdRef,
  } = useNewSessionDialog();
  const [creatingSession, setCreatingSession] = useState(false);

  const handleCreate = async (input: CreateSeminarInput) => {
    try {
      await api.createSeminar(input);
      closeDialog();
      // Call the registered callback to refresh the seminars list
      callbackRef.current?.();
    } catch {
      // Error is handled by the dialog component
    }
  };

  const handleSave = async () => {
    if (!seminar) return;
    const input: UpdateSeminarInput = {
      title: titleRef.current?.value.trim() || undefined,
      author: authorRef.current?.value.trim() || undefined,
      edition_notes: editionNotesRef.current?.value.trim() || undefined,
    };
    setSaving(true);
    try {
      const updated = await api.updateSeminar(seminar.id, input);
      closeEditDialog();
      // Call the registered callback to update the page
      editCallbackRef.current?.(updated);
    } finally {
      setSaving(false);
    }
  };

  const handleCreateSession = async () => {
    const label = sectionLabelRef.current?.value.trim() ?? "";
    if (!label) return;

    const seminarId = seminarIdRef.current;
    if (!seminarId) return;

    const input: CreateSessionInput = {
      section_label: label,
    };
    setCreatingSession(true);
    try {
      const s = await api.createSession(seminarId, input);
      closeSessionDialog();
      navigate(`/sessions/${s.id}`);
    } finally {
      setCreatingSession(false);
    }
  };

  const handleLogout = () => {
    logout({ logoutParams: { returnTo: window.location.origin + "/login" } });
  };

  return (
    <>
      <Flex id="layout" direction="column" minH="100vh" minW="100vw">
        {/* Nav */}
        <Box
          as="nav"
          bg="#1a1a1a"
          color="white"
          borderBottom="2px solid #f59e0b"
          px={{ base: 3, md: 6 }}
          py={3}
        >
          <HStack d="flex" alignItems="center" gap={{ base: 3, md: 6 }}>
            <Text
              mr="1rem"
              fontWeight="bold"
              fontSize="lg"
              cursor="pointer"
              flexShrink={0}
              onClick={() => navigate("/")}
            >
              Formation
            </Text>
            <RouterLink
              to="/seminars"
              style={{ color: "inherit", fontSize: "0.9rem" }}
            >
              Seminars
            </RouterLink>
            <RouterLink
              to="/tutorials"
              style={{ color: "inherit", fontSize: "0.9rem" }}
            >
              Tutorials
            </RouterLink>
            <Spacer />
            {user && (
              <HStack gap={2} flexShrink={0}>
                <Text
                  fontSize="sm"
                  opacity={0.85}
                  display={{ base: "none", sm: "block" }}
                  maxW={{ sm: "140px", md: "none" }}
                  overflow="hidden"
                  textOverflow="ellipsis"
                  whiteSpace="nowrap"
                >
                  {user.email ?? user.name}
                </Text>
                <Button
                  className="secondary"
                  size="sm"
                  variant="outline"
                  colorScheme="whiteAlpha"
                  flexShrink={0}
                  onClick={handleLogout}
                >
                  Sign out
                </Button>
              </HStack>
            )}
          </HStack>
        </Box>

        {/* Page */}
        {children}
      </Flex>

      {/* Create Seminar Dialog */}
      <CreateSeminarDialog
        open={isOpen}
        setOpen={(open) => !open && closeDialog()}
        handleCreate={handleCreate}
      />

      {/* Edit Seminar Dialog */}
      <EditSeminarDialog
        editOpen={editIsOpen}
        setEditOpen={(open) => !open && closeEditDialog()}
        seminar={seminar}
        editTitleRef={titleRef}
        editAuthorRef={authorRef}
        editEditionNotesRef={editionNotesRef}
        saving={saving}
        handleSave={handleSave}
      />

      {/* New Session Dialog */}
      <NewSessionDialog
        sessionOpen={sessionIsOpen}
        setSessionOpen={(open) => !open && closeSessionDialog()}
        sectionLabelRef={sectionLabelRef}
        creating={creatingSession}
        handleCreateSession={handleCreateSession}
      />

      {/* Select Seminar Dialog */}
      <SelectSeminarDialog />

      {/* Select Tutorial Dialog */}
      <SelectTutorialDialog />
    </>
  );
};

export const Layout = () => (
  <BaseLayout>
    <Box
      maxW={{ base: "100vh", md: "4xl" }}
      w={{ md: "full" }}
      h="full"
      mx={{ md: "auto" }}
      pt="6"
      px={{ base: "6", md: "0" }}
      flex={1}
    >
      <Outlet />
    </Box>
  </BaseLayout>
);

export const RunnerLayout = () => (
  <BaseLayout>
    <Box w="full" h="full" flex={1}>
      <Outlet />
    </Box>
  </BaseLayout>
);
