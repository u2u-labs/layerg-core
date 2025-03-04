package server

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gofrs/uuid/v5"
	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/u2u-labs/go-layerg-common/api"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	invalidUsernameRegex = regexp.MustCompilePOSIX("([[:cntrl:]]|[[\t\n\r\f\v]])+")
	invalidCharsRegex    = regexp.MustCompilePOSIX("([[:cntrl:]]|[[:space:]])+")
	emailRegex           = regexp.MustCompile(`^.+@.+\..+$`)
)

type SessionTokenClaims struct {
	TokenId   string            `json:"tid,omitempty"`
	UserId    string            `json:"uid,omitempty"`
	Username  string            `json:"usn,omitempty"`
	Vars      map[string]string `json:"vrs,omitempty"`
	ExpiresAt int64             `json:"exp,omitempty"`
}

func (stc *SessionTokenClaims) Valid() error {
	// Verify expiry.
	if stc.ExpiresAt <= time.Now().UTC().Unix() {
		vErr := new(jwt.ValidationError)
		vErr.Inner = errors.New("Token is expired")
		vErr.Errors |= jwt.ValidationErrorExpired
		return vErr
	}
	return nil
}

type TokenPairCache struct {
	nativeToUA map[string]UATokenPair
	mu         sync.RWMutex
}

type UATokenPair struct {
	AccessToken  string
	RefreshToken string
	AccessExp    int64
	RefreshExp   int64
}

func NewTokenPairCache() *TokenPairCache {
	return &TokenPairCache{
		nativeToUA: make(map[string]UATokenPair),
	}
}

func (c *TokenPairCache) Add(nativeToken string, uaAccess, uaRefresh string, accessExp, refreshExp int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nativeToUA[nativeToken] = UATokenPair{
		AccessToken:  uaAccess,
		RefreshToken: uaRefresh,
		AccessExp:    accessExp,
		RefreshExp:   refreshExp,
	}
}

func (c *TokenPairCache) Update(nativeToken string, uaAccess, uaRefresh string, accessExp, refreshExp int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.nativeToUA[nativeToken]; exists {
		c.nativeToUA[nativeToken] = UATokenPair{
			AccessToken:  uaAccess,
			RefreshToken: uaRefresh,
			AccessExp:    accessExp,
			RefreshExp:   refreshExp,
		}
	}
}

func (c *TokenPairCache) Get(nativeToken string) (UATokenPair, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	pair, exists := c.nativeToUA[nativeToken]
	return pair, exists
}

func (c *TokenPairCache) Remove(nativeToken string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.nativeToUA, nativeToken)
}

// func forwardToGlobalAuthenticator(ctx context.Context, url string, in interface{}) (*api.Session, error) {
// 	requestBody, err := json.Marshal(in)
// 	fmt.Printf("Forwarding request to global authenticator with body: %s\n", requestBody)
// 	if err != nil {
// 		return nil, err
// 	}

// 	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
// 	if err != nil {
// 		return nil, err
// 	}
// 	req.Header.Set("Content-Type", "application/json")

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		return nil, fmt.Errorf("failed to forward request, status code: %d", resp.StatusCode)
// 	}

// 	var session api.Session
// 	err = json.NewDecoder(resp.Body).Decode(&session)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &session, nil
// }

// func forwardToGlobalAuthenticator(ctx context.Context, address string, in interface{}) (*api.Session, error) {
// 	// TODO: for requesting asset, get token from global and request to asset
// 	conn, err := grpc.NewClient(address, grpc.WithInsecure())
// 	if err != nil {
// 		return nil, fmt.Errorf("did not connect: %v", err)
// 	}
// 	defer conn.Close()

// 	client := NewAuthenticatorServiceClient(conn)

// 	// Every game developer need to create an maintainer account on global credential and set here to perform action
// 	encodedServerKey := base64.StdEncoding.EncodeToString([]byte("baohaha:Abc12345:5d2aecb0-0df8-4d46-8a3a-823d4433d096:4b7607caae421b7edd763cb0e883f6a6"))

// 	md := metadata.New(map[string]string{
// 		// "grpc-authorization": encodedServerKey,
// 		"authorization": "Basic " + encodedServerKey,
// 	})

// 	ctx = metadata.NewOutgoingContext(ctx, md)

// 	switch v := in.(type) {
// 	case *api.AuthenticateEmailRequest:
// 		fmt.Printf("Forwarding request to global authenticator with body: %+v\n", v)
// 		session, err := client.AuthenticateEmail(ctx, v)
// 		if err != nil {
// 			return nil, status.Errorf(status.Code(err), "failed to forward request: %v", err)
// 		}
// 		return session, nil

// 	case *api.AuthenticateGoogleRequest:
// 		fmt.Printf("Forwarding request to global authenticator with body: %+v\n", v)
// 		session, err := client.AuthenticateGoogle(ctx, v)
// 		if err != nil {
// 			return nil, status.Errorf(status.Code(err), "failed to forward request: %v", err)
// 		}
// 		return session, nil
// 	case *api.AuthenticateAppleRequest:
// 		session, err := client.AuthenticateApple(ctx, v)
// 		if err != nil {
// 			return nil, status.Errorf(status.Code(err), "failed to forward request: %v", err)
// 		}
// 		return session, nil
// 	case *api.AuthenticateCustomRequest:
// 		session, err := client.AuthenticateCustom(ctx, v)
// 		if err != nil {
// 			return nil, status.Errorf(status.Code(err), "failed to forward request: %v", err)
// 		}
// 		return session, nil
// 	case *api.AuthenticateDeviceRequest:
// 		session, err := client.AuthenticateDevice(ctx, v)
// 		if err != nil {
// 			return nil, status.Errorf(status.Code(err), "failed to forward request: %v", err)
// 		}
// 		return session, nil
// 	case *api.AuthenticateFacebookRequest:
// 		session, err := client.AuthenticateFacebook(ctx, v)
// 		if err != nil {
// 			return nil, status.Errorf(status.Code(err), "failed to forward request: %v", err)
// 		}
// 		return session, nil
// 	case *api.AuthenticateFacebookInstantGameRequest:
// 		session, err := client.AuthenticateFacebookInstantGame(ctx, v)
// 		if err != nil {
// 			return nil, status.Errorf(status.Code(err), "failed to forward request: %v", err)
// 		}
// 		return session, nil
// 	case *api.AuthenticateGameCenterRequest:
// 		session, err := client.AuthenticateGameCenter(ctx, v)
// 		if err != nil {
// 			return nil, status.Errorf(status.Code(err), "failed to forward request: %v", err)
// 		}
// 		return session, nil
// 	case *api.AuthenticateSteamRequest:
// 		session, err := client.AuthenticateSteam(ctx, v)
// 		if err != nil {
// 			return nil, status.Errorf(status.Code(err), "failed to forward request: %v", err)
// 		}
// 		return session, nil
// 	case *api.AuthenticateEvmRequest:
// 		session, err := client.AuthenticateEvm(ctx, v)
// 		if err != nil {
// 			return nil, status.Errorf(status.Code(err), "failed to forward request: %v", err)
// 		}
// 		return session, nil
// 	case *api.AuthenticateTelegramRequest:
// 		session, err := client.AuthenticateTelegram(ctx, v)
// 		if err != nil {
// 			return nil, status.Errorf(status.Code(err), "failed to forward request: %v", err)
// 		}
// 		return session, nil

