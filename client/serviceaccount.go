package client

type saClient struct {
	Client
}

func ServiceAccountClient(baseClient Client) saClient {
	return saClient{
		baseClient.For("serviceAccounts"),
	}
}
