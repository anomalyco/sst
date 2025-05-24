import { registerGlobalMiddleware } from "@tanstack/react-start";
import { signatureMiddleware } from "./middlewares/signature";

registerGlobalMiddleware({
  middleware: [signatureMiddleware],
});
