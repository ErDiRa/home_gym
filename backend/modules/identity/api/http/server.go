package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/erdira/homegym/backend/common/auth"
	"github.com/erdira/homegym/backend/common/errs"
	"github.com/erdira/homegym/backend/common/log"
	"github.com/erdira/homegym/backend/common/web"
	"github.com/erdira/homegym/backend/modules/identity/app"
)

// Server implements the identity part of the API (StrictServerInterface).
type Server struct {
	service *app.Service
	// secureCookies marks the session cookie Secure (HTTPS only); off for local HTTP dev.
	secureCookies bool
}

var _ StrictServerInterface = (*Server)(nil)

func NewServer(service *app.Service, secureCookies bool) *Server {
	if service == nil {
		panic("service cannot be nil")
	}
	return &Server{service: service, secureCookies: secureCookies}
}

// Register mounts the routes on r (the /api router). They are public: login and logout need no
// session, and me checks the cookie itself.
func Register(r chi.Router, s *Server) {
	r.Group(func(r chi.Router) {
		r.Use(web.LimitBody(web.MaxBody))
		strict := NewStrictHandlerWithOptions(s, nil, StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  web.BodyError,
			ResponseErrorHandlerFunc: web.ServiceError,
		})
		HandlerWithOptions(strict, ChiServerOptions{BaseRouter: r, ErrorHandlerFunc: paramError})
	})
}

func paramError(w http.ResponseWriter, r *http.Request, err error) {
	web.ClientError(w, r, http.StatusBadRequest, "invalid request", err)
}

// Login checks the credentials and starts a new session.
func (s *Server) Login(ctx context.Context, req LoginRequestObject) (LoginResponseObject, error) {
	remember := req.Body.Remember != nil && *req.Body.Remember
	session, err := s.service.Login(ctx, req.Body.Email, req.Body.Password, remember)
	if err != nil {
		return nil, err
	}
	cookie := s.sessionCookie(session)
	return Login200JSONResponse{Body: accountToAPI(session.Account), Headers: Login200ResponseHeaders{SetCookie: &cookie}}, nil
}

// Logout clears the cookie. The JWT itself stays valid until it expires (there is no server-side
// session to revoke), but the browser no longer has it.
func (s *Server) Logout(context.Context, LogoutRequestObject) (LogoutResponseObject, error) {
	cleared := s.clearedCookie()
	return Logout204Response{Headers: Logout204ResponseHeaders{SetCookie: &cleared}}, nil
}

// Me returns the signed-in user and extends the session (up to app.MaxSessionAge after login).
// The frontend uses it to restore the session since it cannot read the HttpOnly cookie.
func (s *Server) Me(ctx context.Context, req MeRequestObject) (MeResponseObject, error) {
	var token string
	if req.Params.Token != nil {
		token = *req.Params.Token
	}
	session, err := s.service.Refresh(ctx, token)
	// The account was deleted since login: end the session instead of extending it.
	var e *errs.Error
	if errors.As(err, &e) && e.Kind == errs.KindNotFound {
		log.ContextError(ctx, http.StatusUnauthorized, e.Msg, e.Err)
		cleared := s.clearedCookie()
		return Me401JSONResponse{Body: Error{Error: e.Msg}, Headers: Me401ResponseHeaders{SetCookie: &cleared}}, nil
	}
	if err != nil {
		return nil, err
	}
	cookie := s.sessionCookie(session)
	return Me200JSONResponse{Body: accountToAPI(session.Account), Headers: Me200ResponseHeaders{SetCookie: &cookie}}, nil
}

// sessionCookie is the session's token as an HttpOnly cookie (a Set-Cookie value): with
// "remember me" it expires with the token, otherwise the browser drops it when it closes.
func (s *Server) sessionCookie(session app.Session) string {
	c := s.cookie(session.Token)
	if session.Remember {
		c.MaxAge = int(time.Until(session.Expiry).Seconds())
	}
	return c.String()
}

// clearedCookie deletes the session cookie.
func (s *Server) clearedCookie() string {
	c := s.cookie("")
	c.MaxAge = -1
	return c.String()
}

// cookie is the session cookie's attributes; clearing it needs the same ones, or the browser
// treats it as a different cookie and keeps the old one.
func (s *Server) cookie(value string) *http.Cookie {
	return &http.Cookie{
		Name: auth.CookieName, Value: value, Path: "/",
		HttpOnly: true, Secure: s.secureCookies, SameSite: http.SameSiteLaxMode,
	}
}

func accountToAPI(a app.Account) Account {
	return Account{Id: a.ID, Email: a.Email}
}
