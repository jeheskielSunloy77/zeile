import { z } from 'zod'
import { ZResponseWithData } from './utils.js'

const ZHealthStatus = z.enum(['healthy', 'unhealthy'])

const ZHealthCheck = z.object({
	status: ZHealthStatus,
	response_time: z.string(),
	error: z.string().optional(),
})

export const ZHealthResponse = ZResponseWithData(
	z.object({
		status: ZHealthStatus,
		timestamp: z.string().datetime(),
		environment: z.string(),
		checks: z.object({
			database: ZHealthCheck,
			redis: ZHealthCheck.optional(),
		}),
	})
)
