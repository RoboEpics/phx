package token

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/golang-jwt/jwt/v4"
)

var (
	defaultInstanceAddress     = "http://staging.phoenix.roboepics.com"
	defaultOAuthServiceAddress = "https://fusion.roboepics.com"

	configDir           = homePath("/.phoenix")
	credentialsFilePath = homePath("/.phoenix/credentials.json")

	jwksFilePath = homePath("/.phoenix/jwks.json") // TODO change to PhoenixPublicKeyPEMFilePath
	jwksURL      = defaultOAuthServiceAddress + "/.well-known/jwks.json"
	jwkKID       = "05Xm2o5zBB4h2niEfyJXAZkL8ww"

	fusionOAuthBaseURL = "https://fusion.roboepics.com/oauth2"
	fusionClientID     = "c5c41330-c4a0-4ab0-922a-502ea24f5320"

	loginPath = "/api/login"
)

type LoginCredentialsRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginCredentialsResponse struct {
	AccessToken  string `json:"token"`
	RefreshToken string `json:"refreshToken"`
}

type FusionAuthGrantRefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type JWTToken struct {
	instanceAddress string
	accessToken     string
	refreshToken    string
	publicKey       *ecdsa.PublicKey
	payload         *Payload
	refreshErr      error
}

var _ BaseToken = (*JWTToken)(nil)

func NewJWTToken(instanceAddress, accessToken, refreshToken string) (*JWTToken, error) {
	return &JWTToken{
		instanceAddress: instanceAddress,
		accessToken:     accessToken,
		refreshToken:    refreshToken,
	}, nil
}

func NewDefaultJWTToken() *JWTToken {
	jwtToken := &JWTToken{
		instanceAddress: defaultInstanceAddress,
	}
	jwtToken.LoadCredentialsFromDefaultPath()
	return jwtToken
}

