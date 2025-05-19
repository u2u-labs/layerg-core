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
	"github.com/golang-jwt/jwt/v5"
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
	IssuedAt  int64             `json:"iat,omitempty"`
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

func (s *SessionTokenClaims) GetExpirationTime() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(time.Unix(s.ExpiresAt, 0)), nil
}
func (s *SessionTokenClaims) GetNotBefore() (*jwt.NumericDate, error) {
	return nil, nil
}
func (s *SessionTokenClaims) GetIssuedAt() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(time.Unix(s.IssuedAt, 0)), nil
}
func (s *SessionTokenClaims) GetAudience() (jwt.ClaimStrings, error) {
	return []string{}, nil
}
func (s *SessionTokenClaims) GetIssuer() (string, error) {
	return "", nil
}
func (s *SessionTokenClaims) GetSubject() (string, error) {
	return "", nil
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

const UALoginMessage = "logmein"

func (s *ApiServer) AuthenticateEvm(ctx context.Context, in *api.AuthenticateEvmRequest) (*api.Session, error) {
	// Before hook.
	if fn := s.runtime.BeforeAuthenticateEvm(); fn != nil {
		beforeFn := func(clientIP, clientPort string) error {
			result, err, code := fn(ctx, s.logger, "", "", nil, 0, clientIP, clientPort, in)
			if err != nil {
				return status.Error(code, err.Error())
			}
			if result == nil {
				s.logger.Warn("Intercepted a disabled resource.", zap.Any("resource", ctx.Value(ctxFullMethodKey{}).(string)))
				return status.Error(codes.NotFound, "Requested resource was not found.")
			}
			in = result
			return nil
		}
		err := traceApiBefore(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), beforeFn)
		if err != nil {
			return nil, err
		}
	}

	if in.Account == nil || in.Account.EvmAddress == "" || in.Account.EvmSignature == "" {
		return nil, status.Error(codes.InvalidArgument, "Metamask address and signature are required.")
	}

	// Validate the signature
	valid, err := verifySignature(UALoginMessage, in.Account.EvmAddress, in.Account.EvmSignature)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid signature.")
	}
	if !valid {
		return nil, status.Error(codes.Unauthenticated, "Signature does not match the address.")
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

	dbUserID, dbUsername, uaTokens, created, err := AuthenticateEvm(ctx, s.logger, s.db, in.Account.EvmAddress, in.Account.EvmSignature, username, create, s.config)
	if err != nil {
		return nil, err
	}

	if s.config.GetSession().SingleSession {
		s.sessionCache.RemoveAll(uuid.Must(uuid.FromString(dbUserID)))
	}

	tokenIssuedAt := time.Now().Unix()
	tokenID := uuid.Must(uuid.NewV4()).String()
	token, exp := generateToken(s.config, tokenID, tokenIssuedAt, dbUserID, dbUsername, in.Account.Vars)
	refreshToken, refreshExp := generateRefreshToken(s.config, tokenID, tokenIssuedAt, dbUserID, dbUsername, in.Account.Vars)
	s.sessionCache.Add(uuid.FromStringOrNil(dbUserID), exp, tokenID, refreshExp, tokenID)
	s.tokenPairCache.Add(dbUserID, uaTokens.Data.AccessToken, uaTokens.Data.RefreshToken, uaTokens.Data.AccessTokenExpire, uaTokens.Data.RefreshTokenExpire)
	session := &api.Session{Created: created, Token: token, RefreshToken: refreshToken}

	// After hook.
	if fn := s.runtime.AfterAuthenticateEvm(); fn != nil {
		afterFn := func(clientIP, clientPort string) error {
			return fn(ctx, s.logger, dbUserID, dbUsername, in.Account.Vars, exp, clientIP, clientPort, session, in)
		}
		traceApiAfter(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), afterFn)
	}
	return session, nil
}

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

func (s *ApiServer) AuthenticateCustom(ctx context.Context, in *api.AuthenticateCustomRequest) (*api.Session, error) {
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

	tokenIssuedAt := time.Now().Unix()
	tokenID := uuid.Must(uuid.NewV4()).String()
	token, exp := generateToken(s.config, tokenID, tokenIssuedAt, dbUserID, dbUsername, in.Account.Vars)
	refreshToken, refreshExp := generateRefreshToken(s.config, tokenID, tokenIssuedAt, dbUserID, dbUsername, in.Account.Vars)
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

	tokenIssuedAt := time.Now().Unix()
	tokenID := uuid.Must(uuid.NewV4()).String()
	token, exp := generateToken(s.config, tokenID, tokenIssuedAt, dbUserID, dbUsername, in.Account.Vars)
	refreshToken, refreshExp := generateRefreshToken(s.config, tokenID, tokenIssuedAt, dbUserID, dbUsername, in.Account.Vars)
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

	email := in.Account.Email
	if email == "" {
		return nil, status.Error(codes.InvalidArgument, "Email address is required.")
	}

	otp := in.Otp
	if otp == "" {
		return nil, status.Error(codes.InvalidArgument, "OTP is required.")
	}

	var dbUserID string
	var created bool
	var err error

	create := in.Create == nil || in.Create.Value

	dbUserID, dbUsername, uaAccessToken, uaRefreshToken, uaAccessExp, uaRefreshExp, created, err := AuthenticateEmail(ctx, s.logger, s.db, s.config, email, otp, create)
	if err != nil {
		return nil, err
	}

	if s.config.GetSession().SingleSession {
		s.sessionCache.RemoveAll(uuid.Must(uuid.FromString(dbUserID)))
	}

	tokenIssuedAt := time.Now().Unix()
	tokenID := uuid.Must(uuid.NewV4()).String()
	token, exp := generateToken(s.config, tokenID, tokenIssuedAt, dbUserID, dbUsername, in.Account.Vars)
	refreshToken, refreshExp := generateRefreshToken(s.config, tokenID, tokenIssuedAt, dbUserID, dbUsername, in.Account.Vars)

	s.sessionCache.Add(uuid.FromStringOrNil(dbUserID), exp, tokenID, refreshExp, tokenID)
	s.activeTokenCacheUser.Add(uuid.FromStringOrNil(dbUserID), exp, token, refreshExp, refreshToken, 0, "", 0, "")
	s.tokenPairCache.Add(dbUserID, uaAccessToken, uaRefreshToken, uaAccessExp, uaRefreshExp)
	session := &api.Session{Created: created, Token: token, RefreshToken: refreshToken}

	// After hook.
	if fn := s.runtime.AfterAuthenticateEmail(); fn != nil {
		afterFn := func(clientIP, clientPort string) error {
			return fn(ctx, s.logger, dbUserID, dbUsername, in.Account.Vars, exp, clientIP, clientPort, session, in)
		}

		// Execute the after function lambda wrapped in a trace for stats measurement.
		traceApiAfter(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), afterFn)
	}
	return session, nil
}

