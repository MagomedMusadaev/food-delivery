package entities

type TokensResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"-"`
}
