import type { Route } from "./+types/home";
import { Welcome } from "../welcome/welcome";
import { NavLink } from "react-router";

export function meta({}: Route.MetaArgs) {
  return [
    { title: "New React Router App" },
    { name: "description", content: "Welcome to React Router!" },
  ];
}

export default function Home() {
  return (
    <div>
      <Welcome />
      <nav style={{ marginTop: "2rem" }}>
        <ul>
          <li>
            <NavLink
              to="/server-rendered"
              className="group flex items-center gap-3 self-stretch p-3 leading-normal text-blue-700 hover:underline dark:text-blue-500"
            >
              Server Rendered
            </NavLink>
          </li>
          <li>
            <NavLink
              className="group flex items-center gap-3 self-stretch p-3 leading-normal text-blue-700 hover:underline dark:text-blue-500"
              to="/client-rendered"
            >
              Client Rendered
            </NavLink>
          </li>
          <li>
            <NavLink
              className="group flex items-center gap-3 self-stretch p-3 leading-normal text-blue-700 hover:underline dark:text-blue-500"
              to="/prerendered"
            >
              Prerendered
            </NavLink>
          </li>
          <li>
            <NavLink
              className="group flex items-center gap-3 self-stretch p-3 leading-normal text-blue-700 hover:underline dark:text-blue-500"
              to="/plus+sign-in-path"
            >
              Plus Sign in Path
            </NavLink>
          </li>
          <li>
            <NavLink
              className="group flex items-center gap-3 self-stretch p-3 leading-normal text-blue-700 hover:underline dark:text-blue-500"
              to="/error"
            >
              Error
            </NavLink>
          </li>
        </ul>
      </nav>
    </div>
  );
}
