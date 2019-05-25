package handler

import (
	"crypto/rsa"
	"log"
	"time"

	"code.vegaprotocol.io/vega/internal/auth/handler/keys"
	"github.com/dgrijalva/jwt-go"
)

var (
	verifyKey *rsa.PublicKey
	signKey   *rsa.PrivateKey
	tokenTTL  = time.Hour * 24 * 365
)

type CustomClaims struct {
	*jwt.StandardClaims
	*PartyID
}

type PartyID struct {
	ID string `json:"id"`
}

func InitJWT() {
	var err error
	signBytes, _ := keys.Asset("keys/jwt.priv")
	signKey, err = jwt.ParseRSAPrivateKeyFromPEM(signBytes)
	if err != nil {
		log.Fatalf("error: parsing jwt private rsa key, err=%v", err)
	}

	verifyBytes, _ := keys.Asset("keys/jwt.pub")
	verifyKey, err = jwt.ParseRSAPublicKeyFromPEM(verifyBytes)
	if err != nil {
		log.Fatalf("error: parsing jwt public rsa key, err=%v", err)
	}
}

func createJWTToken(id string) (string, error) {
	// create a signer for rsa 256
	t := jwt.New(jwt.GetSigningMethod("RS256"))

	// set our claims
	t.Claims = &CustomClaims{
		&jwt.StandardClaims{
			ExpiresAt: time.Now().Add(tokenTTL).Unix(),
		},
		&PartyID{id},
	}

	// Creat token string
	return t.SignedString(signKey)
}

func VerifyJWTToken(tokenString string) (PartyID, error) {
	token, err := jwt.ParseWithClaims(
		tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
			return verifyKey, nil
		})
	if err != nil {
		return PartyID{}, err
	}

	claims := token.Claims.(*CustomClaims)
	return *claims.PartyID, nil
}
