import "bootstrap-icons/font/bootstrap-icons.css";
import "bootstrap/dist/css/bootstrap.min.css";
import { NavLink, Outlet, Route, Routes } from "react-router-dom";
import "./App.css";
import Blacklist from "./Blacklist";
import Dashboard from "./Dashboard";
import History from "./History";
import Proxies from "./Proxies";
import Reverify from "./Reverify";
import { ErrorBoundary } from "./components/ErrorBoundary";

function Header() {
  return (
    <header className="p-1 mb-3 border-bottom">
      <div className="container">
        <div className="d-flex align-items-center justify-content-lg-start">
          <a href="/" className="logo align-items-center mb-lg-0">
            <img src="/logo.png" alt="slrp" />
          </a>
          <ul className="nav col-12 col-lg-auto me-lg-auto mb-2 mb-md-0">
            <li>
              <NavLink to="/" className="nav-link px-2 link-secondary">
                Overview
              </NavLink>
            </li>
            <li>
              <NavLink to="/proxies" className="nav-link px-2 link-dark">
                Proxies
              </NavLink>
            </li>
            <li>
              <NavLink to="/history" className="nav-link px-2 link-dark">
                History
              </NavLink>
            </li>
            <li>
              <NavLink to="/reverify" className="nav-link px-2 link-dark">
                Reverify
              </NavLink>
            </li>
            <li>
              <NavLink to="/blacklist" className="nav-link px-2 link-dark">
                Blacklist
              </NavLink>
            </li>
          </ul>
        </div>
      </div>
    </header>
  );
}

function Layout() {
  return (
    <div className="App">
      <Header />
      <main className="container">
        <ErrorBoundary>
          <Outlet />
        </ErrorBoundary>
      </main>
    </div>
  );
}

function App() {
  return (
    <Routes>
      <Route path="/" element={<Layout />}>
        <Route index element={<Dashboard />} />
        <Route path="proxies" element={<Proxies />} />
        <Route path="history" element={<History />} />
        <Route path="reverify" element={<Reverify />} />
        <Route path="blacklist" element={<Blacklist />} />
      </Route>
    </Routes>
  );
}

export default App;
