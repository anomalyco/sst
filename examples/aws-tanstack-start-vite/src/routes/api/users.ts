import { createServerFileRoute } from '@tanstack/react-start/server'
import { createMiddleware, json } from '@tanstack/react-start'
import type { User } from '~/utils/users'

const userLoggerMiddleware = createMiddleware({ type: 'request' }).server(
  async ({ next, request }) => {
    console.info('In: /users')
    const result = await next()
    result.response.headers.set('x-users', 'true')
    console.info('Out: /users')
    return result
  },
)

const testParentMiddleware = createMiddleware({ type: 'request' }).server(
  async ({ next, request }) => {
    console.info('In: testParentMiddleware')
    const result = await next()
    result.response.headers.set('x-test-parent', 'true')
    console.info('Out: testParentMiddleware')
    return result
  },
)

const testMiddleware = createMiddleware({ type: 'request' })
  .middleware([testParentMiddleware])
  .server(async ({ next, request }) => {
    console.info('In: testMiddleware')
    const result = await next()
    result.response.headers.set('x-test', 'true')

    // if (Math.random() > 0.5) {
    //   throw new Response(null, {
    //     status: 302,
    //     headers: { Location: 'https://www.google.com' },
    //   })
    // }

    console.info('Out: testMiddleware')
    return result
  })

export const ServerRoute = createServerFileRoute('/api/users')
  .middleware([testMiddleware, userLoggerMiddleware, testParentMiddleware])
  .methods({
    GET: async ({ request }) => {
      console.info('Fetching users... @', request.url)
      const res = await fetch('https://jsonplaceholder.typicode.com/users')
      if (!res.ok) {
        throw new Error('Failed to fetch users')
      }

      const data = (await res.json()) as Array<User>

      const list = data.slice(0, 10)

      return json(list.map((u) => ({ id: u.id, name: u.name, email: u.email })))
    },
  })
