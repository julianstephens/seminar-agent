import AuthGuard from "@/components/AuthGuard";
import { Layout, RunnerLayout } from "@/components/Layout";
import { CreateTutorialDialogProvider } from "@/contexts/CreateTutorialDialogContext";
import { EditSeminarDialogProvider } from "@/contexts/EditSeminarDialogContext";
import { NewSessionDialogProvider } from "@/contexts/NewSessionDialogContext";
import { SelectSeminarDialogProvider } from "@/contexts/SelectSeminarDialogContext";
import { SelectTutorialDialogProvider } from "@/contexts/SelectTutorialDialogContext";
import { SeminarDialogProvider } from "@/contexts/SeminarDialogContext";
import { SessionEventsProvider } from "@/contexts/SessionEventsContext";
import { TutorialSessionEventsProvider } from "@/contexts/TutorialSessionEventsContext";
import { ApiProvider } from "@/lib/ApiContext";
import Dashboard from "@/pages/Dashboard";
import Export from "@/pages/Export";
import Login from "@/pages/Login";
import SeminarDetail from "@/pages/SeminarDetail";
import SeminarList from "@/pages/SeminarList";
import SeminarSessionRunner from "@/pages/SeminarSessionRunner";
import SessionReview from "@/pages/SessionReview";
import TutorialDetail from "@/pages/TutorialDetail";
import TutorialList from "@/pages/TutorialList";
import TutorialSessionRunner from "@/pages/TutorialSession";
import { BrowserRouter, Route, Routes } from "react-router-dom";

function App() {
  return (
    <BrowserRouter>
      <Routes>
        {/* Public */}
        <Route path="/login" element={<Login />} />

        {/* Authenticated shell - Regular Layout */}
        <Route
          element={
            <AuthGuard>
              <ApiProvider>
                <SessionEventsProvider>
                  <TutorialSessionEventsProvider>
                    <SeminarDialogProvider>
                      <EditSeminarDialogProvider>
                        <NewSessionDialogProvider>
                          <SelectSeminarDialogProvider>
                            <SelectTutorialDialogProvider>
                              <CreateTutorialDialogProvider>
                                <Layout />
                              </CreateTutorialDialogProvider>
                            </SelectTutorialDialogProvider>
                          </SelectSeminarDialogProvider>
                        </NewSessionDialogProvider>
                      </EditSeminarDialogProvider>
                    </SeminarDialogProvider>
                  </TutorialSessionEventsProvider>
                </SessionEventsProvider>
              </ApiProvider>
            </AuthGuard>
          }
        >
          {/* Seminars */}
          <Route path="/seminars" element={<SeminarList />} />
          <Route path="/seminars/:id" element={<SeminarDetail />} />
          <Route
            path="/seminars/:id/export"
            element={<Export resourceType="seminar" />}
          />

          {/* Sessions */}
          <Route path="/sessions/:id/review" element={<SessionReview />} />
          <Route
            path="/sessions/:id/export"
            element={<Export resourceType="session" />}
          />

          {/* Tutorials */}
          <Route path="/tutorials" element={<TutorialList />} />
          <Route path="/tutorials/:id" element={<TutorialDetail />} />
          <Route
            path="/tutorials/:id/export"
            element={<Export resourceType="tutorial" />}
          />
          <Route
            path="/tutorial-sessions/:id/export"
            element={<Export resourceType="tutorial_session" />}
          />

          {/* Default */}
          <Route path="/" element={<Dashboard />} />
        </Route>

        {/* Authenticated shell - Full Screen Runner Layout */}
        <Route
          element={
            <AuthGuard>
              <ApiProvider>
                <SessionEventsProvider>
                  <TutorialSessionEventsProvider>
                    <SeminarDialogProvider>
                      <EditSeminarDialogProvider>
                        <NewSessionDialogProvider>
                          <SelectSeminarDialogProvider>
                            <SelectTutorialDialogProvider>
                              <CreateTutorialDialogProvider>
                                <RunnerLayout />
                              </CreateTutorialDialogProvider>
                            </SelectTutorialDialogProvider>
                          </SelectSeminarDialogProvider>
                        </NewSessionDialogProvider>
                      </EditSeminarDialogProvider>
                    </SeminarDialogProvider>
                  </TutorialSessionEventsProvider>
                </SessionEventsProvider>
              </ApiProvider>
            </AuthGuard>
          }
        >
          {/* Session Runners */}
          <Route path="/sessions/:id" element={<SeminarSessionRunner />} />
          <Route
            path="/tutorial-sessions/:id"
            element={<TutorialSessionRunner />}
          />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

export default App;
