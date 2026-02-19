import fs from 'fs'

import { OpenAPI } from './index.js'

const replaceCustomFileTypesToOpenApiCompatible = (
	jsonString: string
): string => {
	const searchPattern =
		/{"type":"object","properties":{"type":{"type":"string","enum":\["file"\]}},\s*"required":\["type"\]}/g
	const replacement = `{"type":"string","format":"binary"}`

	return jsonString.replace(searchPattern, replacement)
}

const filteredDoc = replaceCustomFileTypesToOpenApiCompatible(
	JSON.stringify(OpenAPI)
)

const formattedDoc = JSON.parse(filteredDoc)

const filePaths = ['./openapi.json', '../../apps/api/static/openapi.json']

filePaths.forEach((filePath) => {
	fs.writeFile(filePath, JSON.stringify(formattedDoc, null, 2), (err) => {
		if (err) {
			return console.error(`Error writing to ${filePath}:`, err)
		}
		console.log(`OpenAPI doc successfully written to ${filePath}`)
	})
})
