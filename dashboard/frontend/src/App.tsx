import { BrowserRouter, Routes, Route } from "react-router-dom";
import Layout from "./components/Layout";
import Overview from "./pages/Overview";
import AutoHealer from "./pages/AutoHealer";
import RolloutManager from "./pages/RolloutManager";
import ResourceOptimizer from "./pages/ResourceOptimizer";
import Autoscaler from "./pages/Autoscaler";
import LiveEvents from "./pages/LiveEvents";
import Audit from "./pages/Audit";

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route index element={<Overview />} />
          <Route path="auto-healer" element={<AutoHealer />} />
          <Route path="rollout-manager" element={<RolloutManager />} />
          <Route path="resource-optimizer" element={<ResourceOptimizer />} />
          <Route path="autoscaler" element={<Autoscaler />} />
          <Route path="events" element={<LiveEvents />} />
          <Route path="audit" element={<Audit />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
