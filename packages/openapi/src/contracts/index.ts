import { initContract } from '@ts-rest/core'
import { authContract } from './auth.js'
import { healthContract } from './health.js'
import { userContract } from './user.js'

const c = initContract()

export const apiContract = c.router({
	health: healthContract,
	auth: authContract,
	user: userContract,
})
