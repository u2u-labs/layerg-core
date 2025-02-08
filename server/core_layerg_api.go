package server

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

type UAHeaderPayload struct {
	Signature string `json:"signature"`
	Timestamp int64  `json:"timestamp"`
	Domain    string `json:"domain"`
}

type LayerGAPIParams struct {
	APIKey    string
	SecretKey string
	Origin    string
}

func CreateSignature(timestamp int64, domain, publicKey, priKey string) (*UAHeaderPayload, error) {
	var secretKey *ecdsa.PrivateKey
	if priKey != "" {
		var err error
		secretKey, err = crypto.HexToECDSA(strings.TrimPrefix(priKey, "0x"))
		if err != nil {
			return nil, fmt.Errorf("invalid secret key: %w", err)
		}
	}
	parsedURL, err := url.Parse(domain)
	if err != nil {
		return nil, err
	}

	hostname := strings.Split(parsedURL.Host, ":")[0]

	message := fmt.Sprintf("%s:%d:%s", hostname, timestamp, publicKey)
	prefix := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), strings.ToLower(message))
	finalHash := crypto.Keccak256Hash([]byte(prefix))

	signature, err := crypto.Sign(finalHash.Bytes(), secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %w", err)
	}
	signature[crypto.RecoveryIDOffset] += 27

	return &UAHeaderPayload{
		Signature: hexutil.Encode(signature),
		Timestamp: timestamp,
		Domain:    domain,
	}, nil
}

func GetSignatureSigner(timestamp int64, domain, publicKey, signatureHex string) (string, error) {
	message := fmt.Sprintf("%s:%d:%s", domain, timestamp, publicKey)
	hash := sha256.Sum256([]byte(strings.ToLower(message)))

	signature, err := hex.DecodeString(signatureHex)
	if err != nil {
		return "", fmt.Errorf("invalid signature format: %w", err)
	}

	recoveredPubKey, err := crypto.SigToPub(hash[:], signature)
	if err != nil {
		return "", fmt.Errorf("failed to recover signer: %w", err)
	}

	signerAddress := crypto.PubkeyToAddress(*recoveredPubKey).Hex()
	return signerAddress, nil
}
