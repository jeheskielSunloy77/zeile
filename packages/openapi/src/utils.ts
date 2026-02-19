import {
	ZForbiddenResponse,
	ZInternalServerErrorResponse,
	ZNotFoundResponse,
	ZUnauthorizedResponse,
} from '@zeile/zod'

export type SecurityType = 'bearer' | 'cookie' | 'bearerOrCookie'

export const getSecurityMetadata = ({
	security = true,
	securityType = 'bearerOrCookie',
}: {
	security?: boolean
	securityType?: SecurityType
} = {}) => {
	const openApiSecurity = (() => {
		switch (securityType) {
			case 'bearer':
				return [{ bearerAuth: [] }]
			case 'cookie':
				return [{ cookieAuth: [] }]
			case 'bearerOrCookie':
				return [{ bearerAuth: [] }, { cookieAuth: [] }]
			default: {
				const _exhaustive: never = securityType
				throw new Error(`Unhandled securityType: ${_exhaustive}`)
			}
		}
	})()

	return {
		...(security && { openApiSecurity }),
	}
}

export const failResponses = {
	401: ZUnauthorizedResponse,
	403: ZForbiddenResponse,
	404: ZNotFoundResponse,
	500: ZInternalServerErrorResponse,
} as const
