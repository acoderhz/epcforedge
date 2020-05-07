// SPDX-License-Identifier: Apache-2.0
// Copyright © 2019-2020 Intel Corporation

package af

import (
	"context"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// DefaultNotifURL const
const DefaultNotifURL = "/af/v1/notifications"

type keyType string

// Route struct
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

// Routes type
type Routes []Route

// NewAFRouter function
func NewAFRouter(afCtx *Context) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	for _, route := range afRoutes {
		var handler http.Handler = route.HandlerFunc
		handler = afLogger(handler, route.Name)

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(
				r.Context(),
				keyType("af-ctx"),
				afCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	return router
}

// NewNotifRouter function
func NewNotifRouter(afCtx *Context) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	for _, route := range notifRoutes {
		var handler http.Handler = route.HandlerFunc
		handler = afLogger(handler, route.Name)

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(
				r.Context(),
				keyType("af-ctx"),
				afCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	return router
}

var notifRoutes = Routes{
	Route{
		"NotificationPost",
		strings.ToUpper("Post"),
		DefaultNotifURL,
		NotificationPost,
	},

	// Policy auth notification routes
	Route{
		"PolicyAuthEventNotify",
		strings.ToUpper("Post"),
		"/af/v1/policy-authorization/notify",
		PolicyAuthEventNotify,
	},
	Route{
		"PolicyAuthEventNotifTerminate",
		strings.ToUpper("Post"),
		"/af/v1/policy-authorization/terminate",
		PolicyAuthEventNotifTerminate,
	},
}

var afRoutes = Routes{

	Route{
		"GetAllSubscriptions",
		strings.ToUpper("Get"),
		"/af/v1/subscriptions",
		GetAllSubscriptions,
	},

	Route{
		"GetSubscription",
		strings.ToUpper("Get"),
		"/af/v1/subscriptions/{subscriptionId}",
		GetSubscription,
	},

	Route{
		"DeleteSubscription",
		strings.ToUpper("Delete"),
		"/af/v1/subscriptions/{subscriptionId}",
		DeleteSubscription,
	},

	Route{
		"SubscriptionPatch",
		strings.ToUpper("Patch"),
		"/af/v1/subscriptions/{subscriptionId}",
		ModifySubscriptionPatch,
	},

	Route{
		"CreateSubscription",
		strings.ToUpper("Post"),
		"/af/v1/subscriptions",
		CreateSubscription,
	},

	Route{
		"SubscriptionPut",
		strings.ToUpper("Put"),
		"/af/v1/subscriptions/{subscriptionId}",
		ModifySubscriptionPut,
	},

	// PFD Management routes

	Route{
		"GetAllPfdTransactions",
		strings.ToUpper("Get"),
		"/af/v1/pfd/transactions",
		GetAllPfdTransactions,
	},

	Route{
		"CreatePfdTransaction",
		strings.ToUpper("Post"),
		"/af/v1/pfd/transactions",
		CreatePfdTransaction,
	},

	Route{
		"GetPfdTransaction",
		strings.ToUpper("Get"),
		"/af/v1/pfd/transactions/{transactionId}",
		GetPfdTransaction,
	},

	Route{
		"DeletePfdTransaction",
		strings.ToUpper("Delete"),
		"/af/v1/pfd/transactions/{transactionId}",
		DeletePfdTransaction,
	},

	Route{
		"PutPfdTransaction",
		strings.ToUpper("Put"),
		"/af/v1/pfd/transactions/{transactionId}",
		PutPfdTransaction,
	},

	Route{
		"GetPfdAppTransaction",
		strings.ToUpper("Get"),
		"/af/v1/pfd/transactions/{transactionId}/applications/{appId}",
		GetPfdAppTransaction,
	},

	Route{
		"DeletePfdAppTransaction",
		strings.ToUpper("Delete"),
		"/af/v1/pfd/transactions/{transactionId}/applications/{appId}",
		DeletePfdAppTransaction,
	},

	Route{
		"PutPfdAppTransaction",
		strings.ToUpper("Put"),
		"/af/v1/pfd/transactions/{transactionId}/applications/{appId}",
		PutPfdAppTransaction,
	},

	Route{
		"PatchPfdAppTransaction",
		strings.ToUpper("Patch"),
		"/af/v1/pfd/transactions/{transactionId}/applications/{appId}",
		PatchPfdAppTransaction,
	},

	// Policy Authorization routes

	Route{
		"CreatePolicyAuthAppSessions",
		strings.ToUpper("Post"),
		"/af/v1/policy-authorization/app-sessions",
		CreatePolicyAuthAppSessions,
	},

	Route{
		"GetPolicyAuthAppSession",
		strings.ToUpper("Get"),
		"/af/v1/policy-authorization/app-sessions/{appSessionId}",
		GetPolicyAuthAppSession,
	},

	Route{
		"ModifyPolicyAuthAppSession",
		strings.ToUpper("Patch"),
		"/af/v1/policy-authorization/app-sessions/{appSessionId}",
		ModifyPolicyAuthAppSession,
	},

	Route{
		"DeletePolicyAuthAppSession",
		strings.ToUpper("Post"),
		"/af/v1/policy-authorization/app-sessions/{appSessionId}/delete",
		DeletePolicyAuthAppSession,
	},

	Route{
		"PolicyAuthEventSubsc",
		strings.ToUpper("Put"),
		"/af/v1/policy-authorization/app-sessions/{appSessionId}/events-subscription",
		PolicyAuthEventSubsc,
	},

	Route{
		"PolicyAuthEventDelete",
		strings.ToUpper("Delete"),
		"/af/v1/policy-authorization/app-sessions/{appSessionId}/events-subscription",
		PolicyAuthEventDelete,
	},
}
