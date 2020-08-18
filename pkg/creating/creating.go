/*
Package creating includes database creating operations
*/
package creating

import (
	"errors"
	"fmt"

	"github.com/GGP1/palo/internal/uuid"
	"github.com/GGP1/palo/pkg/model"
	"github.com/GGP1/palo/pkg/shopping"

	"github.com/jinzhu/gorm"
	"golang.org/x/crypto/bcrypt"
)

// Repository provides access to the storage.
type Repository interface {
	CreateProduct(db *gorm.DB, product *model.Product) error
	CreateReview(db *gorm.DB, review *model.Review) error
	CreateShop(db *gorm.DB, shop *model.Shop) error
	CreateUser(db *gorm.DB, user *model.User) error
}

// Service provides models adding operations.
type Service interface {
	CreateProduct(db *gorm.DB, product *model.Product) error
	CreateReview(db *gorm.DB, review *model.Review) error
	CreateShop(db *gorm.DB, shop *model.Shop) error
	CreateUser(db *gorm.DB, user *model.User) error
}

type service struct {
	r Repository
}

// NewService creates a deleting service with the necessary dependencies.
func NewService(r Repository) Service {
	return &service{r}
}

// CreateProduct validates a product and saves it into the database.
func (s *service) CreateProduct(db *gorm.DB, product *model.Product) error {
	if err := product.Validate(); err != nil {
		return err
	}

	taxes := ((product.Subtotal / 100) * product.Taxes)
	discount := ((product.Subtotal / 100) * product.Discount)
	product.Total = product.Subtotal + taxes - discount

	if err := db.Create(product).Error; err != nil {
		return fmt.Errorf("couldn't create the product: %v", err)
	}

	return nil
}

// CreateReview takes a new review and saves it into the database.
func (s *service) CreateReview(db *gorm.DB, review *model.Review) error {
	if err := db.Create(review).Error; err != nil {
		return fmt.Errorf("couldn't create the review: %v", err)
	}

	return nil
}

// CreateShop validates a shop and saves it into the database.
func (s *service) CreateShop(db *gorm.DB, shop *model.Shop) error {
	if err := shop.Validate(); err != nil {
		return err
	}

	if err := db.Create(shop).Error; err != nil {
		return fmt.Errorf("couldn't create the shop: %v", err)
	}

	return nil
}

// CreateUser validates a user, hashes its password, sends
// a verification email and saves it into the database.
func (s *service) CreateUser(db *gorm.DB, user *model.User) error {
	if err := user.Validate(""); err != nil {
		return err
	}

	rowsAffected := db.Where("email = ?", user.Email).First(&user).RowsAffected
	if rowsAffected != 0 {
		return errors.New("email is already taken")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hash)

	// Create a cart for each user
	id := uuid.GenerateRandRunes(24)
	user.CartID = id

	cart := shopping.NewCart(user.CartID)

	if err := db.Create(cart).Error; err != nil {
		return fmt.Errorf("couldn't create the cart: %v", err)
	}

	if err := db.Create(user).Error; err != nil {
		return fmt.Errorf("couldn't create the user: %v", err)
	}

	return nil
}
