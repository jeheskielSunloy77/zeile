import {
	ZCreateModerationReviewDTO,
	ZDecideModerationReviewDTO,
	ZModerationListQuery,
	ZModerationListResponse,
	ZModerationReview,
	ZModerationReviewIDParams,
} from '@zeile/zod'
import { initContract } from '@ts-rest/core'
import { failResponses, getSecurityMetadata } from '../utils.js'

const c = initContract()
const metadata = getSecurityMetadata({ security: true, securityType: 'bearerOrCookie' })

export const moderationContract = c.router({
	createReview: {
		summary: 'Create moderation review',
		description: 'Submit a catalog book for moderation verification.',
		path: '/api/v1/moderation/reviews',
		method: 'POST',
		body: ZCreateModerationReviewDTO,
		responses: {
			201: ZModerationReview,
			...failResponses,
		},
		metadata,
	},
	listReviews: {
		summary: 'List moderation reviews',
		description: 'List moderation review queue entries.',
		path: '/api/v1/moderation/reviews',
		method: 'GET',
		query: ZModerationListQuery,
		responses: {
			200: ZModerationListResponse,
			...failResponses,
		},
		metadata,
	},
	decideReview: {
		summary: 'Decide moderation review',
		description: 'Approve or reject a moderation review.',
		path: '/api/v1/moderation/reviews/:id/decision',
		method: 'PATCH',
		pathParams: ZModerationReviewIDParams,
		body: ZDecideModerationReviewDTO,
		responses: {
			200: ZModerationReview,
			...failResponses,
		},
		metadata,
	},
})