// 	default:
// 		return nil, fmt.Errorf("unsupported request type")
// 	}
// }

// func (s *ApiServer) AuthenticateEvm(ctx context.Context, in *api.AuthenticateEvmRequest) (*api.Session, error) {
// 	// Before hook.
// 	if fn := s.runtime.BeforeAuthenticateEvm(); fn != nil {
// 		beforeFn := func(clientIP, clientPort string) error {
// 			result, err, code := fn(ctx, s.logger, "", "", nil, 0, clientIP, clientPort, in)
// 			if err != nil {
// 				return status.Error(code, err.Error())
// 			}
// 			if result == nil {
// 				s.logger.Warn("Intercepted a disabled resource.", zap.Any("resource", ctx.Value(ctxFullMethodKey{}).(string)))
// 				return status.Error(codes.NotFound, "Requested resource was not found.")
// 			}
// 			in = result
// 			return nil
// 		}
// 		err := traceApiBefore(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), beforeFn)
// 		if err != nil {
// 			return nil, err
// 		}
// 	}

// 	if in.Account == nil || in.Account.EvmAddress == "" || in.Account.EvmSignature == "" {
// 		return nil, status.Error(codes.InvalidArgument, "Metamask address and signature are required.")
// 	}

// 	// Validate the signature
// 	valid, err := verifySignature("sample", in.Account.EvmAddress, in.Account.EvmSignature)
// 	if err != nil {
// 		return nil, status.Error(codes.InvalidArgument, "Invalid signature.")
// 	}
// 	if !valid {
// 		return nil, status.Error(codes.Unauthenticated, "Signature does not match the address.")
// 	}

// 	username := in.Account.Username
// 	if username == "" {
// 		username = generateUsername()
// 	} else if invalidUsernameRegex.MatchString(username) {
// 		return nil, status.Error(codes.InvalidArgument, "Username invalid, no spaces or control characters allowed.")
// 	} else if len(username) > 128 {
// 		return nil, status.Error(codes.InvalidArgument, "Username invalid, must be 1-128 bytes.")
// 	}

// 	create := in.Create == nil || in.Create.Value

// 	dbUserID, dbUsername, created, err := AuthenticateEvm(ctx, s.logger, s.db, in.Account.EvmAddress, in.Account.EvmAddress, username, create)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if s.config.GetSession().SingleSession {
// 		s.sessionCache.RemoveAll(uuid.Must(uuid.FromString(dbUserID)))
// 	}

// 	tokenID := uuid.Must(uuid.NewV4()).String()
// 	token, exp := generateToken(s.config, tokenID, dbUserID, dbUsername, in.Account.Vars)
// 	refreshToken, refreshExp := generateRefreshToken(s.config, tokenID, dbUserID, dbUsername, in.Account.Vars)
// 	s.sessionCache.Add(uuid.FromStringOrNil(dbUserID), exp, tokenID, refreshExp, tokenID)
// 	session := &api.Session{Created: created, Token: token, RefreshToken: refreshToken}

// 	// After hook.
// 	if fn := s.runtime.AfterAuthenticateEvm(); fn != nil {
// 		afterFn := func(clientIP, clientPort string) error {
// 			return fn(ctx, s.logger, dbUserID, dbUsername, in.Account.Vars, exp, clientIP, clientPort, session, in)
// 		}
// 		traceApiAfter(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), afterFn)
// 	}
// 	// global, err := forwardToGlobalAuthenticator(ctx, "localhost:8349", in)
// 	// }
// 	// if err != nil {
// 	// 	return nil, err
// 	// }
// 	// if in.GetAccount().AuthGlobal.Value == true {
// 	// 	return global, nil
// 	// }
// 	return session, nil
// }

func verifySignature(message, address, signature string) (bool, error) {
	// Ethereum specific prefix
	prefix := fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(message))
	prefixedMsg := []byte(prefix + message)
	msgHash := crypto.Keccak256Hash(prefixedMsg)

	// Remove the 0x prefix if present
	if strings.HasPrefix(signature, "0x") {
		signature = signature[2:]
	}

	// Decode the signature from hex
	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return false, errors.New("invalid hex signature")
	}

	if len(sigBytes) != 65 {
		return false, errors.New("invalid signature length")
	}

	// Adjust the last byte of the signature (the recovery id)
	sigBytes[64] -= 27

	// Recover the public key from the signature
	pubKey, err := crypto.SigToPub(msgHash.Bytes(), sigBytes)
	if err != nil {
		return false, err
	}

	// Convert the public key to an Ethereum address
	recoveredAddr := crypto.PubkeyToAddress(*pubKey).Hex()

	// Verify if the recovered address matches the provided address
	return strings.ToLower(recoveredAddr) == strings.ToLower(address), nil
}

