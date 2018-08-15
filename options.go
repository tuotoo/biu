package biu

import "time"

var globalOptions = struct {
	jwtTimeout        time.Duration
	jwtSecret         func(string) ([]byte, error)
	jwtRefreshTimeout time.Duration

	autoGenPathDoc bool
}{
	jwtTimeout: time.Minute * 5,
	jwtSecret: func(userID string) (secret []byte, err error) {
		return []byte("secret"), nil
	},
	jwtRefreshTimeout: time.Hour * 24 * 7,

	autoGenPathDoc: false,
}

// Setter is a setter for setting global options.
type Setter struct{}

// JWTTimeout sets timeout for JWT.
func JWTTimeout(timeout time.Duration) Setter {
	globalOptions.jwtTimeout = timeout
	return Setter{}
}

// JWTTimeout sets timeout for JWT.
func (Setter) JWTTimeout(timeout time.Duration) Setter {
	return JWTTimeout(timeout)
}

// JWTSecret sets secret for JWT.
func JWTSecret(f func(userID string) (secret []byte, err error)) Setter {
	globalOptions.jwtSecret = f
	return Setter{}
}

// JWTSecret sets secret for JWT.
func (Setter) JWTSecret(f func(userID string) (secret []byte, err error)) Setter {
	return JWTSecret(f)
}

// JWTRefreshTimeout sets refresh timeout for JWT.
func JWTRefreshTimeout(timeout time.Duration) Setter {
	globalOptions.jwtRefreshTimeout = timeout
	return Setter{}
}

// JWTRefreshTimeout sets refresh timeout for JWT.
func (Setter) JWTRefreshTimeout(timeout time.Duration) Setter {
	return JWTRefreshTimeout(timeout)
}

// EnableGenPathDoc enable auto generate path parameter documents in each route.
func EnableGenPathDoc() Setter {
	globalOptions.autoGenPathDoc = true
	return Setter{}
}

// EnableGenPathDoc enable auto generate path parameter documents in each route.
func (Setter) EnableGenPathDoc() Setter {
	return EnableGenPathDoc()
}
