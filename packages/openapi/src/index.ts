import { extendZodWithOpenApi } from '@anatine/zod-openapi'
import { generateOpenApi } from '@ts-rest/open-api'
import { z } from 'zod'

extendZodWithOpenApi(z)

import { apiContract } from './contracts/index.js'

type SecurityRequirementObject = {
	[key: string]: string[]
}

export type OperationMapper = NonNullable<
	Parameters<typeof generateOpenApi>[2]
>['operationMapper']

const hasSecurity = (
	metadata: unknown
): metadata is { openApiSecurity: SecurityRequirementObject[] } => {
	return (
		!!metadata && typeof metadata === 'object' && 'openApiSecurity' in metadata
	)
}

const operationMapper: OperationMapper = (operation, appRoute) => ({
	...operation,
	...(hasSecurity(appRoute.metadata)
		? {
				security: appRoute.metadata.openApiSecurity,
			}
		: {}),
})

export const OpenAPI = Object.assign(
	generateOpenApi(
		apiContract,
		{
			openapi: '3.0.2',
			info: {
				version: '1.0.0',
				title: 'zeile REST API - Documentation',
				description: 'zeile REST API - Documentation',
			},
			servers: [
				{
					url: 'http://localhost:8080',
					description: 'Local Server',
				},
			],
		},
		{
			operationMapper,
			setOperationId: 'concatenated-path',
		}
	),
	{
		components: {
			securitySchemes: {
				cookieAuth: {
					type: 'apiKey',
					name: 'access_token',
					in: 'cookie',
				},
				bearerAuth: {
					type: 'http',
					scheme: 'bearer',
					bearerFormat: 'JWT',
				},
			},
		},
	}
)
