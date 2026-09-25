import { Route, Routes } from "react-router";
import { Layout } from "./components/Layout";
import { Home } from "./routes/Home";
import { Person } from "./routes/Person";
import { Imprint, Methodology, NotFound, Privacy } from "./routes/Static";
import { Timeline } from "./routes/Timeline";

export function App() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/person/:id" element={<Person />} />
        <Route path="/person/:id/:topic" element={<Timeline />} />
        <Route path="/methodik" element={<Methodology />} />
        <Route path="/impressum" element={<Imprint />} />
        <Route path="/datenschutz" element={<Privacy />} />
        <Route path="*" element={<NotFound />} />
      </Routes>
    </Layout>
  );
}
