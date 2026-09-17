package gql

func GetUserAgent(_ string) string {
	return UserAgents["Android"]["TV"]
}
