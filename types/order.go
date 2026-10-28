package types

type OrderResult struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
	Amount  int    `json:"amount"`
}

type OrderParams struct {
	OrderID string `json:"order_id" jsonschema:"Order ID to query"`
}
