package jwt

import (
	"errors"
	"time"
)

var (
	// ErrExpired indicates that token is used after expiry time indicated in "exp" claim.
	ErrExpired = errors.New("token expired")
	// ErrNotValidYet indicates that token is used before time indicated in "nbf" claim.
	ErrNotValidYet = errors.New("token not valid yet")
	// ErrIssuedInTheFuture indicates that the "iat" claim is in the future.
	ErrIssuedInTheFuture = errors.New("token issued in the future")
)

// Claims holds the standard JWT claims (payload fields).
// It can be used to validate the JWT and to sign it.
// It completes the `SignOption` interface.
type Claims struct {
	// The opposite of the exp claim. A number representing a specific
	// date and time in the format “seconds since epoch” as defined by POSIX.
	// This claim sets the exact moment from which this JWT is considered valid.
	// The current time (see `Clock` package-level variable)
	// must be equal to or later than this date and time.
	NotBefore int64 `json:"nbf,omitempty"`
	// A number representing a specific date and time (in the same
	// format as exp and nbf) at which this JWT was issued.
	IssuedAt int64 `json:"iat,omitempty"`
	// A number representing a specific date and time in the
	// format “seconds since epoch” as defined by POSIX6.
	// This claims sets the exact moment from which
	// this JWT is considered invalid. This implementation allow for a certain skew
	// between clocks (by considering this JWT to be valid for a few minutes after the expiration
	// date, modify the `Clock` variable).
	Expiry int64 `json:"exp,omitempty"`
	// A string representing a unique identifier for this JWT. This claim may be
	// used to differentiate JWTs with other similar content (preventing replays, for instance). It is
	// up to the implementation to guarantee uniqueness
	ID string `json:"jti,omitempty"`
	// A string or URI that uniquely identifies the party
	// that issued the JWT. Its interpretation is application specific (there is no central authority
	// managing issuers).
	Issuer string `json:"iss,omitempty"`
	// A string or URI that uniquely identifies the party
	// that this JWT carries information about. In other words, the claims contained in this JWT
	// are statements about this party. The JWT spec specifies that this claim must be unique in
	// the context of the issuer or, in cases where that is not possible, globally unique. Handling of
	// this claim is application specific.
	Subject string `json:"sub,omitempty"`
	// Either a single string or URI or an array of such
	// values that uniquely identify the intended recipients of this JWT. In other words, when this
	// claim is present, the party reading the data in this JWT must find itself in the aud claim or
	// disregard the data contained in the JWT. As in the case of the iss and sub claims, this claim
	// is application specific.
	Audience []string `json:"aud,omitempty"`
}

// See TokenValidator and its implementations
// for further validation options.
func validateClaims(t time.Time, claims Claims) error {
	now := t.Round(time.Second).Unix()

	if claims.NotBefore > 0 {
		if now < claims.NotBefore {
			return ErrNotValidYet
		}
	}

	if claims.IssuedAt > 0 {
		if now < claims.IssuedAt {
			return ErrIssuedInTheFuture
		}
	}

	if claims.Expiry > 0 {
		if now > claims.Expiry {
			return ErrExpired
		}
	}

	return nil
}

// ApplyClaims implements the `SignOption` interface.
func (c Claims) ApplyClaims(dest *Claims) {
	if v := c.NotBefore; v > 0 {
		dest.NotBefore = v
	}

	if v := c.IssuedAt; v > 0 {
		dest.IssuedAt = v
	}

	if v := c.Expiry; v > 0 {
		dest.Expiry = v
	}

	if v := c.ID; v != "" {
		dest.ID = v
	}

	if v := c.Issuer; v != "" {
		dest.Issuer = v
	}

	if v := c.Subject; v != "" {
		dest.Subject = v
	}

	if v := c.Audience; len(v) > 0 {
		dest.Audience = v
	}
}

// MaxAge is a SignOption to set the expiration "exp", "iat" JWT standard claims.
// Can be passed as last input argument of the `Sign` function.
//
// If maxAge > second then sets expiration to the token.
// It's a helper field to set the `Expiry` and `IssuedAt`
// fields at once.
//
// See the `Clock` package-level variable to modify
// the current time function.
func MaxAge(maxAge time.Duration) SignOptionFunc {
	return func(c *Claims) {
		if maxAge <= time.Second {
			return
		}
		now := Clock()
		c.Expiry = now.Add(maxAge).Unix()
		c.IssuedAt = now.Unix()
	}
}

// MaxAgeMap is a helper to set "exp" and "iat" claims to a map claims.
// Usage:
// claims := map[string]interface{}{"foo": "bar"}
// MaxAgeMap(15 * time.Minute, claims)
// Sign(alg, key, claims)
func MaxAgeMap(maxAge time.Duration, claims Map) {
	if claims == nil {
		return
	}

	if maxAge <= time.Second {
		return
	}

	now := Clock()
	if claims["exp"] == nil {
		claims["exp"] = now.Add(maxAge).Unix()
		claims["iat"] = now.Unix()
	}
}

// Merge accepts two claim structs or maps
// and returns a flattened JSON result of both (no checks for duplicatations are maden).
//
// Usage:
//
//  claims := Merge(map[string]interface{}{"foo":"bar"}, Claims{
//    MaxAge: 15 * time.Minute,
//    Issuer: "an-issuer",
//  })
//  Sign(alg, key, claims)
//
// Merge is automatically called when:
//
//  Sign(alg, key, claims, MaxAge(time.Duration))
//  Sign(alg, key, claims, Claims{...})
func Merge(claims interface{}, other interface{}) []byte {
	claimsB, err := Marshal(claims)
	if err != nil {
		return nil
	}

	otherB, err := Marshal(other)
	if err != nil {
		return nil
	}

	if len(otherB) == 0 {
		return claimsB
	}

	claimsB = claimsB[0 : len(claimsB)-1] // remove last '}'
	otherB = otherB[1:]                   // remove first '{'

	raw := append(claimsB, ',')
	raw = append(raw, otherB...)
	return raw
}
