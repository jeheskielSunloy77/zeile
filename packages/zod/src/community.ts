import { z } from 'zod'
import { ZModel, ZPaginatedResponse } from './utils.js'
import { ZVisibility } from './library.js'

export const ZCommunityProfile = z
	.object({
		userId: z.string().uuid(),
		displayName: z.string().max(120).optional(),
		bio: z.string().max(2000).optional(),
		avatarUrl: z.string().url().optional(),
		showReadingActivity: z.boolean(),
		showHighlights: z.boolean(),
		showLists: z.boolean(),
	})
	.extend(ZModel.omit({ id: true, deletedAt: true }).shape)

export const ZActivityEvent = z.object({
	id: z.string().uuid(),
	userId: z.string().uuid(),
	eventType: z.string().min(1),
	resourceType: z.string().min(1),
	resourceId: z.string().uuid().optional(),
	payloadJson: z.record(z.string(), z.any()).default({}),
	visibility: ZVisibility,
	createdAt: z.string().datetime(),
})

export const ZUpdateCommunityProfileDTO = z.object({
	displayName: z.string().max(120).optional(),
	bio: z.string().max(2000).optional(),
	avatarUrl: z.string().url().optional(),
	showReadingActivity: z.boolean().optional(),
	showHighlights: z.boolean().optional(),
	showLists: z.boolean().optional(),
})

export const ZCommunityProfilePathParams = z.object({
	userId: z.string().uuid(),
})

export const ZCommunityActivityResponse = ZPaginatedResponse(ZActivityEvent)
