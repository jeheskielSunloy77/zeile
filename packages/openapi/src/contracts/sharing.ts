import {
	ZBookSharePolicy,
	ZCreateShareLinkDTO,
	ZCreateShareListDTO,
	ZCreateShareListItemDTO,
	ZGetManyQuery,
	ZResolvedShareResource,
	ZShareLink,
	ZShareLinksRevokeResponse,
	ZShareLinkIDParams,
	ZShareList,
	ZShareListIDParams,
	ZShareListItem,
	ZShareListItemsResponse,
	ZShareListsListResponse,
	ZShareResolveParams,
	ZUpdateShareListDTO,
	ZUpsertBookSharePolicyDTO,
} from '@zeile/zod'
import { initContract } from '@ts-rest/core'
import { failResponses, getSecurityMetadata } from '../utils.js'

const c = initContract()
const metadata = getSecurityMetadata({ security: true, securityType: 'bearerOrCookie' })

export const sharingContract = c.router({
	createList: {
		summary: 'Create share list',
		description: 'Create a share list for authenticated user.',
		path: '/api/v1/sharing/lists',
		method: 'POST',
		body: ZCreateShareListDTO,
		responses: {
			201: ZShareList,
			...failResponses,
		},
		metadata,
	},
	listLists: {
		summary: 'List share lists',
		description: 'List share lists of authenticated user.',
		path: '/api/v1/sharing/lists',
		method: 'GET',
		query: ZGetManyQuery,
		responses: {
			200: ZShareListsListResponse,
			...failResponses,
		},
		metadata,
	},
	updateList: {
		summary: 'Update share list',
		description: 'Update share list details and publish state.',
		path: '/api/v1/sharing/lists/:id',
		method: 'PATCH',
		pathParams: ZShareListIDParams,
		body: ZUpdateShareListDTO,
		responses: {
			200: ZShareList,
			...failResponses,
		},
		metadata,
	},
	createListItem: {
		summary: 'Create share list item',
		description: 'Add a book or highlight to a share list.',
		path: '/api/v1/sharing/lists/:id/items',
		method: 'POST',
		pathParams: ZShareListIDParams,
		body: ZCreateShareListItemDTO,
		responses: {
			201: ZShareListItem,
			...failResponses,
		},
		metadata,
	},
	listListItems: {
		summary: 'List share list items',
		description: 'List items in a share list.',
		path: '/api/v1/sharing/lists/:id/items',
		method: 'GET',
		pathParams: ZShareListIDParams,
		responses: {
			200: ZShareListItemsResponse,
			...failResponses,
		},
		metadata,
	},
	upsertBookSharePolicy: {
		summary: 'Upsert book share policy',
		description: 'Update sharing policy for one library book.',
		path: '/api/v1/sharing/book-share-policies',
		method: 'PUT',
		body: ZUpsertBookSharePolicyDTO,
		responses: {
			200: ZBookSharePolicy,
			...failResponses,
		},
		metadata,
	},
	createShareLink: {
		summary: 'Create share link',
		description: 'Create share link for list, highlight, or book file.',
		path: '/api/v1/sharing/links',
		method: 'POST',
		body: ZCreateShareLinkDTO,
		responses: {
			201: ZShareLink,
			...failResponses,
		},
		metadata,
	},
	revokeShareLink: {
		summary: 'Revoke share link',
		description: 'Deactivate a share link owned by authenticated user.',
		path: '/api/v1/sharing/links/:id/revoke',
		method: 'POST',
		pathParams: ZShareLinkIDParams,
		responses: {
			200: ZShareLinksRevokeResponse,
			...failResponses,
		},
		metadata,
	},
	resolveShareLink: {
		summary: 'Resolve share link',
		description: 'Resolve a share link token into resource payload.',
		path: '/api/v1/sharing/resolve/:token',
		method: 'GET',
		pathParams: ZShareResolveParams,
		responses: {
			200: ZResolvedShareResource,
			...failResponses,
		},
		metadata,
	},
})
