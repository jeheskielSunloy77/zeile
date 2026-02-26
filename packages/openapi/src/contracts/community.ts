import {
	ZCommunityActivityResponse,
	ZCommunityProfile,
	ZCommunityProfilePathParams,
	ZGetManyQuery,
	ZUpdateCommunityProfileDTO,
} from '@zeile/zod'
import { initContract } from '@ts-rest/core'
import { failResponses, getSecurityMetadata } from '../utils.js'

const c = initContract()
const metadata = getSecurityMetadata({ security: true, securityType: 'bearerOrCookie' })

export const communityContract = c.router({
	getProfile: {
		summary: 'Get community profile',
		description: 'Get user community profile.',
		path: '/api/v1/community/profiles/:userId',
		method: 'GET',
		pathParams: ZCommunityProfilePathParams,
		responses: {
			200: ZCommunityProfile,
			...failResponses,
		},
		metadata,
	},
	updateMyProfile: {
		summary: 'Update my profile',
		description: 'Update current user profile settings.',
		path: '/api/v1/community/profile',
		method: 'PATCH',
		body: ZUpdateCommunityProfileDTO,
		responses: {
			200: ZCommunityProfile,
			...failResponses,
		},
		metadata,
	},
	listActivity: {
		summary: 'List profile activity',
		description: 'List activity feed events for a user.',
		path: '/api/v1/community/profiles/:userId/activity',
		method: 'GET',
		pathParams: ZCommunityProfilePathParams,
		query: ZGetManyQuery,
		responses: {
			200: ZCommunityActivityResponse,
			...failResponses,
		},
		metadata,
	},
})
