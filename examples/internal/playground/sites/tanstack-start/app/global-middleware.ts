import { registerGlobalMiddleware } from '@tanstack/react-start'
import { logMiddleware } from './utils/loggingMiddleware'

registerGlobalMiddleware({
  middleware: [logMiddleware],
})
