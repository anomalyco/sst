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
            <a
              className="group flex items-center gap-3 self-stretch p-3 leading-normal text-blue-700 hover:underline dark:text-blue-500"
              href="/server-rendered"
            >
              Server Rendered
            </a>
          </li>
          <li>
            <a
              className="group flex items-center gap-3 self-stretch p-3 leading-normal text-blue-700 hover:underline dark:text-blue-500"
              href="/client-rendered"
            >
              Client Rendered
            </a>
          </li>
          <li>
            <a
              className="group flex items-center gap-3 self-stretch p-3 leading-normal text-blue-700 hover:underline dark:text-blue-500"
              href="/prerendered"
            >
              Prerendered
            </a>
          </li>
          <li>
            <a
              className="group flex items-center gap-3 self-stretch p-3 leading-normal text-blue-700 hover:underline dark:text-blue-500"
              href="/plus+sign-in-path"
            >
              Plus Sign in Path
            </a>
          </li>
          <li>
            <a
              className="group flex items-center gap-3 self-stretch p-3 leading-normal text-blue-700 hover:underline dark:text-blue-500"
              href="/error"
            >
              Error
            </a>
          </li>
        </ul>
      </nav>
    </div>
  );
}
