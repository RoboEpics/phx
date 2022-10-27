package token

type StaticToken struct {
	apiToken string
	uuid     string
	groups   []string
}

var _ BaseToken = (*StaticToken)(nil)

func NewStaticToken(apiToken, uuid string, groups []string) *StaticToken {
	return &StaticToken{
		apiToken: apiToken,
		uuid:     uuid,
		groups:   groups,
	}
}

func (t *StaticToken) Token() string {
	return t.apiToken
}

func (t *StaticToken) UUID() string {
	return t.uuid
}

func (t *StaticToken) Groups() []string {
	return t.groups
}

func (t *StaticToken) IsLoggedIn() bool {
	return t.apiToken != ""
}

func (t *StaticToken) RefreshError() error {
	return nil
}
