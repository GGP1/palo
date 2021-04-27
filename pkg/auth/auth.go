// Package auth provides authentication and authorization support.
package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/GGP1/adak/internal/token"
	"github.com/GGP1/adak/pkg/user"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

var errInvalidEmailOrPwd = errors.New("invalid email or password")

// Session implements session interface.
type Session struct {
	sync.RWMutex

	db         *sqlx.DB
	userClient user.UsersClient

	// user session
	store map[string]userData
	// last time the user logged in
	cleaned time.Time
	// session length
	length int
	// number of tries to log in
	tries map[string][]struct{}
	// time to wait after failing x times (increments every fail)
	delay map[string]time.Time
}

type userData struct {
	lastSeen time.Time
}

// NewSession returns a new session server.
func NewSession(db *sqlx.DB, userConn *grpc.ClientConn) *Session {
	return &Session{
		db:         db,
		userClient: user.NewUsersClient(userConn),
		store:      make(map[string]userData),
		cleaned:    time.Now(),
		length:     0,
		tries:      make(map[string][]struct{}),
		delay:      make(map[string]time.Time),
	}
}

// Run starts the server.
func (s *Session) Run(port int) error {
	srv := grpc.NewServer()
	RegisterSessionServer(srv, s)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return errors.Wrapf(err, "session: failed listening on port %d", port)
	}

	return srv.Serve(lis)
}

// AlreadyLoggedIn checks if the user has an active session or not.
func (s *Session) AlreadyLoggedIn(ctx context.Context, req *AlreadyLoggedInRequest) (*AlreadyLoggedInResponse, error) {
	s.Lock()
	user, ok := s.store[req.SessionID]
	if ok {
		user.lastSeen = time.Now()
		s.store[req.SessionID] = user
	}
	s.Unlock()

	return &AlreadyLoggedInResponse{SessionLen: int64(s.length), LoggedIn: ok}, nil
}

// Login authenticates users.
func (s *Session) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	s.RLock()
	if !s.delay[req.Email].IsZero() && s.delay[req.Email].Sub(time.Now()) > 0 {
		return nil, fmt.Errorf("please wait %v before trying again", s.delay[req.Email].Sub(time.Now()))
	}
	s.RUnlock()

	// Check if the email exists and if it is verified
	u, err := s.userClient.GetByEmail(ctx, &user.GetByEmailRequest{Email: req.Email})
	if err != nil {
		s.loginDelay(req.Email)
		log.Debug().Err(err)
		return nil, errInvalidEmailOrPwd
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.User.Password), []byte(req.Password)); err != nil {
		s.loginDelay(req.Email)
		log.Debug().Err(err)
		return nil, errInvalidEmailOrPwd
	}

	// sessionID is used to add the user to the session map
	sID := token.GenerateRunes(27)

	// Set session data and delete tries and delay
	s.Lock()
	s.store[sID] = userData{time.Now()}
	delete(s.tries, req.Email)
	delete(s.delay, req.Email)
	s.Unlock()

	return &LoginResponse{
		UserID:     u.User.ID,
		SessionID:  sID,
		CartID:     u.User.CartID,
		SessionLen: int64(s.length),
	}, nil
}

// Logout removes the user session and its cookies.
func (s *Session) Logout(ctx context.Context, req *LogoutRequest) (*LogoutResponse, error) {
	s.Lock()
	defer s.Unlock()
	delete(s.store, req.SessionID)

	if time.Now().Sub(s.cleaned) > (time.Minute * 30) {
		s.clean()
	}

	return &LogoutResponse{}, nil
}

// clean deletes all the sessions that have expired.
func (s *Session) clean() {
	for key, value := range s.store {
		if time.Now().Sub(value.lastSeen) > (time.Hour * 168) {
			delete(s.store, key)
		}
	}
	s.cleaned = time.Now()
}

// loginDelay increments the time that the user will have to wait after failing.
func (s *Session) loginDelay(email string) {
	s.Lock()
	s.tries[email] = append(s.tries[email], struct{}{})
	d := (len(s.tries[email]) * 2)
	s.delay[email] = time.Now().Add(time.Second * time.Duration(d))
	s.Unlock()
}

// DeleteCookie removes a cookie.
func DeleteCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "0",
		Expires:  time.Unix(1414414788, 1414414788000),
		Path:     "/",
		Domain:   "localhost",
		Secure:   false,
		HttpOnly: true,
		MaxAge:   -1,
	})
}

// SetCookie creates a cookie.
func SetCookie(w http.ResponseWriter, name, value, path string, length int) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Domain:   "localhost",
		Secure:   false,
		HttpOnly: true,
		SameSite: 3,
		MaxAge:   length,
	})
}