func generateToken(config Config, tokenID string, tokenIssuedAt int64, userID, username string, vars map[string]string) (string, int64) {
	exp := time.Now().UTC().Add(time.Duration(config.GetSession().TokenExpirySec) * time.Second).Unix()
	return generateTokenWithExpiry(config.GetSession().EncryptionKey, tokenID, tokenIssuedAt, userID, username, vars, exp)
}

func generateRefreshToken(config Config, tokenID string, tokenIssuedAt int64, userID string, username string, vars map[string]string) (string, int64) {
	exp := time.Now().UTC().Add(time.Duration(config.GetSession().RefreshTokenExpirySec) * time.Second).Unix()
	return generateTokenWithExpiry(config.GetSession().RefreshEncryptionKey, tokenID, tokenIssuedAt, userID, username, vars, exp)
}

func generateTokenWithExpiry(signingKey, tokenID string, tokenIssuedAt int64, userID, username string, vars map[string]string, exp int64) (string, int64) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &SessionTokenClaims{
		TokenId:   tokenID,
		UserId:    userID,
		Username:  username,
		Vars:      vars,
		ExpiresAt: exp,
		IssuedAt:  tokenIssuedAt,
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

func (s *ApiServer) SendEmailAuthOTP(ctx context.Context, in *api.SendEmailOTPRequest) (*emptypb.Empty, error) {
	if in.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "Email is required.")
	}
	err := EmailUASendOTP(ctx, in.Email, s.config)
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

	tokenIssuedAt := time.Now().Unix()
	tokenID := uuid.Must(uuid.NewV4()).String()
	token, exp := generateToken(s.config, tokenID, tokenIssuedAt, dbUserID, username, in.Account.Vars)
	refreshToken, refreshExp := generateRefreshToken(s.config, tokenID, tokenIssuedAt, dbUserID, username, in.Account.Vars)

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

func (s *ApiServer) AuthenticateUA(ctx context.Context, in *api.UASocialLoginRequest) (*api.Session, error) {
	// Before hook.
	if fn := s.runtime.BeforeAuthenticateUA(); fn != nil {
		beforeFn := func(clientIP, clientPort string) error {
			result, err, code := fn(ctx, s.logger, "", "", nil, 0, clientIP, clientPort, in)
			if err != nil {
				return status.Error(code, err.Error())
			}
			if result == nil {
				s.logger.Warn("Intercepted a disabled resource.", zap.Any("resource", ctx.Value(ctxFullMethodKey{}).(string)))
				return status.Error(codes.NotFound, "Requested resource was not found.")
			}
			in = result
			return nil
		}

		err := traceApiBefore(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), beforeFn)
		if err != nil {
			return nil, err
		}
	}

	// Call UA service to get login data
	uaLoginData, err := SocialLoginUA(ctx, "", in, s.config)
	if err != nil {
		return nil, status.Error(codes.Internal, "Error authenticating with UA service")
	}

	dbUserID, dbUsername, created, err := AuthenticateUA(ctx, s.logger, s.db, uaLoginData.Data)
	if err != nil {
		return nil, err
	}

	if s.config.GetSession().SingleSession {
		s.sessionCache.RemoveAll(uuid.Must(uuid.FromString(dbUserID)))
	}

	// Generate native tokens
	tokenIssuedAt := time.Now().Unix()
	tokenID := uuid.Must(uuid.NewV4()).String()
	token, exp := generateToken(s.config, tokenID, tokenIssuedAt, dbUserID, dbUsername, nil)
	refreshToken, refreshExp := generateRefreshToken(s.config, tokenID, tokenIssuedAt, dbUserID, dbUsername, nil)

	// Store both native and UA tokens in the caches
	s.sessionCache.Add(uuid.FromStringOrNil(dbUserID), exp, tokenID, refreshExp, tokenID)
	s.tokenPairCache.Add(dbUserID, uaLoginData.Data.AccessToken, uaLoginData.Data.RefreshToken,
		uaLoginData.Data.AccessTokenExpire, uaLoginData.Data.RefreshTokenExpire)

	session := &api.Session{Created: created, Token: token, RefreshToken: refreshToken}

	// After hook.
	if fn := s.runtime.AfterAuthenticateUA(); fn != nil {
		afterFn := func(clientIP, clientPort string) error {
			return fn(ctx, s.logger, dbUserID, dbUsername, nil, exp, clientIP, clientPort, session, in)
		}
		traceApiAfter(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), afterFn)
	}

	return session, nil
}
