import { createServerFileRoute } from '@tanstack/react-start/server'
import { json } from '@tanstack/react-start'
import type { User } from '~/utils/users'

export const ServerRoute = createServerFileRoute('/api/users/$userId').methods({
  GET: async ({ params, request }) => {
    console.info(`Fetching users by id=${params.userId}... @`, request.url)
    try {
      const res = await fetch(
        'https://jsonplaceholder.typicode.com/users/' + params.userId,
      )
      if (!res.ok) {
        throw new Error('Failed to fetch user')
      }

      const user = (await res.json()) as User

      return json({
        id: user.id,
        name: user.name,
        email: user.email,
      })
    } catch (e) {
      console.error(e)
      return json({ error: 'User not found' }, { status: 404 })
    }
  },
})
