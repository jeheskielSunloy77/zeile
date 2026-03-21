import { z } from 'zod'
import { ZGetManyQuery, ZModel, ZPaginatedResponse } from './utils.js'
import { ZUserLibraryBook } from './library.js'

export const ZCommunityBookOwner = z.object({
	id: z.string().uuid(),
	username: z.string().min(3).max(50),
	avatarUrl: z.string().url().optional(),
})

export const ZCommunityBookAsset = z.object({
	id: z.string().uuid(),
	mimeType: z.string().min(1),
	sizeBytes: z.number().nonnegative(),
	checksum: z.string().min(1),
	publicUrl: z.string().url().optional(),
})

export const ZCommunityBook = z
	.object({
		catalogBookId: z.string().uuid(),
		preferredAssetId: z.string().uuid(),
		owner: ZCommunityBookOwner,
		title: z.string().min(1).max(255),
		authors: z.string(),
		identifiers: z.record(z.string(), z.string()).default({}),
		language: z.string().optional(),
		sourceType: z.string().min(1),
		addedAt: z.string().datetime(),
		preferredAsset: ZCommunityBookAsset,
	})
	.extend(ZModel.omit({ deletedAt: true }).shape)

export const ZCommunityBookIDParams = z.object({
	id: z.string().uuid(),
})

export const ZCommunityBooksQuery = ZGetManyQuery.extend({
	q: z.string().optional(),
	ownerUsername: z.string().optional(),
})

export const ZCommunityBooksResponse = ZPaginatedResponse(ZCommunityBook)
export const ZCommunitySaveBookResponse = ZUserLibraryBook
