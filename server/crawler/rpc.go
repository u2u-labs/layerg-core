package crawler

import (
	"github.com/u2u-labs/layerg-core/server/crawler/utils/models"
	"github.com/unicornultrafoundation/go-u2u/ethclient"
)

func initChainClient(chain *models.Chain) (*ethclient.Client, error) {
	return ethclient.Dial(chain.RpcUrl)
}