func (s *ApiServer) AuthenticateApple(ctx context.Context, in *api.AuthenticateAppleRequest) (*api.Session, error) {
	// Before hook.
	if fn := s.runtime.BeforeAuthenticateApple(); fn != nil {
		beforeFn := func(clientIP, clientPort string) error {
			result, err, code := fn(ctx, s.logger, "", "", nil, 0, clientIP, clientPort, in)
			if err != nil {
				return status.Error(code, err.Error())
			}
			if result == nil {
				// If result is nil, requested resource is disabled.
				s.logger.Warn("Intercepted a disabled resource.", zap.Any("resource", ctx.Value(ctxFullMethodKey{}).(string)))
				return status.Error(codes.NotFound, "Requested resource was not found.")
			}
			in = result
			return nil
		}

		// Execute the before function lambda wrapped in a trace for stats measurement.
		err := traceApiBefore(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), beforeFn)
		if err != nil {
			return nil, err
		}
	}

	if s.config.GetSocial().Apple.BundleId == "" {
		return nil, status.Error(codes.FailedPrecondition, "Apple authentication is not configured.")
	}

	if in.Account == nil || in.Account.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "Apple ID token is required.")
	}

	username := in.Username
	if username == "" {
		username = generateUsername()
	} else if invalidUsernameRegex.MatchString(username) {
		return nil, status.Error(codes.InvalidArgument, "Username invalid, no spaces or control characters allowed.")
	} else if len(username) > 128 {
		return nil, status.Error(codes.InvalidArgument, "Username invalid, must be 1-128 bytes.")
	}

	create := in.Create == nil || in.Create.Value

	dbUserID, dbUsername, created, err := AuthenticateApple(ctx, s.logger, s.db, s.socialClient, s.config.GetSocial().Apple.BundleId, in.Account.Token, username, create)
	if err != nil {
		return nil, err
	}

	if s.config.GetSession().SingleSession {
		s.sessionCache.RemoveAll(uuid.Must(uuid.FromString(dbUserID)))
	}

	tokenID := uuid.Must(uuid.NewV4()).String()
	token, exp := generateToken(s.config, tokenID, dbUserID, dbUsername, in.Account.Vars)
	refreshToken, refreshExp := generateRefreshToken(s.config, tokenID, dbUserID, dbUsername, in.Account.Vars)
	s.sessionCache.Add(uuid.FromStringOrNil(dbUserID), exp, tokenID, refreshExp, tokenID)
	session := &api.Session{Created: created, Token: token, RefreshToken: refreshToken}

	// After hook.
	if fn := s.runtime.AfterAuthenticateApple(); fn != nil {
		afterFn := func(clientIP, clientPort string) error {
			return fn(ctx, s.logger, dbUserID, dbUsername, in.Account.Vars, exp, clientIP, clientPort, session, in)
		}

		// Execute the after function lambda wrapped in a trace for stats measurement.
		traceApiAfter(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), afterFn)
	}
	// global, err := forwardToGlobalAuthenticator(ctx, "localhost:8349", in)
	// }
	// if err != nil {
	// 	return nil, err
	// }
	// if in.AuthGlobal.Value == true {
	// 	return global, nil
	// }
	return session, nil
}

func (s *ApiServer) AuthenticateCustom(ctx context.Context, in *api.AuthenticateCustomRequest) (*api.Session, error) {
	// _, err := forwardToGlobalAuthenticator(ctx, "localhost:8349", in)
	// }
	// if err != nil {
	// 	return nil, err
	// }
	// Before hook.
	if fn := s.runtime.BeforeAuthenticateCustom(); fn != nil {
		beforeFn := func(clientIP, clientPort string) error {
			result, err, code := fn(ctx, s.logger, "", "", nil, 0, clientIP, clientPort, in)
			if err != nil {
				return status.Error(code, err.Error())
			}
			if result == nil {
				// If result is nil, requested resource is disabled.
				s.logger.Warn("Intercepted a disabled resource.", zap.Any("resource", ctx.Value(ctxFullMethodKey{}).(string)))
				return status.Error(codes.NotFound, "Requested resource was not found.")
			}
			in = result
			return nil
		}

		// Execute the before function lambda wrapped in a trace for stats measurement.
		err := traceApiBefore(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), beforeFn)
		if err != nil {
			return nil, err
		}
	}

	if in.Account == nil || in.Account.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "Custom ID is required.")
	} else if invalidCharsRegex.MatchString(in.Account.Id) {
		return nil, status.Error(codes.InvalidArgument, "Custom ID invalid, no spaces or control characters allowed.")
	} else if len(in.Account.Id) < 6 || len(in.Account.Id) > 128 {
		return nil, status.Error(codes.InvalidArgument, "Custom ID invalid, must be 6-128 bytes.")
	}

	username := in.Username
	if username == "" {
		username = generateUsername()
	} else if invalidUsernameRegex.MatchString(username) {
		return nil, status.Error(codes.InvalidArgument, "Username invalid, no spaces or control characters allowed.")
	} else if len(username) > 128 {
		return nil, status.Error(codes.InvalidArgument, "Username invalid, must be 1-128 bytes.")
	}

	create := in.Create == nil || in.Create.Value

	dbUserID, dbUsername, created, err := AuthenticateCustom(ctx, s.logger, s.db, in.Account.Id, username, create)
	if err != nil {
		return nil, err
	}

	if s.config.GetSession().SingleSession {
		s.sessionCache.RemoveAll(uuid.Must(uuid.FromString(dbUserID)))
	}

	tokenID := uuid.Must(uuid.NewV4()).String()
	token, exp := generateToken(s.config, tokenID, dbUserID, dbUsername, in.Account.Vars)
	refreshToken, refreshExp := generateRefreshToken(s.config, tokenID, dbUserID, dbUsername, in.Account.Vars)
	s.sessionCache.Add(uuid.FromStringOrNil(dbUserID), exp, tokenID, refreshExp, tokenID)
	session := &api.Session{Created: created, Token: token, RefreshToken: refreshToken}

	// After hook.
	if fn := s.runtime.AfterAuthenticateCustom(); fn != nil {
		afterFn := func(clientIP, clientPort string) error {
			return fn(ctx, s.logger, dbUserID, dbUsername, in.Account.Vars, exp, clientIP, clientPort, session, in)
		}

		// Execute the after function lambda wrapped in a trace for stats measurement.
		traceApiAfter(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), afterFn)
	}

	// global, err := forwardToGlobalAuthenticator(ctx, "localhost:8349", in)
	// }
	// if err != nil {
	// 	return nil, err
	// }
	// if in.AuthGlobal.Value == true {
	// 	return global, nil
	// }

	return session, nil
}

