import { initContract } from '@ts-rest/core'
import { authContract } from './auth.js'
import { communityContract } from './community.js'
import { healthContract } from './health.js'
import { libraryContract } from './library.js'
import { moderationContract } from './moderation.js'
import { sharingContract } from './sharing.js'
import { userContract } from './user.js'

const c = initContract()

export const apiContract = c.router({
	health: healthContract,
	auth: authContract,
	user: userContract,
	library: libraryContract,
	sharing: sharingContract,
	community: communityContract,
	moderation: moderationContract,
})
