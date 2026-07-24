import { lazy, Suspense } from "react";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import ProjectList from "./pages/ProjectList";

const Annotator = lazy(() => import("./pages/Annotator"));

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<ProjectList />} />
        <Route
          path="/projects/:projectId"
          element={(
            <Suspense fallback={<div className="p-8">Loading...</div>}>
              <Annotator />
            </Suspense>
          )}
        />
      </Routes>
    </BrowserRouter>
  );
}