func (s *ApiServer) AuthenticateDevice(ctx context.Context, in *api.AuthenticateDeviceRequest) (*api.Session, error) {
	// Before hook.
	if fn := s.runtime.BeforeAuthenticateDevice(); fn != nil {
		beforeFn := func(clientIP, clientPort string) error {
			result, err, code := fn(ctx, s.logger, "", "", nil, 0, clientIP, clientPort, in)
			if err != nil {
				return status.Error(code, err.Error())
			}
			if result == nil {
				// If result is nil, requested resource is disabled.
				s.logger.Warn("Intercepted a disabled resource.", zap.Any("resource", ctx.Value(ctxFullMethodKey{}).(string)))
				return status.Error(codes.NotFound, "Requested resource was not found.")
			}
			in = result
			return nil
		}

		// Execute the before function lambda wrapped in a trace for stats measurement.
		err := traceApiBefore(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), beforeFn)
		if err != nil {
			return nil, err
		}
	}

	if in.Account == nil || in.Account.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "Device ID is required.")
	} else if invalidCharsRegex.MatchString(in.Account.Id) {
		return nil, status.Error(codes.InvalidArgument, "Device ID invalid, no spaces or control characters allowed.")
	} else if len(in.Account.Id) < 10 || len(in.Account.Id) > 128 {
		return nil, status.Error(codes.InvalidArgument, "Device ID invalid, must be 10-128 bytes.")
	}

	username := in.Username
	if username == "" {
		username = generateUsername()
	} else if invalidUsernameRegex.MatchString(username) {
		return nil, status.Error(codes.InvalidArgument, "Username invalid, no spaces or control characters allowed.")
	} else if len(username) > 128 {
		return nil, status.Error(codes.InvalidArgument, "Username invalid, must be 1-128 bytes.")
	}

	create := in.Create == nil || in.Create.Value

	dbUserID, dbUsername, created, err := AuthenticateDevice(ctx, s.logger, s.db, in.Account.Id, username, create)
	if err != nil {
		return nil, err
	}

	if s.config.GetSession().SingleSession {
		s.sessionCache.RemoveAll(uuid.Must(uuid.FromString(dbUserID)))
	}

	tokenID := uuid.Must(uuid.NewV4()).String()
	token, exp := generateToken(s.config, tokenID, dbUserID, dbUsername, in.Account.Vars)
	refreshToken, refreshExp := generateRefreshToken(s.config, tokenID, dbUserID, dbUsername, in.Account.Vars)
	s.sessionCache.Add(uuid.FromStringOrNil(dbUserID), exp, tokenID, refreshExp, tokenID)
	session := &api.Session{Created: created, Token: token, RefreshToken: refreshToken}

	// After hook.
	if fn := s.runtime.AfterAuthenticateDevice(); fn != nil {
		afterFn := func(clientIP, clientPort string) error {
			return fn(ctx, s.logger, dbUserID, dbUsername, in.Account.Vars, exp, clientIP, clientPort, session, in)
		}

		// Execute the after function lambda wrapped in a trace for stats measurement.
		traceApiAfter(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), afterFn)
	}
	// global, err := forwardToGlobalAuthenticator(ctx, "localhost:8349", in)
	// }
	// if err != nil {
	// 	return nil, err
	// }
	// if in.AuthGlobal.Value == true {
	// 	return global, nil
	// }
	return session, nil
}

func (s *ApiServer) AuthenticateEmail(ctx context.Context, in *api.AuthenticateEmailRequest) (*api.Session, error) {
	// Before hook.
	if fn := s.runtime.BeforeAuthenticateEmail(); fn != nil {
		beforeFn := func(clientIP, clientPort string) error {
			result, err, code := fn(ctx, s.logger, "", "", nil, 0, clientIP, clientPort, in)
			if err != nil {
				return status.Error(code, err.Error())
			}
			if result == nil {
				// If result is nil, requested resource is disabled.
				s.logger.Warn("Intercepted a disabled resource.", zap.Any("resource", ctx.Value(ctxFullMethodKey{}).(string)))
				return status.Error(codes.NotFound, "Requested resource was not found.")
			}
			in = result
			return nil
		}

		// Execute the before function lambda wrapped in a trace for stats measurement.
		err := traceApiBefore(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), beforeFn)
		if err != nil {
			return nil, err
		}
	}

	email := in.Account
	if email == nil {
		return nil, status.Error(codes.InvalidArgument, "Email address and password is required.")
	}

	var attemptUsernameLogin bool
	if email.Email == "" {
		// Password was supplied, but no email. Perhaps the user is attempting to login with username/password.
		attemptUsernameLogin = true
	} else if invalidCharsRegex.MatchString(email.Email) {
		return nil, status.Error(codes.InvalidArgument, "Invalid email address, no spaces or control characters allowed.")
	} else if !emailRegex.MatchString(email.Email) {
		return nil, status.Error(codes.InvalidArgument, "Invalid email address format.")
	} else if len(email.Email) < 10 || len(email.Email) > 255 {
		return nil, status.Error(codes.InvalidArgument, "Invalid email address, must be 10-255 bytes.")
	}

	if len(email.Password) < 8 {
		return nil, status.Error(codes.InvalidArgument, "Password must be at least 8 characters long.")
	}

	username := in.Username
	if username == "" {
		// If no username was supplied and the email was missing.
		if attemptUsernameLogin {
			return nil, status.Error(codes.InvalidArgument, "Username is required when email address is not supplied.")
		}

		// Email address was supplied, we are allowed to generate a username.
		username = generateUsername()
	} else if invalidUsernameRegex.MatchString(username) {
		return nil, status.Error(codes.InvalidArgument, "Username invalid, no spaces or control characters allowed.")
	} else if len(username) > 128 {
		return nil, status.Error(codes.InvalidArgument, "Username invalid, must be 1-128 bytes.")
	}

	var dbUserID string
	var created bool
	var err error

	if attemptUsernameLogin {
		// Attempting to log in with username/password. Create flag is ignored, creation is not possible here.
		dbUserID, err = AuthenticateUsername(ctx, s.logger, s.db, username, email.Password)
	} else {
		// Attempting email authentication, may or may not create.
		cleanEmail := strings.ToLower(email.Email)
		create := in.Create == nil || in.Create.Value

		dbUserID, username, created, err = AuthenticateEmail(ctx, s.logger, s.db, cleanEmail, email.Password, username, create)
	}
	if err != nil {
		return nil, err
	}

	if s.config.GetSession().SingleSession {
		s.sessionCache.RemoveAll(uuid.Must(uuid.FromString(dbUserID)))
	}

	tokenID := uuid.Must(uuid.NewV4()).String()
	token, exp := generateToken(s.config, tokenID, dbUserID, username, in.Account.Vars)
	refreshToken, refreshExp := generateRefreshToken(s.config, tokenID, dbUserID, username, in.Account.Vars)
	s.sessionCache.Add(uuid.FromStringOrNil(dbUserID), exp, tokenID, refreshExp, tokenID)
	s.activeTokenCacheUser.Add(uuid.FromStringOrNil(dbUserID), exp, token, refreshExp, refreshToken, 0, "", 0, "")
	session := &api.Session{Created: created, Token: token, RefreshToken: refreshToken}

	// After hook.
	if fn := s.runtime.AfterAuthenticateEmail(); fn != nil {
		afterFn := func(clientIP, clientPort string) error {
			return fn(ctx, s.logger, dbUserID, username, in.Account.Vars, exp, clientIP, clientPort, session, in)
		}

		// Execute the after function lambda wrapped in a trace for stats measurement.
		traceApiAfter(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), afterFn)
	}
	// global, err := forwardToGlobalAuthenticator(ctx, "localhost:8349", in)
	// // }
	// if err != nil {
	// 	return nil, err
	// }
	// s.activeTokenCacheUser.Add(uuid.FromStringOrNil(dbUserID), 0, "", 0, "", 60, global.Token, 3600, global.RefreshToken)

	// if in.AuthGlobal.Value == true {
	// 	return global, nil
	// }

	return session, nil
}

