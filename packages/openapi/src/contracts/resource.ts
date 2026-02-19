import {
	ZGetManyQuery,
	ZPaginatedResponse,
	ZPreloadsQuery,
	ZResponse,
	ZResponseWithData,
} from '@zeile/zod'
import { initContract } from '@ts-rest/core'
import { z } from 'zod'
import {
	failResponses,
	getSecurityMetadata,
	type SecurityType,
} from '../utils.js'

type ResourceContractOptions = {
	path: string
	resource: string
	resourcePlural: string
	schemas: {
		entity: z.ZodTypeAny
		createDTO: z.ZodTypeAny
		updateDTO: z.ZodTypeAny
	}
	security?: boolean
	securityType?: SecurityType
}

const c = initContract()

const idParams = z.object({
	id: z.string().uuid(),
})

export const createResourceContract = ({
	path,
	resource,
	resourcePlural,
	schemas,
	security = true,
	securityType = 'bearerOrCookie',
}: ResourceContractOptions) => {
	const metadata = getSecurityMetadata({ security, securityType })

	return c.router({
		getMany: {
			summary: `Get Many ${resourcePlural}`,
			description: `Retrieve a paginated list of ${resourcePlural} that can be filtered, sorted, and preloaded.`,
			path,
			method: 'GET',
			query: ZGetManyQuery,
			responses: {
				200: ZPaginatedResponse(schemas.entity),
				...failResponses,
			},
			metadata,
		},
		getById: {
			summary: `Get ${resource} by ID`,
			description: `Retrieve a single ${resource} by its unique identifier (ID), with optional preloaded relationships.`,
			path: `${path}/:id`,
			method: 'GET',
			pathParams: idParams,
			query: ZPreloadsQuery,
			responses: {
				200: ZResponseWithData(schemas.entity),
				...failResponses,
			},
			metadata,
		},
		store: {
			summary: `Store ${resource}`,
			description: `Create a new ${resource} with the provided data, with validation and will return the created entity.`,
			path,
			method: 'POST',
			body: schemas.createDTO,
			responses: {
				201: ZResponseWithData(schemas.entity),
				...failResponses,
			},
			metadata,
		},
		update: {
			summary: `Update ${resource}`,
			description: `Update an existing ${resource} by its ID with the provided data, and return the updated entity.`,
			path: `${path}/:id`,
			method: 'PATCH',
			pathParams: idParams,
			body: schemas.updateDTO,
			responses: {
				200: ZResponseWithData(schemas.entity),
				...failResponses,
			},
			metadata,
		},
		destroy: {
			summary: `Destroy ${resource}`,
			description: `Soft delete the specified ${resource} by its ID. This action is reversible.`,
			path: `${path}/:id`,
			method: 'DELETE',
			pathParams: idParams,
			responses: {
				200: ZResponse,
				...failResponses,
			},
			metadata,
		},
		kill: {
			summary: `Kill ${resource}`,
			description: `Permanently delete the specified ${resource} by its ID. This action is irreversible.`,
			path: `${path}/:id/kill`,
			method: 'DELETE',
			pathParams: idParams,
			responses: {
				200: ZResponse,
				...failResponses,
			},
			metadata,
		},
		restore: {
			summary: `Restore ${resource}`,
			description: `Restore a previously soft-deleted ${resource} by its ID and then return the restored entity.`,
			path: `${path}/:id/restore`,
			method: 'PATCH',
			pathParams: idParams,
			query: ZPreloadsQuery,
			body: ZResponse,
			responses: {
				200: ZResponseWithData(schemas.entity),
				...failResponses,
			},
			metadata,
		},
	})
}
