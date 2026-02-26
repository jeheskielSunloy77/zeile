import { z } from 'zod'
import { ZModel, ZPaginatedResponse, ZResponse } from './utils.js'
import { ZBookAsset, ZHighlight, ZUserLibraryBook, ZVisibility } from './library.js'

export const ZShareList = z
	.object({
		userId: z.string().uuid(),
		name: z.string().min(1).max(120),
		description: z.string().optional(),
		visibility: ZVisibility,
		isPublished: z.boolean(),
		publishedAt: z.string().datetime().optional(),
	})
	.extend(ZModel.shape)

export const ZShareListItem = z
	.object({
		listId: z.string().uuid(),
		itemType: z.enum(['book', 'highlight']),
		userLibraryBookId: z.string().uuid().optional(),
		highlightId: z.string().uuid().optional(),
		position: z.number().int().min(0),
	})
	.extend(ZModel.shape)

export const ZBookSharePolicy = z
	.object({
		userId: z.string().uuid(),
		userLibraryBookId: z.string().uuid(),
		rawFileSharing: z.enum(['private', 'public_link']),
		allowMetadataSharing: z.boolean(),
	})
	.extend(ZModel.shape)

export const ZShareLink = z
	.object({
		userId: z.string().uuid(),
		resourceType: z.enum(['list', 'highlight', 'book_file']),
		resourceId: z.string().uuid(),
		token: z.string().min(1),
		requiresAuth: z.boolean(),
		isActive: z.boolean(),
		expiresAt: z.string().datetime().optional(),
	})
	.extend(ZModel.shape)

export const ZResolvedShareResource = z.object({
	resourceType: z.enum(['list', 'highlight', 'book_file']),
	link: ZShareLink,
	shareList: ZShareList.optional(),
	highlight: ZHighlight.optional(),
	libraryBook: ZUserLibraryBook.optional(),
	bookAsset: ZBookAsset.optional(),
})

export const ZCreateShareListDTO = z.object({
	name: z.string().min(1).max(120),
	description: z.string().max(1000).optional(),
	visibility: ZVisibility.optional(),
})

export const ZUpdateShareListDTO = z.object({
	name: z.string().min(1).max(120).optional(),
	description: z.string().max(1000).optional(),
	visibility: ZVisibility.optional(),
	isPublished: z.boolean().optional(),
})

export const ZCreateShareListItemDTO = z.object({
	itemType: z.enum(['book', 'highlight']),
	userLibraryBookId: z.string().uuid().optional(),
	highlightId: z.string().uuid().optional(),
	position: z.number().int().min(0).default(0),
})

export const ZUpsertBookSharePolicyDTO = z.object({
	userLibraryBookId: z.string().uuid(),
	rawFileSharing: z.enum(['private', 'public_link']),
	allowMetadataSharing: z.boolean(),
})

export const ZCreateShareLinkDTO = z.object({
	resourceType: z.enum(['list', 'highlight', 'book_file']),
	resourceId: z.string().uuid(),
	expiresAt: z.string().datetime().optional(),
})

export const ZShareListIDParams = z.object({
	id: z.string().uuid(),
})

export const ZShareLinkIDParams = z.object({
	id: z.string().uuid(),
})

export const ZShareResolveParams = z.object({
	token: z.string().min(1),
})

export const ZShareListsListResponse = ZPaginatedResponse(ZShareList)
export const ZShareListItemsResponse = z.array(ZShareListItem)
export const ZShareLinksRevokeResponse = ZResponse