func (s *ApiServer) AuthenticateFacebook(ctx context.Context, in *api.AuthenticateFacebookRequest) (*api.Session, error) {
	// Before hook.
	if fn := s.runtime.BeforeAuthenticateFacebook(); fn != nil {
		beforeFn := func(clientIP, clientPort string) error {
			result, err, code := fn(ctx, s.logger, "", "", nil, 0, clientIP, clientPort, in)
			if err != nil {
				return status.Error(code, err.Error())
			}
			if result == nil {
				// If result is nil, requested resource is disabled.
				s.logger.Warn("Intercepted a disabled resource.", zap.Any("resource", ctx.Value(ctxFullMethodKey{}).(string)))
				return status.Error(codes.NotFound, "Requested resource was not found.")
			}
			in = result
			return nil
		}

		// Execute the before function lambda wrapped in a trace for stats measurement.
		err := traceApiBefore(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), beforeFn)
		if err != nil {
			return nil, err
		}
	}

	if in.Account == nil || in.Account.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "Facebook access token is required.")
	}

	username := in.Username
	if username == "" {
		username = generateUsername()
	} else if invalidUsernameRegex.MatchString(username) {
		return nil, status.Error(codes.InvalidArgument, "Username invalid, no spaces or control characters allowed.")
	} else if len(username) > 128 {
		return nil, status.Error(codes.InvalidArgument, "Username invalid, must be 1-128 bytes.")
	}

	create := in.Create == nil || in.Create.Value

	dbUserID, dbUsername, created, err := AuthenticateFacebook(ctx, s.logger, s.db, s.socialClient, s.config.GetSocial().FacebookLimitedLogin.AppId, in.Account.Token, username, create)
	if err != nil {
		return nil, err
	}

	// Import friends if requested.
	if in.Sync != nil && in.Sync.Value {
		_ = importFacebookFriends(ctx, s.logger, s.db, s.tracker, s.router, s.socialClient, uuid.FromStringOrNil(dbUserID), dbUsername, in.Account.Token, false)
	}

	if s.config.GetSession().SingleSession {
		s.sessionCache.RemoveAll(uuid.Must(uuid.FromString(dbUserID)))
	}

	tokenID := uuid.Must(uuid.NewV4()).String()
	token, exp := generateToken(s.config, tokenID, dbUserID, dbUsername, in.Account.Vars)
	refreshToken, refreshExp := generateRefreshToken(s.config, tokenID, dbUserID, dbUsername, in.Account.Vars)
	s.sessionCache.Add(uuid.FromStringOrNil(dbUserID), exp, tokenID, refreshExp, tokenID)
	session := &api.Session{Created: created, Token: token, RefreshToken: refreshToken}

	// After hook.
	if fn := s.runtime.AfterAuthenticateFacebook(); fn != nil {
		afterFn := func(clientIP, clientPort string) error {
			return fn(ctx, s.logger, dbUserID, dbUsername, in.Account.Vars, exp, clientIP, clientPort, session, in)
		}

		// Execute the after function lambda wrapped in a trace for stats measurement.
		traceApiAfter(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), afterFn)
	}

	// global, err := forwardToGlobalAuthenticator(ctx, "localhost:8349", in)
	// }
	// if err != nil {
	// 	return nil, err
	// }
	// if in.AuthGlobal.Value == true {
	// 	return global, nil
	// }

	return session, nil
}

func (s *ApiServer) AuthenticateFacebookInstantGame(ctx context.Context, in *api.AuthenticateFacebookInstantGameRequest) (*api.Session, error) {
	// Before hook.
	if fn := s.runtime.BeforeAuthenticateFacebookInstantGame(); fn != nil {
		beforeFn := func(clientIP, clientPort string) error {
			result, err, code := fn(ctx, s.logger, "", "", nil, 0, clientIP, clientPort, in)
			if err != nil {
				return status.Error(code, err.Error())
			}
			if result == nil {
				// If result is nil, requested resource is disabled.
				s.logger.Warn("Intercepted a disabled resource.", zap.Any("resource", ctx.Value(ctxFullMethodKey{}).(string)))
				return status.Error(codes.NotFound, "Requested resource was not found.")
			}
			in = result
			return nil
		}

		// Execute the before function lambda wrapped in a trace for stats measurement.
		err := traceApiBefore(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), beforeFn)
		if err != nil {
			return nil, err
		}
	}

	if in.Account == nil || in.Account.SignedPlayerInfo == "" {
		return nil, status.Error(codes.InvalidArgument, "Signed Player Info for a Facebook Instant Game is required.")
	}

	username := in.Username
	if username == "" {
		username = generateUsername()
	} else if invalidUsernameRegex.MatchString(username) {
		return nil, status.Error(codes.InvalidArgument, "Username invalid, no spaces or control characters allowed.")
	} else if len(username) > 128 {
		return nil, status.Error(codes.InvalidArgument, "Username invalid, must be 1-128 bytes.")
	}

	create := in.Create == nil || in.Create.Value

	dbUserID, dbUsername, created, err := AuthenticateFacebookInstantGame(ctx, s.logger, s.db, s.socialClient, s.config.GetSocial().FacebookInstantGame.AppSecret, in.Account.SignedPlayerInfo, username, create)
	if err != nil {
		return nil, err
	}

	if s.config.GetSession().SingleSession {
		s.sessionCache.RemoveAll(uuid.Must(uuid.FromString(dbUserID)))
	}

	tokenID := uuid.Must(uuid.NewV4()).String()
	token, exp := generateToken(s.config, tokenID, dbUserID, dbUsername, in.Account.Vars)
	refreshToken, refreshExp := generateRefreshToken(s.config, tokenID, dbUserID, dbUsername, in.Account.Vars)
	s.sessionCache.Add(uuid.FromStringOrNil(dbUserID), exp, tokenID, refreshExp, tokenID)
	session := &api.Session{Created: created, Token: token, RefreshToken: refreshToken}

	// After hook.
	if fn := s.runtime.AfterAuthenticateFacebookInstantGame(); fn != nil {
		afterFn := func(clientIP, clientPort string) error {
			return fn(ctx, s.logger, dbUserID, dbUsername, in.Account.Vars, exp, clientIP, clientPort, session, in)
		}

		// Execute the after function lambda wrapped in a trace for stats measurement.
		traceApiAfter(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), afterFn)
	}
	// global, err := forwardToGlobalAuthenticator(ctx, "localhost:8349", in)
	// }
	// if err != nil {
	// 	return nil, err
	// }
	// if in.AuthGlobal.Value == true {
	// 	return global, nil
	// }

	return session, nil
}

