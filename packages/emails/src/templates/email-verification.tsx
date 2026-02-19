import { Heading, Section, Text } from '@react-email/components'
import { EmailLayout } from '../components/email-layout.js'

interface EmailVerificationProps {
	username: string
	verificationCode: string
	expiresInMinutes: string
}

export const EmailVerification = ({
	username = '{{.Username}}',
	verificationCode = '{{.VerificationCode}}',
	expiresInMinutes = '{{.ExpiresInMinutes}}',
}: EmailVerificationProps) => {
	return (
		<EmailLayout preview='Verify your email'>
			<Heading className='text-2xl font-bold text-gray-800 mt-4'>
				Verify your email
			</Heading>

			<Section>
				<Text className='text-gray-700 text-base'>Hi {username},</Text>
				<Text className='text-gray-700 text-base'>
					Use the code below to confirm your email address.
				</Text>
			</Section>

			<Section className='my-8 text-center'>
				<Text className='text-3xl font-bold tracking-[0.3em] bg-gray-100 inline-block px-6 py-3 rounded-md'>
					{verificationCode}
				</Text>
			</Section>

			<Text className='text-gray-600 text-sm'>
				This code expires in {expiresInMinutes} minutes.
			</Text>
			<Text className='text-gray-500 text-xs'>
				If you didn't request this, you can ignore this email.
			</Text>
		</EmailLayout>
	)
}

EmailVerification.PreviewProps = {
	username: 'John',
	verificationCode: '123456',
	expiresInMinutes: '30',
}

export default EmailVerification
