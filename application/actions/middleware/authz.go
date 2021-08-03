package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gobuffalo/buffalo"
	"github.com/gofrs/uuid"

	"github.com/silinternational/riskman-api/domain"
	"github.com/silinternational/riskman-api/models"
)

var authableResources = map[string]models.Authable{
	domain.TypeUser: &models.User{},
}

func AuthZ(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		actor, ok := c.Value(domain.ContextKeyCurrentUser).(models.User)
		if !ok {
			return c.Error(http.StatusUnauthorized, fmt.Errorf("actor must be authenticated to proceed"))
		}

		rName, rID, rSub := getResourceIDSubresource(c.Request().URL.Path)

		var resource models.Authable
		var isAuthable bool
		if resource, isAuthable = authableResources[rName]; !isAuthable {
			return c.Error(http.StatusInternalServerError, fmt.Errorf("resource expected to be authable but isn't"))
		}

		if rID != uuid.Nil {
			tx := models.Tx(c)
			if tx == nil {
				return c.Error(http.StatusInternalServerError, fmt.Errorf("failed to intialize db connection"))
			}
			if err := resource.FindByID(tx, rID); err != nil {
				// TODO: this perhaps should return a 404, or just pass the error along based on api.AppError
				return c.Error(http.StatusInternalServerError, fmt.Errorf("failed to load resource: %s", err))
			}
		}

		var p models.Permission

		switch c.Request().Method {
		case http.MethodGet:
			p = models.PermissionList
			if rID != uuid.Nil {
				p = models.PermissionView
			}
		case http.MethodPost:
			p = models.PermissionCreate
		case http.MethodPut:
			p = models.PermissionUpdate
		case http.MethodDelete:
			p = models.PermissionDelete
		default:
			p = models.PermissionDenied
		}

		if !resource.IsActorAllowedTo(actor, p, rSub, limitedRequest(c.Request())) {
			return c.Error(http.StatusForbidden, fmt.Errorf("actor not allowed to perform that action on this resource"))
		}

		// put found resource into context if found
		if resource.GetID() != uuid.Nil {
			c.Set(rName, resource)
		}

		return next(c)
	}
}

// limitedRequest returns a new *http.Request with most information about the request, excluding
// Body and Forms that read from Body to ensure the Body content is still available for later processing
func limitedRequest(req *http.Request) *http.Request {
	return &http.Request{
		Method:           req.Method,
		URL:              req.URL,
		Proto:            req.Proto,
		ProtoMajor:       req.ProtoMajor,
		ProtoMinor:       req.ProtoMinor,
		Header:           req.Header,
		ContentLength:    req.ContentLength,
		TransferEncoding: req.TransferEncoding,
		Host:             req.Host,
		RemoteAddr:       req.RemoteAddr,
		RequestURI:       req.RequestURI,
	}
}

func getResourceIDSubresource(path string) (string, uuid.UUID, string) {
	resource, id, sub := "", uuid.Nil, ""

	if path == "" {
		return resource, id, sub
	}

	cleanPath := strings.TrimPrefix(path, "/")
	cleanPath = strings.TrimSuffix(cleanPath, "/")
	pathParts := strings.Split(cleanPath, "/")

	if len(pathParts) == 0 {
		return resource, id, sub
	}

	resource = pathParts[0]

	if len(pathParts) > 1 {
		id = uuid.FromStringOrNil(pathParts[1])
	}

	if len(pathParts) > 2 && id != uuid.Nil {
		sub = pathParts[2]
	}

	return resource, id, sub
}
