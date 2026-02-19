import { z } from 'zod'

import { ZStoreUserDTO, ZUser } from './user.js'

export const ZAuthToken = z.object({
	token: z.string(),
	expiresAt: z.string().datetime(),
})

export const ZAuthResult = ZUser

export const ZAuthRegisterDTO = ZStoreUserDTO.pick({
	email: true,
	username: true,
	password: true,
})

export const ZAuthLoginDTO = z.object({
	identifier: z.string(),
	password: z.string(),
})

export const ZAuthGoogleCallbackQuery = z.object({
	code: z.string(),
	state: z.string(),
})

export const ZAuthVerifyEmailDTO = z.object({
	email: z.string().email(),
	code: z.string().min(4).max(10),
})

export const ZAuthVerifyEmailResponse = ZUser
