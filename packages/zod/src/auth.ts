import { z } from 'zod'

import { ZStoreUserDTO, ZUser } from './user.js'

export const ZAuthToken = z.object({
	token: z.string(),
	expiresAt: z.string().datetime(),
})

export const ZAuthResult = ZUser

export const ZAuthRefreshDTO = z.object({
	refreshToken: z.string().min(32).max(256).optional(),
})

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

export const ZAuthDeviceStartResponse = z.object({
	deviceCode: z.string(),
	userCode: z.string(),
	verificationUri: z.string(),
	expiresAt: z.string().datetime(),
	intervalSeconds: z.number().int().min(1),
})

export const ZAuthDevicePollDTO = z.object({
	deviceCode: z.string().min(16).max(256),
})

export const ZAuthDeviceApproveDTO = z.object({
	userCode: z.string().min(4).max(32),
})

export const ZAuthDevicePollResponse = z.object({
	status: z.enum(['pending', 'approved']),
	expiresAt: z.string().datetime().optional(),
	intervalSeconds: z.number().int().min(1).optional(),
	user: ZUser.optional(),
	token: ZAuthToken.optional(),
	refreshToken: ZAuthToken.optional(),
})
