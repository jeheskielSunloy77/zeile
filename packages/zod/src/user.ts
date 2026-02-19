import { z } from 'zod'
import { ZModel } from './utils.js'

export const ZUser = z
	.object({
		email: z.string().email(),
		username: z.string().min(3).max(50),
		googleId: z.string().optional(),
		emailVerifiedAt: z.string().datetime().optional(),
		lastLoginAt: z.string().datetime().optional(),
		isAdmin: z.boolean().default(false),
	})
	.extend(ZModel.shape)

export const ZStoreUserDTO = ZUser.pick({
	email: true,
	username: true,
	googleId: true,
}).extend({
	password: z.string().min(8).max(128),
})

export const ZUpdateUserDTO = ZUser.pick({
	email: true,
	username: true,
})
