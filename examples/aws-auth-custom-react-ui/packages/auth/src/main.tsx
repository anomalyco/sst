import "./main.css";
import { hydrate } from "@openauthjs/react/client";

console.log("This only runs in the browser");
console.log("Here is an browser API: ", localStorage);

hydrate();
