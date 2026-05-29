package models

import "context"

// authUserCtxKey is the context key for the authenticated user.
type authUserCtxKey struct{}

// WithAuthUser returns a new context that contains the given AuthUser.
func WithAuthUser(ctx context.Context, u *AuthUser) context.Context {
	return context.WithValue(ctx, authUserCtxKey{}, u)
}

// AuthUserFromContext returns the AuthUser stored in the context, or nil.
func AuthUserFromContext(ctx context.Context) *AuthUser {
	if v := ctx.Value(authUserCtxKey{}); v != nil {
		if u, ok := v.(*AuthUser); ok {
			return u
		}
	}
	return nil
}

// AuthUserNameFromContext returns the authenticated user's name, or empty string.
func AuthUserNameFromContext(ctx context.Context) string {
	if u := AuthUserFromContext(ctx); u != nil && u.Name != "" {
		return u.Name
	}
	return ""
}
