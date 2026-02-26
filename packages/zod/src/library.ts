import { z } from 'zod'
import {
	ZGetManyQuery,
	ZModel,
	ZPaginatedResponse,
	ZResponse,
	ZResponseWithData,
} from './utils.js'

export const ZReadingMode = z.enum(['epub', 'pdf_text', 'pdf_layout'])
export const ZVisibility = z.enum(['private', 'authenticated'])
export const ZVerificationStatus = z.enum([
	'pending',
	'verified_public_domain',
	'rejected',
])

export const ZBookCatalog = z
	.object({
		title: z.string().min(1).max(255),
		authors: z.string().default(''),
		identifiers: z.record(z.string(), z.string()).default({}),
		language: z.string().optional(),
		verificationStatus: ZVerificationStatus,
		sourceType: z.string().default('user_upload'),
	})
	.extend(ZModel.shape)

export const ZBookAsset = z
	.object({
		catalogBookId: z.string().uuid(),
		uploaderUserId: z.string().uuid(),
		storagePath: z.string().min(1),
		publicUrl: z.string().url().optional(),
		mimeType: z.string().min(1),
		sizeBytes: z.number().nonnegative(),
		checksum: z.string().min(1),
		ingestStatus: z.enum(['pending', 'completed', 'failed']),
	})
	.extend(ZModel.shape)

export const ZUserLibraryBook = z
	.object({
		userId: z.string().uuid(),
		catalogBookId: z.string().uuid(),
		preferredAssetId: z.string().uuid().optional(),
		state: z.enum(['active', 'archived']),
		visibilityInProfile: z.boolean(),
		addedAt: z.string().datetime(),
		archivedAt: z.string().datetime().optional(),
	})
	.extend(ZModel.shape)

export const ZReadingState = z
	.object({
		userId: z.string().uuid(),
		userLibraryBookId: z.string().uuid(),
		mode: ZReadingMode,
		locatorJson: z.record(z.string(), z.any()).default({}),
		progressPercent: z.number().min(0).max(100),
		version: z.number().int().min(1),
	})
	.extend(ZModel.shape)

export const ZHighlight = z
	.object({
		userId: z.string().uuid(),
		userLibraryBookId: z.string().uuid(),
		mode: ZReadingMode,
		locatorJson: z.record(z.string(), z.any()).default({}),
		excerpt: z.string().optional(),
		visibility: ZVisibility,
		listId: z.string().uuid().optional(),
		isDeleted: z.boolean(),
	})
	.extend(ZModel.shape)

export const ZCreateCatalogBookDTO = z.object({
	title: z.string().min(1).max(255),
	authors: z.string().max(1024).default(''),
	identifiers: z.record(z.string(), z.string()).optional(),
	language: z.string().max(32).optional(),
	sourceType: z.string().max(64).optional(),
})

export const ZCreateLibraryBookDTO = z.object({
	catalogBookId: z.string().uuid(),
	preferredAssetId: z.string().uuid().optional(),
	visibilityInProfile: z.boolean().optional(),
})

export const ZUpdateLibraryBookDTO = z.object({
	state: z.enum(['active', 'archived']).optional(),
	preferredAssetId: z.string().uuid().optional(),
	visibilityInProfile: z.boolean().optional(),
})

export const ZUploadBookAssetDTO = z.object({
	catalogBookId: z.string().uuid(),
	checksum: z.string().optional(),
	file: z.any(),
})

export const ZUpsertReadingStateDTO = z.object({
	locatorJson: z.record(z.string(), z.any()).default({}),
	progressPercent: z.number().min(0).max(100),
	ifMatchVersion: z.number().int().min(1).optional(),
})

export const ZCreateHighlightDTO = z.object({
	mode: ZReadingMode,
	locatorJson: z.record(z.string(), z.any()).default({}),
	excerpt: z.string().max(2000).optional(),
	visibility: ZVisibility.optional(),
})

export const ZUpdateHighlightDTO = z.object({
	locatorJson: z.record(z.string(), z.any()).optional(),
	excerpt: z.string().max(2000).optional(),
	visibility: ZVisibility.optional(),
})

export const ZLibraryCatalogListResponse = ZPaginatedResponse(ZBookCatalog)
export const ZLibraryBooksListResponse = ZPaginatedResponse(ZUserLibraryBook)
export const ZLibraryHighlightsResponse = z.array(ZHighlight)

export const ZLibraryDeleteResponse = ZResponse
export const ZLibraryCatalogCreateResponse = ZResponseWithData(ZBookCatalog)

export const ZLibraryBookIDParams = z.object({
	id: z.string().uuid(),
})

export const ZLibraryHighlightIDParams = z.object({
	highlightId: z.string().uuid(),
})

export const ZReadingStatePathParams = z.object({
	id: z.string().uuid(),
	mode: ZReadingMode,
})

export const ZLibraryListQuery = ZGetManyQuery
