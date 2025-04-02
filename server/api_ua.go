package server

// func (s *ApiServer) SendUAOnchainTX(ctx context.Context, in *api.SendOnchainTransactionRequest) (*emptypb.Empty, error) {
// 	md, _ := metadata.FromIncomingContext(ctx)
// 	auth, _ := md["authorization"]

// 	uaToken, _ := s.tokenPairCache.Get(auth[0])
// 	request := &runtime.TransactionRequest{
// 		To:                   in.TransactionReq.To,
// 		Value:                in.TransactionReq.Value,
// 		Data:                 in.TransactionReq.Data,
// 		MaxPriorityFeePerGas: in.TransactionReq.MaxPriorityFeePerGas,
// 	}
// 	payload := &runtime.UATransactionRequest{
// 		ProjectID:      s.config.GetLayerGCoreConfig().UAPorjectID,
// 		ChainID:        int(in.ChainId),
// 		Sponsor:        in.Sponsor,
// 		TransactionReq: request,
// 	}
// 	err := SendUAOnchainTX(ctx, uaToken.AccessToken, *payload, s.config)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &emptypb.Empty{}, nil
// }
