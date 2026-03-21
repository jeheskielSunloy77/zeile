import {
	ZCommunityBook,
	ZCommunityBookIDParams,
	ZCommunityBooksQuery,
	ZCommunityBooksResponse,
	ZCommunitySaveBookResponse,
} from '@kern/zod'
import { initContract } from '@ts-rest/core'
import { failResponses, getSecurityMetadata } from '../utils.js'

const c = initContract()
const metadata = getSecurityMetadata({ security: true, securityType: 'bearerOrCookie' })

export const communityContract = c.router({
	listBooks: {
		summary: 'List community books',
		description: 'List public user-owned books available in the community.',
		path: '/api/v1/community/books',
		method: 'GET',
		query: ZCommunityBooksQuery,
		responses: {
			200: ZCommunityBooksResponse,
			...failResponses,
		},
		metadata,
	},
	getBook: {
		summary: 'Get community book',
		description: 'Get one public user-owned book from the community.',
		path: '/api/v1/community/books/:id',
		method: 'GET',
		pathParams: ZCommunityBookIDParams,
		responses: {
			200: ZCommunityBook,
			...failResponses,
		},
		metadata,
	},
	saveBook: {
		summary: 'Save community book',
		description: 'Save a public community book into the current user library.',
		path: '/api/v1/community/books/:id/save',
		method: 'POST',
		pathParams: ZCommunityBookIDParams,
		responses: {
			200: ZCommunitySaveBookResponse,
			...failResponses,
		},
		metadata,
	},
})
