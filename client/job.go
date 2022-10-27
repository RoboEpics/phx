package client

type jobClient struct {
	Client
}

func JobClient(baseClient Client) bucketClient {
	return bucketClient{
		baseClient.For("jobs"),
	}
}
