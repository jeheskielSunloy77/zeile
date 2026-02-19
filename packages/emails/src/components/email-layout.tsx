import {
	Body,
	Container,
	Head,
	Html,
	Preview,
	Section,
	Tailwind,
	Text,
} from '@react-email/components'
import type { ReactNode } from 'react'

type EmailLayoutProps = {
	preview: string
	children: ReactNode
	footer?: ReactNode
}

const DefaultFooter = () => (
	<Section className='mt-8 text-center'>
		<Text className='text-gray-500 text-xs'>
			Copyright {new Date().getFullYear()} zeile. All rights reserved.
		</Text>
		<Text className='text-gray-500 text-xs'>
			123 Project Street, Suite 100, San Francisco, CA 94103
		</Text>
	</Section>
)

export const EmailLayout = ({ preview, children, footer }: EmailLayoutProps) => {
	return (
		<Html>
			<Head />
			<Preview>{preview}</Preview>
			<Tailwind>
				<Body className='bg-gray-100 font-sans'>
					<Container className='bg-white p-8 rounded-lg shadow-sm my-10 mx-auto max-w-[600px]'>
						{children}
						{footer ?? <DefaultFooter />}
					</Container>
				</Body>
			</Tailwind>
		</Html>
	)
}
