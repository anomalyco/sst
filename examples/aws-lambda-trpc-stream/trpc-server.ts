import { initTRPC, TRPCError } from '@trpc/server';
import { awsLambdaStreamingRequestHandler } from '@trpc/server/adapters/aws-lambda';
import z from 'zod';

const { router, procedure } = initTRPC.create();

const apiRouter = router({
  getById: procedure
    .input(
      z.object({
        delay: z.number(),
        idx: z.number(),
      }),
    )
    .query(async ({ input }) => {
      await new Promise((r) => setTimeout(r, input.delay));
      if (Math.random() < 0.5) {
        console.log('sending response', input.delay);
        return { id: Math.random(), idx: input.idx };
      }
      console.log('sending error', input.idx);
      throw new TRPCError({
        code: 'BAD_REQUEST',
        message: 'Random failure',
      });
    }),
});

export type ApiRouter = typeof apiRouter;

export const handler = awslambda.streamifyResponse(
  awsLambdaStreamingRequestHandler({
    router: apiRouter,
    allowMethodOverride: true,
  }),
);
