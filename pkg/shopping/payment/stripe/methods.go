package stripe

import (
	"fmt"

	"github.com/GGP1/palo/pkg/model"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/paymentmethod"
)

// AttachMethod attaches a PaymentMethod object to a Customer.
func AttachMethod(customerID, methodID string) (*stripe.PaymentMethod, error) {
	params := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(customerID),
	}

	pm, err := paymentmethod.Attach(methodID, params)
	if err != nil {
		return nil, fmt.Errorf("payments: error attaching PaymentMethod: %v", err)
	}

	return pm, nil
}

// CreateMethod creates a new payment method.
func CreateMethod(card model.Card) (string, error) {
	params := &stripe.PaymentMethodParams{
		Card: &stripe.PaymentMethodCardParams{
			Number:   stripe.String(card.Number),
			ExpMonth: stripe.String(card.ExpMonth),
			ExpYear:  stripe.String(card.ExpYear),
			CVC:      stripe.String(card.CVC),
		},
		Type: stripe.String("card"),
	}

	pm, err := paymentmethod.New(params)
	if err != nil {
		return "", fmt.Errorf("payments: error creating PaymentMethod: %v", err)
	}

	return pm.ID, nil
}

// DetachMethod detaches a PaymentMethod object from a Customer.
func DetachMethod(methodID string) (*stripe.PaymentMethod, error) {
	pm, err := paymentmethod.Detach(methodID, nil)
	if err != nil {
		return nil, fmt.Errorf("payments: error detaching PaymentMethod: %v", err)
	}

	return pm, nil
}

// ListMethods returns a list of PaymentMethods for a given Customer.
func ListMethods(customerID string) []*stripe.PaymentMethod {
	var list []*stripe.PaymentMethod
	params := &stripe.PaymentMethodListParams{
		Customer: stripe.String(customerID),
		Type:     stripe.String("card"),
	}

	i := paymentmethod.List(params)
	for i.Next() {
		list = append(list, i.PaymentMethod())
	}

	return list
}

// RetrieveMethod retrieves a PaymentMethod object.
func RetrieveMethod(methodID string) (*stripe.PaymentMethod, error) {
	pm, err := paymentmethod.Get(methodID, nil)
	if err != nil {
		return nil, fmt.Errorf("payments: error retrieving PaymentMethod: %v", err)
	}

	return pm, nil
}

// UpdateMethod updates a PaymentMethod object.
// A PaymentMethod must be attached a customer to be updated.
func UpdateMethod(methodID string) (*stripe.PaymentMethod, error) {
	params := &stripe.PaymentMethodParams{}

	pm, err := paymentmethod.Update(methodID, params)
	if err != nil {
		return nil, fmt.Errorf("payments: error updating PaymentMethod: %v", err)
	}

	return pm, nil
}
