package token

type BaseToken interface {
	Token() string
	UUID() string
	Groups() []string
	IsLoggedIn() bool
	RefreshError() error
}
