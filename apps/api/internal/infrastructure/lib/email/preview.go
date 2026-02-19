package email

var PreviewData = map[string]map[string]string{
	"welcome": {
		"UserFirstName": "John",
	},
	"email_verification": {
		"Username":         "John",
		"VerificationCode": "123456",
		"ExpiresInMinutes": "30",
	},
}
