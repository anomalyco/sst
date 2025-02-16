import { createMiddleware } from '@tanstack/start'

export const logMiddleware = createMiddleware()
  .client(async (ctx) => {
    const clientTime = new Date()

    return ctx.next({
      context: {
        clientTime,
      },
      sendContext: {
        clientTime,
      },
    })
  })
  .server(async (ctx) => {
    const serverTime = new Date()

    return ctx.next({
      sendContext: {
        serverTime,
        durationToServer:
          serverTime.getTime() - ctx.context.clientTime.getTime(),
      },
    })
  })
  .clientAfter(async (ctx) => {
    const now = new Date()

    console.log('Client Req/Res:', {
      duration: ctx.context.clientTime.getTime() - now.getTime(),
      durationToServer: ctx.context.durationToServer,
      durationFromServer: now.getTime() - ctx.context.serverTime.getTime(),
    })

    return ctx.next()
  })
