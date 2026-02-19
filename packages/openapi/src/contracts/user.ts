import { ZStoreUserDTO, ZUpdateUserDTO, ZUser } from '@zeile/zod'

import { createResourceContract } from './resource.js'

export const userContract = createResourceContract({
	path: '/api/v1/users',
	resource: 'User',
	resourcePlural: 'Users',
	schemas: {
		entity: ZUser,
		createDTO: ZStoreUserDTO,
		updateDTO: ZUpdateUserDTO,
	},
})