func (s *ApiServer) AuthenticateGameCenter(ctx context.Context, in *api.AuthenticateGameCenterRequest) (*api.Session, error) {
	// Before hook.
	if fn := s.runtime.BeforeAuthenticateGameCenter(); fn != nil {
		beforeFn := func(clientIP, clientPort string) error {
			result, err, code := fn(ctx, s.logger, "", "", nil, 0, clientIP, clientPort, in)
			if err != nil {
				return status.Error(code, err.Error())
			}
			if result == nil {
				// If result is nil, requested resource is disabled.
				s.logger.Warn("Intercepted a disabled resource.", zap.Any("resource", ctx.Value(ctxFullMethodKey{}).(string)))
				return status.Error(codes.NotFound, "Requested resource was not found.")
			}
			in = result
			return nil
		}

		// Execute the before function lambda wrapped in a trace for stats measurement.
		err := traceApiBefore(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), beforeFn)
		if err != nil {
			return nil, err
		}
	}

	if in.Account == nil {
		return nil, status.Error(codes.InvalidArgument, "GameCenter access credentials are required.")
	} else if in.Account.BundleId == "" {
		return nil, status.Error(codes.InvalidArgument, "GameCenter bundle ID is required.")
	} else if in.Account.PlayerId == "" {
		return nil, status.Error(codes.InvalidArgument, "GameCenter player ID is required.")
	} else if in.Account.PublicKeyUrl == "" {
		return nil, status.Error(codes.InvalidArgument, "GameCenter public key URL is required.")
	} else if in.Account.Salt == "" {
		return nil, status.Error(codes.InvalidArgument, "GameCenter salt is required.")
	} else if in.Account.Signature == "" {
		return nil, status.Error(codes.InvalidArgument, "GameCenter signature is required.")
	} else if in.Account.TimestampSeconds == 0 {
		return nil, status.Error(codes.InvalidArgument, "GameCenter timestamp is required.")
	}

	username := in.Username
	if username == "" {
		username = generateUsername()
	} else if invalidUsernameRegex.MatchString(username) {
		return nil, status.Error(codes.InvalidArgument, "Username invalid, no spaces or control characters allowed.")
	} else if len(username) > 128 {
		return nil, status.Error(codes.InvalidArgument, "Username invalid, must be 1-128 bytes.")
	}

	create := in.Create == nil || in.Create.Value

	dbUserID, dbUsername, created, err := AuthenticateGameCenter(ctx, s.logger, s.db, s.socialClient, in.Account.PlayerId, in.Account.BundleId, in.Account.TimestampSeconds, in.Account.Salt, in.Account.Signature, in.Account.PublicKeyUrl, username, create)
	if err != nil {
		return nil, err
	}

	if s.config.GetSession().SingleSession {
		s.sessionCache.RemoveAll(uuid.Must(uuid.FromString(dbUserID)))
	}

	tokenID := uuid.Must(uuid.NewV4()).String()
	token, exp := generateToken(s.config, tokenID, dbUserID, dbUsername, in.Account.Vars)
	refreshToken, refreshExp := generateRefreshToken(s.config, tokenID, dbUserID, dbUsername, in.Account.Vars)
	s.sessionCache.Add(uuid.FromStringOrNil(dbUserID), exp, tokenID, refreshExp, tokenID)
	session := &api.Session{Created: created, Token: token, RefreshToken: refreshToken}

	// After hook.
	if fn := s.runtime.AfterAuthenticateGameCenter(); fn != nil {
		afterFn := func(clientIP, clientPort string) error {
			return fn(ctx, s.logger, dbUserID, dbUsername, in.Account.Vars, exp, clientIP, clientPort, session, in)
		}

		// Execute the after function lambda wrapped in a trace for stats measurement.
		traceApiAfter(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), afterFn)
	}
	// global, err := forwardToGlobalAuthenticator(ctx, "localhost:8349", in)
	// }
	// if err != nil {
	// 	return nil, err
	// }
	// if in.AuthGlobal.Value == true {
	// 	return global, nil
	// }

	return session, nil
}

func (s *ApiServer) AuthenticateGoogle(ctx context.Context, in *api.AuthenticateGoogleRequest) (*api.Session, error) {
	// Before hook.
	if fn := s.runtime.BeforeAuthenticateGoogle(); fn != nil {
		beforeFn := func(clientIP, clientPort string) error {
			result, err, code := fn(ctx, s.logger, "", "", nil, 0, clientIP, clientPort, in)
			if err != nil {
				return status.Error(code, err.Error())
			}
			if result == nil {
				// If result is nil, requested resource is disabled.
				s.logger.Warn("Intercepted a disabled resource.", zap.Any("resource", ctx.Value(ctxFullMethodKey{}).(string)))
				return status.Error(codes.NotFound, "Requested resource was not found.")
			}
			in = result
			return nil
		}

		// Execute the before function lambda wrapped in a trace for stats measurement.
		err := traceApiBefore(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), beforeFn)
		if err != nil {
			return nil, err
		}
	}

	if in.Account == nil || (in.Account.Token == "" && in.Code == "") {
		return nil, status.Error(codes.InvalidArgument, "Google access token is required.")
	}

	username := in.Username
	if username == "" {
		username = generateUsername()
	} else if invalidUsernameRegex.MatchString(username) {
		return nil, status.Error(codes.InvalidArgument, "Username invalid, no spaces or control characters allowed.")
	} else if len(username) > 128 {
		return nil, status.Error(codes.InvalidArgument, "Username invalid, must be 1-128 bytes.")
	}

	create := in.Create == nil || in.Create.Value

	uaInfo := NewUALoginCallBackRequest(in.Code, in.Error, in.State)
	dbUserID, dbUsername, uaTokens, created, err := AuthenticateGoogle(ctx, s.logger, s.db, s.socialClient, s.config, in.Account.Token, username, create, uaInfo)
	if err != nil {
		return nil, err
	}

	if s.config.GetSession().SingleSession {
		s.sessionCache.RemoveAll(uuid.Must(uuid.FromString(dbUserID)))
	}

	// Store both native and UA tokens in the caches
	tokenID := uuid.Must(uuid.NewV4()).String()
	token, exp := generateToken(s.config, tokenID, dbUserID, dbUsername, in.Account.Vars)
	refreshToken, refreshExp := generateRefreshToken(s.config, tokenID, dbUserID, dbUsername, in.Account.Vars)
	s.sessionCache.Add(uuid.FromStringOrNil(dbUserID), exp, tokenID, refreshExp, tokenID)
	session := &api.Session{Created: created, Token: token, RefreshToken: refreshToken}
	s.tokenPairCache.Add(dbUserID, uaTokens.Data.AccessToken, uaTokens.Data.RefreshToken, uaTokens.Data.AccessTokenExpire, uaTokens.Data.RefreshTokenExpire)

	// After hook.
	if fn := s.runtime.AfterAuthenticateGoogle(); fn != nil {
		afterFn := func(clientIP, clientPort string) error {
			return fn(ctx, s.logger, dbUserID, dbUsername, in.Account.Vars, exp, clientIP, clientPort, session, in)
		}

		// Execute the after function lambda wrapped in a trace for stats measurement.
		traceApiAfter(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), afterFn)
	}
	// global, err := forwardToGlobalAuthenticator(ctx, "localhost:8349", in)
	// }
	// if err != nil {
	// 	return nil, err
	// }
	// if in.AuthGlobal.Value == true {
	// 	return global, nil
	// }
	return session, nil
}

