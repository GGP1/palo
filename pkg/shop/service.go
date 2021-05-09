package shop

import (
	"context"
	"time"

	"github.com/GGP1/adak/pkg/product"
	"github.com/GGP1/adak/pkg/review"
	"gopkg.in/guregu/null.v4/zero"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

// Service provides shop operations.
type Service interface {
	Create(ctx context.Context, shop Shop) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context) ([]Shop, error)
	GetByID(ctx context.Context, id string) (Shop, error)
	Search(ctx context.Context, query string) ([]Shop, error)
	Update(ctx context.Context, id string, shop UpdateShop) error
}

type service struct {
	db *sqlx.DB
	mc *memcache.Client
}

// NewService returns a new shop service.
func NewService(db *sqlx.DB, mc *memcache.Client) Service {
	return &service{db, mc}
}

// Create a shop.
func (s *service) Create(ctx context.Context, shop Shop) error {
	sQuery := `INSERT INTO shops
	(id, name, created_at)
	VALUES ($1, $2, $3)`

	_, err := s.db.ExecContext(ctx, sQuery, shop.ID, shop.Name, time.Now())
	if err != nil {
		return errors.Wrap(err, "couldn't create the shop")
	}

	lQuery := `INSERT INTO locations
	(shop_id, country, state, zip_code, city, address)
	VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = s.db.ExecContext(ctx, lQuery, shop.ID, shop.Location.Country, shop.Location.State,
		shop.Location.ZipCode, shop.Location.City, shop.Location.Address)
	if err != nil {
		return errors.Wrap(err, "couldn't create the location")
	}

	return nil
}

// Delete permanently deletes a shop from the database.
func (s *service) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM shops WHERE id=$1", id)
	if err != nil {
		return errors.Wrap(err, "deleting shop from database")
	}

	if err := s.mc.Delete(id); err != nil && err != memcache.ErrCacheMiss {
		return errors.Wrap(err, "deleting shop from cache")
	}

	return nil
}

// Get returns a list with all the shops stored in the database.
func (s *service) Get(ctx context.Context) ([]Shop, error) {
	var shops []Shop

	if err := s.db.SelectContext(ctx, &shops, "SELECT * FROM shops"); err != nil {
		return nil, errors.Wrap(err, "couldn't find the shops")
	}

	return shops, nil
}

// GetByID retrieves the shop requested from the database.
func (s *service) GetByID(ctx context.Context, id string) (Shop, error) {
	q := `SELECT s.*, l.*, r.*, p.* 
	FROM shops s
	LEFT JOIN locations l ON s.id=l.shop_id
	LEFT JOIN reviews r ON s.id=r.shop_id
	LEFT JOIN products p ON s.id=p.shop_id
	WHERE s.id=$1`

	rows, err := s.db.QueryContext(ctx, q, id)
	if err != nil {
		return Shop{}, errors.Wrap(err, "fetching shop")
	}
	defer rows.Close()

	var shop Shop
	for rows.Next() {
		l := Location{}
		r := review.Review{}
		p := product.Product{}
		err := rows.Scan(
			&shop.ID, &shop.Name, &shop.CreatedAt, &shop.UpdatedAt,
			&l.ShopID, &l.Country, &l.State, &l.ZipCode, &l.City, &l.Address,
			&r.ID, &r.Stars, &r.Comment, &r.UserID, &r.ProductID, &r.ShopID, &r.CreatedAt,
			&p.ID, &p.ShopID, &p.Stock, &p.Brand, &p.Category, &p.Type, &p.Description, &p.Weight,
			&p.Discount, &p.Taxes, &p.Subtotal, &p.Total, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return Shop{}, errors.Wrap(err, "couldn't scan shop")
		}

		shop.Location = l
		shop.Reviews = append(shop.Reviews, r)
		shop.Products = append(shop.Products, p)
	}

	return shop, nil
}

// Search looks for the shops that contain the value specified. (Only text fields)
func (s *service) Search(ctx context.Context, query string) ([]Shop, error) {
	var shops []Shop
	q := "SELECT * FROM shops WHERE to_tsvector(id || ' ' || name) @@ plainto_tsquery($1)"

	if err := s.db.SelectContext(ctx, &shops, q, query); err != nil {
		return nil, errors.Wrap(err, "couldn't find shops")
	}

	return shops, nil
}

// Update updates shop fields.
func (s *service) Update(ctx context.Context, id string, shop UpdateShop) error {
	q := "UPDATE shops SET name=$2, updated_at=$3 WHERE id=$1"
	_, err := s.db.ExecContext(ctx, q, id, shop.Name, zero.TimeFrom(time.Now()))
	if err != nil {
		return errors.Wrap(err, "couldn't update the shop")
	}

	if err := s.mc.Delete(id); err != nil && err != memcache.ErrCacheMiss {
		return errors.Wrap(err, "couldn't delete shop from cache")
	}

	return nil
}

// TODO: finish and add to Service interface
func (s *service) UpdateLocation(ctx context.Context, shopID string, location Location) error {
	return nil
}
