import type { TutorialSessionDetail } from "@/lib/types";
import {
  Alert,
  Badge,
  Box,
  Button,
  Heading,
  HStack,
  Icon,
  Text,
  VStack,
} from "@chakra-ui/react";
import { useState } from "react";
import { LuArrowLeft, LuDownload } from "react-icons/lu";

export const TutorialSessionHeader = ({
  detail,
  onBack,
  onExport,
}: {
  detail: TutorialSessionDetail;
  onBack: () => void;
  onExport: () => void;
}) => {
  const [kindDesc] = useState(
    detail.kind && detail.kind === "diagnostic"
      ? "This is a short session examining recent artifacts to identify emerging patterns in your reasoning."
      : detail.kind && detail.kind === "extended"
        ? "This is a longer session examining recent artifacts, as well as reviewing and assigning problemsets."
        : "This is a tutorial session.",
  );

  return (
    <HStack
      id="tutorialSessionHeader"
      mb={4}
      justify="space-between"
      align="start"
      bgColor="#1a1a1a"
      wrap="wrap"
      w="full"
      gap={2}
      borderBottom="1px #333 solid"
    >
      <HStack maxW="5xl" w="full" mx="auto" pt="2" px="6" pb="2">
        <VStack w="full">
          <HStack w="full" justifyContent="space-between" alignItems="start">
            <Box>
              <Heading size="md" fontWeight="bold">
                Tutorial Session
              </Heading>
              <Text fontSize="xs" color="gray.500">
                Started {new Date(detail.started_at).toLocaleString()}
              </Text>
              {detail.kind && (
                <Text fontSize="xs" color="gray.500">
                  {detail.kind.replace(/_/g, " ")}
                </Text>
              )}
            </Box>
            <HStack gap={2} flexShrink={0}>
              <Badge
                colorPalette={
                  detail.status === "complete"
                    ? "green"
                    : detail.status === "abandoned"
                      ? "gray"
                      : "yellow"
                }
              >
                {detail.status}
              </Badge>
              <Button
                size="xs"
                py={2}
                px={4}
                display="flex"
                alignItems="center"
                className="grey"
                onClick={onExport}
              >
                <Icon w={4} h={4}>
                  <LuDownload />
                </Icon>
                Export
              </Button>
              <Button
                size="xs"
                py={2}
                px={4}
                display="flex"
                alignItems="center"
                className="grey"
                onClick={onBack}
              >
                <Icon>
                  <LuArrowLeft />
                </Icon>
                Back
              </Button>
            </HStack>
          </HStack>
          <Alert.Root
            bgColor="#0a0a0a"
            border="1px rgba(245, 158, 11, 0.3) solid"
            rounded="lg"
            p={3}
          >
            <Alert.Indicator color="#f59e0b" />
            <VStack alignItems="start">
              <Alert.Title
                fontSize="sm"
                color="white"
                textTransform="capitalize"
              >
                {detail.kind} Tutorial
              </Alert.Title>
              <Alert.Description fontSize="xs" color="fg.muted">
                {kindDesc}
              </Alert.Description>
            </VStack>
          </Alert.Root>
        </VStack>
      </HStack>
    </HStack>
  );
};