func NewDefaultJWTTokenWithCredentials(username, password string) (*JWTToken, error) {
	t := &JWTToken{
		instanceAddress: defaultInstanceAddress,
	}

	err := t.Login(username, password)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func NewJWTTokenWithCredentials(instanceAddress, username, password string) (*JWTToken, error) {
	t := &JWTToken{
		instanceAddress: instanceAddress,
	}

	err := t.Login(username, password)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (t *JWTToken) Token() string {
	if t == nil {
		return ""
	}
	if err := t.ensureToken(); err != nil {
		t.refreshErr = err
		return ""
	}
	return t.accessToken
}

func (t *JWTToken) UUID() string {
	if t == nil {
		return ""
	}
	if err := t.ensureToken(); err != nil {
		t.refreshErr = err
		return ""
	}
	if t.payload == nil {
		return ""
	}
	return t.payload.UserID
}

func (t *JWTToken) Groups() []string {
	if t == nil {
		return nil
	}
	if err := t.ensureToken(); err != nil {
		t.refreshErr = err
		return nil
	}
	if t.payload == nil {
		return nil
	}
	return t.payload.Roles
}

func (t *JWTToken) IsLoggedIn() bool {
	if t == nil {
		return false
	}
	if err := t.ensureToken(); err != nil {
		t.refreshErr = err
		return false
	}
	return t.accessToken != ""
}

func (t *JWTToken) RefreshError() error {
	err := t.refreshErr
	t.refreshErr = nil
	return err
}

func (t *JWTToken) LoadCredentialsFromFile(file *os.File) error {
	content, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	var jwtToken LoginCredentialsResponse
	err = json.Unmarshal(content, &jwtToken)
	if err != nil {
		return err
	}

	t.accessToken = jwtToken.AccessToken
	t.refreshToken = jwtToken.RefreshToken

	return nil
}

func (t *JWTToken) LoadCredentialsFromDefaultPath() error {
	file, err := os.Open(credentialsFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		log.Println("Cannot read creds file:", err)
		return err
	}
	defer file.Close()
	return t.LoadCredentialsFromFile(file)
}

func (t *JWTToken) Login(username, password string) error {
	if len(username)*len(password) == 0 {
		return fmt.Errorf("empty username/password")
	}

	buffer := &bytes.Buffer{}
	if err := json.NewEncoder(buffer).Encode(&LoginCredentialsRequest{
		Username: username,
		Password: password,
	}); err != nil {
		panic("failed to encode request body")
	}

	response, err := http.Post(t.instanceAddress+loginPath, "application/json", buffer)
	if err != nil {
		return fmt.Errorf("error while requesting server: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid credentials")
	}

	var loginResponse LoginCredentialsResponse
	if err := json.NewDecoder(response.Body).Decode(&loginResponse); err != nil {
		return fmt.Errorf("could not parse server response: %v", err)
	}

	content, err := json.Marshal(loginResponse)
	if err != nil {
		return fmt.Errorf("failed to marshal the login response: %v", err)
	}

	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to mkdir %s: %w", configDir, err)
	}
	if err = os.WriteFile(credentialsFilePath, content, 0600); err != nil {
		return fmt.Errorf("failed to write credentials to '%s': %w", credentialsFilePath, err)
	}

	t.accessToken = loginResponse.AccessToken
	t.refreshToken = loginResponse.RefreshToken

	return nil
}

func (t *JWTToken) ensureToken() error {
	if t.accessToken == "" {
		// Try to refresh immediately if access token is not provided
		return t.refresh()
	}

	publicKey, err := t.loadPublicKey()
	if err != nil {
		return err
	}

	parser := Parser{
		PublicKey: publicKey,
	}
	payload, err := parser.ParseAndValidate(t.accessToken)
	if err != nil {
		return t.refresh()
	}
	t.payload = payload
	return nil
}

func (t *JWTToken) refresh() error {
	if t.refreshToken == "" {
		return fmt.Errorf("refresh token is empty")
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", t.refreshToken)
	data.Set("client_id", fusionClientID)

	response, err := http.Post(fusionOAuthBaseURL+"/token", "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))

	if err != nil {
		return fmt.Errorf("error while requesting RoboEpics FusionAuth server: %v", err)
	}

	// 400: Invalid refresh token
	// 401: Invalid client id
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("server failed to grant new refresh token: %v", err)
	}

	var fusionResponse FusionAuthGrantRefreshTokenResponse
	if err := json.NewDecoder(response.Body).Decode(&fusionResponse); err != nil {
		return fmt.Errorf("could not parse server response: %v", err)
	}

	content, err := json.Marshal(&LoginCredentialsResponse{
		AccessToken:  fusionResponse.AccessToken,
		RefreshToken: fusionResponse.RefreshToken,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %v", err)
	}

	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to mkdir %s: %w", configDir, err)
	}
	if err = os.WriteFile(credentialsFilePath, content, 0600); err != nil {
		return fmt.Errorf("failed to write credentials to '%s': %v", credentialsFilePath, err)
	}

	t.accessToken = fusionResponse.AccessToken
	t.refreshToken = fusionResponse.RefreshToken

	return nil
}

type JWKFull struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	KID        string   `json:"kid"`
	PublicKeys []string `json:"x5c"`
}

func (t *JWTToken) loadPublicKey() (*ecdsa.PublicKey, error) {
	if t.publicKey != nil {
		return t.publicKey, nil
	}

	var JWKS []byte
	var publicKeyPEM string

	// Check if the JWKS file is already stored in the expected path
	fileContent, err := os.ReadFile(jwksFilePath)
	if err == nil {
		JWKS = fileContent
	} else {
		func() {
			// Download the JWKS file
			resp, err := http.Get(jwksURL)
			if err != nil {
				log.Fatal("Could not retrieve JWT public key!", err)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Fatal("Could not retrieve JWT public key!", err)
			}
			defer resp.Body.Close()
			JWKS = body
		}()

		// Try to store it in the expected path for future use
		func() {
			if err := os.MkdirAll(configDir, 0700); err != nil {
				log.Printf("failed to mkdir %s: %s\n", configDir, err)
				return
			}
			file, err := os.Create(jwksFilePath)
			if err != nil {
				log.Printf("JWT public key file could not be stored in %s! The file will be downloaded again the next time this program is run.\n", jwksFilePath)
				return
			}
			defer file.Close()
			// Write the JWKS response body to file
			if size, err := file.Write(JWKS); err != nil || size != len(JWKS) {
				log.Printf("There was a problem storing the JWT public key file %s! The file may be downloaded again the next time this program is run.\n", jwksFilePath)
			}
		}()
	}

	// Unmarshal the JWKS data and extract the Phoenix Public Key
	var jwkFull JWKFull
	err = json.Unmarshal(JWKS, &jwkFull)
	if err != nil {
		return nil, err
	}

	for _, v := range jwkFull.Keys {
		if v.KID == jwkKID {
			publicKeyPEM = v.PublicKeys[0]
		}
	}

	// Parse the PEM public key
	publicKeyParsed, err := jwt.ParseECPublicKeyFromPEM([]byte("-----BEGIN PUBLIC KEY-----\n" + publicKeyPEM + "\n-----END PUBLIC KEY-----"))
	if err != nil {
		return nil, err
	}
	t.publicKey = publicKeyParsed
	return publicKeyParsed, nil
}

func home() string {
	h, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return h
}

func homePath(parts ...string) string {
	homep := append([]string{home()}, parts...)
	return path.Join(homep...)
}