func (s *ApiServer) AuthenticateSteam(ctx context.Context, in *api.AuthenticateSteamRequest) (*api.Session, error) {
	// Before hook.
	if fn := s.runtime.BeforeAuthenticateSteam(); fn != nil {
		beforeFn := func(clientIP, clientPort string) error {
			result, err, code := fn(ctx, s.logger, "", "", nil, 0, clientIP, clientPort, in)
			if err != nil {
				return status.Error(code, err.Error())
			}
			if result == nil {
				// If result is nil, requested resource is disabled.
				s.logger.Warn("Intercepted a disabled resource.", zap.Any("resource", ctx.Value(ctxFullMethodKey{}).(string)))
				return status.Error(codes.NotFound, "Requested resource was not found.")
			}
			in = result
			return nil
		}

		// Execute the before function lambda wrapped in a trace for stats measurement.
		err := traceApiBefore(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), beforeFn)
		if err != nil {
			return nil, err
		}
	}

	if s.config.GetSocial().Steam.PublisherKey == "" || s.config.GetSocial().Steam.AppID == 0 {
		return nil, status.Error(codes.FailedPrecondition, "Steam authentication is not configured.")
	}

	if in.Account == nil || in.Account.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "Steam access token is required.")
	}

	username := in.Username
	if username == "" {
		username = generateUsername()
	} else if invalidUsernameRegex.MatchString(username) {
		return nil, status.Error(codes.InvalidArgument, "Username invalid, no spaces or control characters allowed.")
	} else if len(username) > 128 {
		return nil, status.Error(codes.InvalidArgument, "Username invalid, must be 1-128 bytes.")
	}

	create := in.Create == nil || in.Create.Value

	dbUserID, dbUsername, steamID, created, err := AuthenticateSteam(ctx, s.logger, s.db, s.socialClient, s.config.GetSocial().Steam.AppID, s.config.GetSocial().Steam.PublisherKey, in.Account.Token, username, create)
	if err != nil {
		return nil, err
	}

	// Import friends if requested.
	if in.Sync != nil && in.Sync.Value {
		_ = importSteamFriends(ctx, s.logger, s.db, s.tracker, s.router, s.socialClient, uuid.FromStringOrNil(dbUserID), dbUsername, s.config.GetSocial().Steam.PublisherKey, steamID, false)
	}

	if s.config.GetSession().SingleSession {
		s.sessionCache.RemoveAll(uuid.Must(uuid.FromString(dbUserID)))
	}

	tokenID := uuid.Must(uuid.NewV4()).String()
	token, exp := generateToken(s.config, tokenID, dbUserID, dbUsername, in.Account.Vars)
	refreshToken, refreshExp := generateRefreshToken(s.config, tokenID, dbUserID, dbUsername, in.Account.Vars)
	s.sessionCache.Add(uuid.FromStringOrNil(dbUserID), exp, tokenID, refreshExp, tokenID)
	session := &api.Session{Created: created, Token: token, RefreshToken: refreshToken}

	// After hook.
	if fn := s.runtime.AfterAuthenticateSteam(); fn != nil {
		afterFn := func(clientIP, clientPort string) error {
			return fn(ctx, s.logger, dbUserID, dbUsername, in.Account.Vars, exp, clientIP, clientPort, session, in)
		}

		// Execute the after function lambda wrapped in a trace for stats measurement.
		traceApiAfter(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), afterFn)
	}
	// global, err := forwardToGlobalAuthenticator(ctx, "localhost:8349", in)
	// }
	// if err != nil {
	// 	return nil, err
	// }
	// if in.AuthGlobal.Value == true {
	// 	return global, nil
	// }
	return session, nil
}

func (s *ApiServer) AuthenticateTwitter(ctx context.Context, in *api.AuthenticateTwitterRequest) (*api.Session, error) {
	// Before hook.
	if fn := s.runtime.BeforeAuthenticateTwitter(); fn != nil {
		beforeFn := func(clientIP, clientPort string) error {
			result, err, code := fn(ctx, s.logger, "", "", nil, 0, clientIP, clientPort, in)
			if err != nil {
				return status.Error(code, err.Error())
			}
			if result == nil {
				// If result is nil, requested resource is disabled.
				s.logger.Warn("Intercepted a disabled resource.", zap.Any("resource", ctx.Value(ctxFullMethodKey{}).(string)))
				return status.Error(codes.NotFound, "Requested resource was not found.")
			}
			in = result
			return nil
		}

		// Execute the before function lambda wrapped in a trace for stats measurement.
		err := traceApiBefore(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), beforeFn)
		if err != nil {
			return nil, err
		}
	}

	if in.Code == "" {
		return nil, status.Error(codes.InvalidArgument, "Twitter UA code is required.")
	}

	username := in.Username
	if username == "" {
		username = generateUsername()
	} else if invalidUsernameRegex.MatchString(username) {
		return nil, status.Error(codes.InvalidArgument, "Username invalid, no spaces or control characters allowed.")
	} else if len(username) > 128 {
		return nil, status.Error(codes.InvalidArgument, "Username invalid, must be 1-128 bytes.")
	}

	create := in.Create == nil || in.Create.Value

	uaInfo := NewUALoginCallBackRequest(in.Code, in.Error, in.State)
	dbUserID, dbUsername, uaTokens, created, err := AuthenticateTwitter(ctx, s.logger, s.db, s.socialClient, s.config, username, create, uaInfo)
	if err != nil {
		return nil, err
	}

	if s.config.GetSession().SingleSession {
		s.sessionCache.RemoveAll(uuid.Must(uuid.FromString(dbUserID)))
	}

	// Store both native and UA tokens in the caches
	tokenID := uuid.Must(uuid.NewV4()).String()
	token, exp := generateToken(s.config, tokenID, dbUserID, dbUsername, in.Account.Vars)
	refreshToken, refreshExp := generateRefreshToken(s.config, tokenID, dbUserID, dbUsername, in.Account.Vars)
	s.sessionCache.Add(uuid.FromStringOrNil(dbUserID), exp, tokenID, refreshExp, tokenID)
	session := &api.Session{Created: created, Token: token, RefreshToken: refreshToken}
	s.tokenPairCache.Add(dbUserID, uaTokens.Data.AccessToken, uaTokens.Data.RefreshToken, uaTokens.Data.AccessTokenExpire, uaTokens.Data.RefreshTokenExpire)

	// After hook.
	if fn := s.runtime.AfterAuthenticateTwitter(); fn != nil {
		afterFn := func(clientIP, clientPort string) error {
			return fn(ctx, s.logger, dbUserID, dbUsername, nil, exp, clientIP, clientPort, session, in)
		}

		// Execute the after function lambda wrapped in a trace for stats measurement.
		traceApiAfter(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), afterFn)
	}
	// global, err := forwardToGlobalAuthenticator(ctx, "localhost:8349", in)
	// }
	// if err != nil {
	// 	return nil, err
	// }
	// if in.AuthGlobal.Value == true {
	// 	return global, nil
	// }
	return session, nil
}

