import { z } from 'zod'
import { ZGetManyQuery, ZModel, ZPaginatedResponse } from './utils.js'

export const ZModerationReview = z
	.object({
		catalogBookId: z.string().uuid(),
		submittedByUserId: z.string().uuid(),
		status: z.enum(['pending', 'approved', 'rejected']),
		decision: z.enum(['approved', 'rejected']).optional(),
		evidenceJson: z.record(z.string(), z.any()).default({}),
		reviewerUserId: z.string().uuid().optional(),
		reviewedAt: z.string().datetime().optional(),
	})
	.extend(ZModel.shape)

export const ZCreateModerationReviewDTO = z.object({
	catalogBookId: z.string().uuid(),
	evidence: z.record(z.string(), z.any()).default({}),
})

export const ZDecideModerationReviewDTO = z.object({
	decision: z.enum(['approved', 'rejected']),
})

export const ZModerationReviewIDParams = z.object({
	id: z.string().uuid(),
})

export const ZModerationListQuery = ZGetManyQuery.extend({
	status: z.enum(['pending', 'approved', 'rejected']).optional(),
})

export const ZModerationListResponse = ZPaginatedResponse(ZModerationReview)
