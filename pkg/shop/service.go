package shop

import (
	"context"
	"time"

	"github.com/GGP1/palo/internal/token"
	"github.com/GGP1/palo/pkg/product"
	"github.com/GGP1/palo/pkg/review"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

// Repository provides access to the storage.
type Repository interface {
	Create(ctx context.Context, shop *Shop) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context) ([]Shop, error)
	GetByID(ctx context.Context, id string) (Shop, error)
	Search(ctx context.Context, search string) ([]Shop, error)
	Update(ctx context.Context, shop *Shop, id string) error
}

// Service provides shop operations.
type Service interface {
	Create(ctx context.Context, shop *Shop) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context) ([]Shop, error)
	GetByID(ctx context.Context, id string) (Shop, error)
	Search(ctx context.Context, search string) ([]Shop, error)
	Update(ctx context.Context, shop *Shop, id string) error
}

type service struct {
	r  Repository
	DB *sqlx.DB
}

// NewService creates a deleting service with the necessary dependencies.
func NewService(r Repository, db *sqlx.DB) Service {
	return &service{r, db}
}

// Create creates a shop.
func (s *service) Create(ctx context.Context, shop *Shop) error {
	sQuery := `INSERT INTO shops
	(id, name, created_at, updated_at)
	VALUES ($1, $2, $3, $4)`

	lQuery := `INSERT INTO locations
	(shop_id, country, state, zip_code, city, address)
	VALUES ($1, $2, $3, $4, $5, $6)`

	if err := shop.Validate(); err != nil {
		return err
	}

	id := token.GenerateRunes(30)
	shop.CreatedAt = time.Now()

	_, err := s.DB.ExecContext(ctx, sQuery, id, shop.Name, shop.CreatedAt, shop.UpdatedAt)
	if err != nil {
		return errors.Wrap(err, "couldn't create the shop")
	}

	_, err = s.DB.ExecContext(ctx, lQuery, id, shop.Location.Country, shop.Location.State,
		shop.Location.ZipCode, shop.Location.City, shop.Location.Address)
	if err != nil {
		return errors.Wrap(err, "couldn't create the location")
	}

	return nil
}

// Delete permanently deletes a shop from the database.
func (s *service) Delete(ctx context.Context, id string) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM shops WHERE id=$1", id)
	if err != nil {
		return errors.Wrap(err, "couldn't delete the shop")
	}

	return nil
}

// Get returns a list with all the shops stored in the database.
func (s *service) Get(ctx context.Context) ([]Shop, error) {
	var (
		shops []Shop
		list  []Shop
	)

	ch := make(chan Shop)
	errCh := make(chan error)

	if err := s.DB.SelectContext(ctx, &shops, "SELECT * FROM shops"); err != nil {
		return nil, errors.Wrap(err, "shops not found")
	}

	for _, shop := range shops {
		go func(shop Shop) {
			var (
				location Location
				reviews  []review.Review
				products []product.Product
			)

			if err := s.DB.GetContext(ctx, &location, "SELECT * FROM locations WHERE shop_id=$1", shop.ID); err != nil {
				errCh <- errors.Wrap(err, "location not found")
			}

			if err := s.DB.SelectContext(ctx, &reviews, "SELECT * FROM reviews WHERE shop_id=$1", shop.ID); err != nil {
				errCh <- errors.Wrap(err, "reviews not found")
			}

			if err := s.DB.SelectContext(ctx, &products, "SELECT * FROM products WHERE shop_id=$1", shop.ID); err != nil {
				errCh <- errors.Wrap(err, "products not found")
			}

			shop.Location = location
			shop.Reviews = reviews
			shop.Products = products

			ch <- shop
		}(shop)
	}

	select {
	case s := <-ch:
		list = append(list, s)
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return list, nil
}

// GetByID retrieves the shop requested from the database.
func (s *service) GetByID(ctx context.Context, id string) (Shop, error) {
	var (
		shop     Shop
		location Location
		reviews  []review.Review
		products []product.Product
	)

	if err := s.DB.GetContext(ctx, &shop, "SELECT * FROM shops WHERE id=$1", id); err != nil {
		return Shop{}, errors.Wrap(err, "shop not found")
	}

	if err := s.DB.GetContext(ctx, &location, "SELECT * FROM locations WHERE shop_id=$1", id); err != nil {
		return Shop{}, errors.Wrap(err, "location not found")
	}

	if err := s.DB.SelectContext(ctx, &reviews, "SELECT * FROM reviews WHERE shop_id=$1", id); err != nil {
		return Shop{}, errors.Wrap(err, "reviews not found")
	}

	if err := s.DB.SelectContext(ctx, &products, "SELECT * FROM products WHERE shop_id=$1", id); err != nil {
		return Shop{}, errors.Wrap(err, "products not found")
	}

	shop.Location = location
	shop.Reviews = reviews
	shop.Products = products

	return shop, nil
}

// Search looks for the shops that contain the value specified. (Only text fields)
func (s *service) Search(ctx context.Context, search string) ([]Shop, error) {
	var (
		shops []Shop
		list  []Shop
	)

	ch := make(chan Shop)
	errCh := make(chan error)

	q := `SELECT * FROM shops WHERE
	to_tsvector(id || ' ' || name) @@ to_tsquery($1)`

	if err := s.DB.SelectContext(ctx, &shops, q, search); err != nil {
		return nil, errors.Wrap(err, "couldn't find shops")
	}

	for _, shop := range shops {
		go func(shop Shop) {
			var (
				location Location
				reviews  []review.Review
				products []product.Product
			)

			if err := s.DB.GetContext(ctx, &location, "SELECT * FROM locations WHERE shop_id=$1", shop.ID); err != nil {
				errCh <- errors.Wrap(err, "location not found")
			}

			if err := s.DB.SelectContext(ctx, &reviews, "SELECT * FROM reviews WHERE shop_id=$1", shop.ID); err != nil {
				errCh <- errors.Wrap(err, "reviews not found")
			}

			if err := s.DB.SelectContext(ctx, &products, "SELECT * FROM products WHERE shop_id=$1", shop.ID); err != nil {
				errCh <- errors.Wrap(err, "products not found")
			}

			shop.Location = location
			shop.Reviews = reviews
			shop.Products = products

			ch <- shop
		}(shop)
	}

	select {
	case s := <-ch:
		list = append(list, s)
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return list, nil
}

// Update updates shop fields.
func (s *service) Update(ctx context.Context, shop *Shop, id string) error {
	q := `UPDATE shops SET name=$2, country=$3, city=$4, address=$5
	WHERE id=$1`

	_, err := s.DB.ExecContext(ctx, q, id, shop.Name, shop.Location.Country,
		shop.Location.City, shop.Location.Address)
	if err != nil {
		return errors.Wrap(err, "couldn't update the shop")
	}

	return nil
}