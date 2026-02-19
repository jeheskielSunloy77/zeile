import { ZHealthResponse } from '@zeile/zod'
import { initContract } from '@ts-rest/core'
import { failResponses } from '../utils.js'

const c = initContract()

export const healthContract = c.router({
	getHealth: {
		summary: 'Get health',
		path: '/health',
		method: 'GET',
		description: 'Get health status',
		responses: {
			200: ZHealthResponse,
			...failResponses,
		},
	},
})
