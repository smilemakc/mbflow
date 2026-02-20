package grpc

import "context"

type contextKey string

const (
	ctxKeySystemKeyID  contextKey = "system_key_id"
	ctxKeyServiceName  contextKey = "service_name"
	ctxKeyUserID       contextKey = "user_id"
	ctxKeyImpersonated contextKey = "impersonated"
	ctxKeyIsAdmin      contextKey = "is_admin"
	ctxKeyAuthMethod   contextKey = "auth_method"
)

func ContextWithSystemKeyID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeySystemKeyID, id)
}

func SystemKeyIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeySystemKeyID).(string)
	return v, ok
}

func ContextWithServiceName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, ctxKeyServiceName, name)
}

func ServiceNameFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeyServiceName).(string)
	return v, ok
}

func ContextWithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyUserID, id)
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeyUserID).(string)
	return v, ok
}

func ContextWithImpersonated(ctx context.Context, impersonated bool) context.Context {
	return context.WithValue(ctx, ctxKeyImpersonated, impersonated)
}

func ImpersonatedFromContext(ctx context.Context) bool {
	v, _ := ctx.Value(ctxKeyImpersonated).(bool)
	return v
}

func ContextWithIsAdmin(ctx context.Context, isAdmin bool) context.Context {
	return context.WithValue(ctx, ctxKeyIsAdmin, isAdmin)
}

func ContextWithAuthMethod(ctx context.Context, method string) context.Context {
	return context.WithValue(ctx, ctxKeyAuthMethod, method)
}
