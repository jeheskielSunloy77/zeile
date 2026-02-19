import {
	ZAuthGoogleCallbackQuery,
	ZAuthLoginDTO,
	ZAuthRegisterDTO,
	ZAuthResult,
	ZAuthVerifyEmailDTO,
	ZAuthVerifyEmailResponse,
	ZEmpty,
	ZResponse,
	ZUser,
} from '@zeile/zod'
import { initContract } from '@ts-rest/core'
import { failResponses } from '../utils.js'

const c = initContract()

export const authContract = c.router({
	register: {
		summary: 'Register',
		description: 'Register a new user',
		path: '/api/v1/auth/register',
		method: 'POST',
		body: ZAuthRegisterDTO,
		responses: {
			201: ZAuthResult,
			...failResponses,
		},
	},
	login: {
		summary: 'Login',
		description: 'Login with email/username and password',
		path: '/api/v1/auth/login',
		method: 'POST',
		body: ZAuthLoginDTO,
		responses: {
			200: ZAuthResult,
			...failResponses,
		},
	},
	googleLogin: {
		summary: 'Google login',
		description: 'Redirect to Google OAuth',
		path: '/api/v1/auth/google',
		method: 'GET',
		responses: {
			302: ZEmpty,
		},
	},
	googleCallback: {
		summary: 'Google login callback',
		description: 'Handle Google OAuth callback and redirect',
		path: '/api/v1/auth/google/callback',
		method: 'GET',
		query: ZAuthGoogleCallbackQuery,
		responses: {
			302: ZEmpty,
		},
	},
	verifyEmail: {
		summary: 'Verify email',
		description: 'Verify user email using a verification code',
		path: '/api/v1/auth/verify-email',
		method: 'POST',
		body: ZAuthVerifyEmailDTO,
		responses: {
			200: ZAuthVerifyEmailResponse,
			...failResponses,
		},
	},
	refresh: {
		summary: 'Refresh session',
		description: 'Refresh access using the refresh cookie',
		path: '/api/v1/auth/refresh',
		method: 'POST',
		body: ZEmpty,
		responses: {
			200: ZAuthResult,
			...failResponses,
		},
	},
	me: {
		summary: 'Get current user',
		description: 'Return the current authenticated user',
		path: '/api/v1/auth/me',
		method: 'GET',
		responses: {
			200: ZUser,
			...failResponses,
		},
	},
	resendVerification: {
		summary: 'Resend verification',
		description: 'Resend the email verification code',
		path: '/api/v1/auth/resend-verification',
		method: 'POST',
		body: ZEmpty,
		responses: {
			200: ZResponse,
			...failResponses,
		},
	},
	logout: {
		summary: 'Logout',
		description: 'Logout the current session',
		path: '/api/v1/auth/logout',
		method: 'POST',
		body: ZEmpty,
		responses: {
			200: ZResponse,
			...failResponses,
		},
	},
	logoutAll: {
		summary: 'Logout all',
		description: 'Logout from all sessions',
		path: '/api/v1/auth/logout-all',
		method: 'POST',
		body: ZEmpty,
		responses: {
			200: ZResponse,
			...failResponses,
		},
	},
})
