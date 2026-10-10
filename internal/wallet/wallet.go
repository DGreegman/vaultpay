package wallet 

import "github.com/google/uuid"


type Currency string 

const (
	CurrencyNGN Currency = "NGN"
	CurrencyUSD Currency = "USD"
)

type Status string 

const (
	StatusActive Status = "active"
	StatusFrozen Status = "frozen"
	StatusClosed Status = "closed"
)

type Wallet struct {
	ID	   		uuid.UUID 
	UserID 		uuid.UUID
	Currency 	Currency
	Status  	Status
}