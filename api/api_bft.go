package api

type PublicSubchainAPI struct {
	s Backend
}

func NewPublicSubchainAPI(s Backend) *PublicSubchainAPI {
	return &PublicSubchainAPI{s}
}

// func (api *PublicSubchainAPI) GetVerSet() []bft.Verifier {
// 	return api.s.List()
// }

// func (api *PublicSubchainAPI) GetProposer() bft.Verifier {
// 	return api.s.GetProposer()
// }
