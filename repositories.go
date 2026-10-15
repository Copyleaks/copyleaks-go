package copyleaks

import (
	"context"
	"net/http"
)

// RepositoryRole is the role of a user in a Private Cloud Hub.
// See https://docs.copyleaks.com/reference/data-types/authenticity/private-cloud-hub/roles.
type RepositoryRole int

const (
	// RepositoryRoleViewer can see part of the hub metadata and scan against the hub.
	RepositoryRoleViewer RepositoryRole = 1
	// RepositoryRoleContributor can also add documents, and delete their own.
	RepositoryRoleContributor RepositoryRole = 2
	// RepositoryRoleAdmin can also invite users and delete any document.
	RepositoryRoleAdmin RepositoryRole = 3
	// RepositoryRoleSuperAdmin can also delete the hub and manage its billing.
	RepositoryRoleSuperAdmin RepositoryRole = 4
)

// RepositoryStatus is the state of a Private Cloud Hub.
// See https://docs.copyleaks.com/reference/data-types/authenticity/private-cloud-hub/status.
type RepositoryStatus int

const (
	// RepositoryStatusRunning is a healthy hub.
	RepositoryStatusRunning RepositoryStatus = 0
	// RepositoryStatusPendingForDeletion is a hub about to be deleted with all its documents; it cannot be used.
	RepositoryStatusPendingForDeletion RepositoryStatus = 1
	// RepositoryStatusError is a hub that is not healthy. Contact Copyleaks support.
	RepositoryStatusError RepositoryStatus = 2
	// RepositoryStatusUpdating is a hub under maintenance. It may not be available.
	RepositoryStatusUpdating RepositoryStatus = 3
	// RepositoryStatusPaymentRequired is a hub suspended for a missing or failed payment.
	RepositoryStatusPaymentRequired RepositoryStatus = 4
)

// RepositoryInfo is information about a Private Cloud Hub.
type RepositoryInfo struct {
	// ID is the repository ID.
	ID string `json:"id"`
	// Name is the name of the hub.
	Name string `json:"name"`
	// Description is the description of the hub.
	Description string `json:"description"`
	// Permission is the role of the caller in the hub.
	Permission RepositoryRole `json:"permission"`
	// Status is the state of the hub.
	Status RepositoryStatus `json:"status"`
	// MaxCredits is the credit limit of the hub.
	MaxCredits int `json:"maxCredits"`
	// CurrentCredits is the number of credits that the hub has.
	CurrentCredits int `json:"currentCredits"`
	// MaskingPolicy sets who sees the content of the hub's documents in their results.
	MaskingPolicy MaskingPolicy `json:"maskingPolicy"`
	// CreationTime is when the hub was created, in ISO 8601 format.
	CreationTime string `json:"creationTime"`
}

// RepositoryInfo returns the credits, metadata and status of a Private Cloud Hub. It needs the Admin or Super
// Admin role; the repository ID is in the Copyleaks Admin Dashboard.
// See https://docs.copyleaks.com/reference/actions/private-cloud-hub/info.
func (c *Client) RepositoryInfo(ctx context.Context, token *AuthToken, repositoryID string) (*RepositoryInfo, error) {
	if repositoryID == "" {
		return nil, invalid("repository ID is required")
	}
	if err := VerifyAuthToken(token); err != nil {
		return nil, err
	}
	var res RepositoryInfo
	err := c.do(ctx, apiRequest{
		method: http.MethodGet,
		path:   "/v3/repositories/repository/" + segment(repositoryID) + "/info",
		token:  token,
	}, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
