import { z } from 'zod'

export const ZResponse = z.object({
	status: z.number().int().default(200),
	message: z.string().default('Request processed successfully.'),
	success: z.boolean().default(true),
})

export const ZEmpty = z.object({}).strict()

export function ZResponseWithData<T>(schema: z.ZodSchema<T>) {
	return z.object({ data: schema }).extend(ZResponse.shape)
}

export function ZPaginatedResponse<T>(schema: z.ZodSchema<T>) {
	return ZResponse.extend(
		z.object({
			total: z.number().default(946),
			page: z.number(),
			limit: z.number().default(20),
			totalPages: z.number().default(Math.ceil(946 / 20)),
			data: z.array(schema),
			message: z.string().default('Fetched paginated data successfully!'),
			status: z.literal(200),
		}).shape
	)
}

export const ZModel = z.object({
	id: z.string().uuid(),
	createdAt: z.string().datetime(),
	updatedAt: z.string().datetime(),
	deletedAt: z.string().datetime().optional(),
})

export const ZGetManyQuery = z.object({
	limit: z.coerce.number().int().nonnegative().optional(),
	offset: z.coerce.number().int().nonnegative().optional(),
	preloads: z.string().optional(),
	orderBy: z.string().optional(),
	orderDirection: z.enum(['asc', 'desc']).optional(),
})

export const ZPreloadsQuery = z.object({
	preloads: z.string().optional(),
})

export const ZUnauthorizedResponse = ZResponse.extend({
	status: z.literal(401),
	message: z
		.string()
		.default('Sorry, you are not authorized to access this resource.'),
	success: z.literal(false),
})

export const ZForbiddenResponse = ZResponse.extend({
	status: z.literal(403),
	message: z
		.string()
		.default('Sorry, you do not have permission to access this resource.'),
	success: z.literal(false),
})

export const ZNotFoundResponse = ZResponse.extend({
	status: z.literal(404),
	message: z.string().default('The requested resource was not found.'),
	success: z.literal(false),
})

export const ZInternalServerErrorResponse = ZResponse.extend({
	status: z.literal(500),
	message: z
		.string()
		.default('Sorry, something went wrong on our end. Please try again later.'),
	success: z.literal(false),
})
