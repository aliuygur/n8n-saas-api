package subscription

import (
	"context"

	polargo "github.com/polarsource/polar-go"
	"github.com/polarsource/polar-go/models/components"
)

type CreateCheckoutURLRequest struct {
	Products          []string
	CustomerID        string
	CustomerEmail     string
	CustomerIPAddress string
	SuccessURL        string
	CancelURL         string
	Seats             int64
}

type CreateCheckoutURLResponse struct {
	CheckoutID  string
	CheckoutURL string
}

func (s *Service) CreateCheckoutURL(ctx context.Context, req CreateCheckoutURLRequest) (*CreateCheckoutURLResponse, error) {

	checkoutParams := components.CheckoutCreate{
		Products:           req.Products,
		ExternalCustomerID: polargo.Pointer(req.CustomerID),
		CustomerEmail:      polargo.Pointer(req.CustomerEmail),
		CustomerIPAddress:  polargo.Pointer(req.CustomerIPAddress),
		SuccessURL:         polargo.Pointer(req.SuccessURL),
		ReturnURL:          polargo.Pointer(req.CancelURL),
		Seats:              polargo.Pointer(req.Seats),
	}

	res, err := s.polarClient.Checkouts.Create(ctx, checkoutParams)
	if err != nil {
		return nil, err
	}

	return &CreateCheckoutURLResponse{
		CheckoutID:  res.Checkout.ID,
		CheckoutURL: res.Checkout.URL,
	}, nil

}