func generateToken(config Config, tokenID, userID, username string, vars map[string]string) (string, int64) {
	exp := time.Now().UTC().Add(time.Duration(config.GetSession().TokenExpirySec) * time.Second).Unix()
	return generateTokenWithExpiry(config.GetSession().EncryptionKey, tokenID, userID, username, vars, exp)
}

func generateRefreshToken(config Config, tokenID, userID string, username string, vars map[string]string) (string, int64) {
	exp := time.Now().UTC().Add(time.Duration(config.GetSession().RefreshTokenExpirySec) * time.Second).Unix()
	return generateTokenWithExpiry(config.GetSession().RefreshEncryptionKey, tokenID, userID, username, vars, exp)
}

func generateTokenWithExpiry(signingKey, tokenID, userID, username string, vars map[string]string, exp int64) (string, int64) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &SessionTokenClaims{
		TokenId:   tokenID,
		UserId:    userID,
		Username:  username,
		Vars:      vars,
		ExpiresAt: exp,
	})
	signedToken, _ := token.SignedString([]byte(signingKey))
	return signedToken, exp
}

func generateUsername() string {
	const usernameAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 10)
	for i := range b {
		b[i] = usernameAlphabet[rand.Intn(len(usernameAlphabet))]
	}
	return string(b)
}

func (s *ApiServer) SendTelegramAuthOTP(ctx context.Context, in *api.SendTelegramOTPRequest) (*emptypb.Empty, error) {
	if in.TelegramId == "" {
		return nil, status.Error(codes.InvalidArgument, "Telegram ID is required.")
	}
	err := SendTelegramAuthOTP(ctx, s.logger, s.config, in.TelegramId)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *ApiServer) AuthenticateTelegram(ctx context.Context, in *api.AuthenticateTelegramRequest) (*api.Session, error) {
	// Before hook.
	if fn := s.runtime.BeforeAuthenticateTelegram(); fn != nil {
		beforeFn := func(clientIP, clientPort string) error {
			result, err, code := fn(ctx, s.logger, "", "", nil, 0, clientIP, clientPort, in)
			if err != nil {
				return status.Error(code, err.Error())
			}
			if result == nil {
				// If result is nil, requested resource is disabled.
				s.logger.Warn("Intercepted a disabled resource.", zap.Any("resource", ctx.Value(ctxFullMethodKey{}).(string)))
				return status.Error(codes.NotFound, "Requested resource was not found.")
			}
			in = result
			return nil
		}

		// Execute the before function lambda wrapped in a trace for stats measurement.
		err := traceApiBefore(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), beforeFn)
		if err != nil {
			return nil, err
		}
	}

	if in.Account.TelegramId == "" {
		return nil, status.Error(codes.InvalidArgument, "Telegram ID is required.")
	}

	if in.Account.Otp == "" {
		return nil, status.Error(codes.InvalidArgument, "OTP is required.")
	}

	username := in.Account.Username
	if username == "" {
		username = generateUsername()
	} else if invalidUsernameRegex.MatchString(username) {
		return nil, status.Error(codes.InvalidArgument, "Username invalid, no spaces or control characters allowed.")
	} else if len(username) > 128 {
		return nil, status.Error(codes.InvalidArgument, "Username invalid, must be 1-128 bytes.")
	}

	create := in.Create == nil || in.Create.Value

	dbUserID, dbUsername, uaAccessToken, uaRefreshToken, uaAccessExp, uaRefreshExp, created, err := AuthenticateTelegram(ctx, s.logger, s.db, s.config, in.Account.TelegramId, int(in.Account.ChainId), username, in.Account.Firstname, in.Account.Lastname, in.Account.AvatarUrl, in.Account.Otp, create)
	if err != nil {
		return nil, err
	}

	if s.config.GetSession().SingleSession {
		s.sessionCache.RemoveAll(uuid.Must(uuid.FromString(dbUserID)))
	}

	tokenID := uuid.Must(uuid.NewV4()).String()
	token, exp := generateToken(s.config, tokenID, dbUserID, username, in.Account.Vars)
	refreshToken, refreshExp := generateRefreshToken(s.config, tokenID, dbUserID, username, in.Account.Vars)

	// Store both native and UA tokens in the caches
	s.sessionCache.Add(uuid.FromStringOrNil(dbUserID), exp, tokenID, refreshExp, tokenID)
	s.tokenPairCache.Add(dbUserID, uaAccessToken, uaRefreshToken, uaAccessExp, uaRefreshExp)

	session := &api.Session{Created: created, Token: token, RefreshToken: refreshToken}

	// After hook.
	if fn := s.runtime.AfterAuthenticateTelegram(); fn != nil {
		afterFn := func(clientIP, clientPort string) error {
			return fn(ctx, s.logger, dbUserID, dbUsername, in.Account.Vars, exp, clientIP, clientPort, session, in)
		}

		// Execute the after function lambda wrapped in a trace for stats measurement.
		traceApiAfter(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), afterFn)
	}

	return session, nil
}
